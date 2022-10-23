package postgresql

import (
	"context"
	"fmt"

	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/errors"

	authv1alpha1 "github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/common"
	"github.com/alex123012/database-users-operator/pkg/database"
	"github.com/alex123012/database-users-operator/pkg/utils"
	"github.com/go-logr/logr"
)

type processhandler = func(context.Context, bool) error

func NewPostgresFromConfig(config *authv1alpha1.Config, userResource *authv1alpha1.User, client common.KubeInterface, logger logr.Logger) common.DatabaseInterface {
	configResource := &config.Spec.PostgreSQL
	newConfig := NewPostgresConfig(fmt.Sprintf("%s.%s.svc.cluster.local", configResource.Host, configResource.Namespace),
		configResource.Port,
		configResource.User, "",
		configResource.DatabaseName, configResource.SSLMode, "", "", "")
	return NewPostgres(newConfig, configResource, userResource, client, logger)
}

type Postgres struct {
	conn             *database.DBconnector
	config           *PostgresConfig
	logger           logr.Logger
	client           common.KubeInterface
	configResource   *authv1alpha1.PostgreSQLConfig
	userResource     *authv1alpha1.User
	createCerts      bool
	postgresRootData map[string][]byte
}

func NewPostgres(config *PostgresConfig, configResource *authv1alpha1.PostgreSQLConfig, userResource *authv1alpha1.User, client common.KubeInterface, logger logr.Logger) *Postgres {
	return &Postgres{
		config:         config,
		logger:         logger,
		userResource:   userResource,
		configResource: configResource,
		client:         client,
	}
}
func (p *Postgres) Connect(ctx context.Context) error {
	switch p.config.SSLMode {
	case database.SSLModeVERIFYCA, database.SSLModeREQUIRE, database.SSLModeVERIFYFULL:
		postgresRootSecret, err := p.client.GetV1Secret(
			ctx,
			p.configResource.SSLCredentials.UserSecret.Name,
			p.configResource.SSLCredentials.UserSecret.Namespace,
			p.logger,
		)
		if err != nil {
			return err
		}
		p.config.SSLCACert = utils.FilePathFromHome("postgres-certs/ca.crt")
		p.config.SSLUserCert = utils.FilePathFromHome(fmt.Sprintf("postgres-certs/client.%s.crt", p.configResource.User))
		p.config.SSLUserKey = utils.FilePathFromHome(fmt.Sprintf("postgres-certs/client.%s.key", p.configResource.User))

		p.postgresRootData = postgresRootSecret.Data
		if err := utils.CreateFileFromBytes(p.config.SSLCACert, p.postgresRootData["ca.crt"]); err != nil {
			return err
		}

		if err := utils.CreateFileFromBytes(p.config.SSLUserCert, p.postgresRootData["tls.crt"]); err != nil {
			return err
		}

		if err := utils.CreateFileFromBytes(p.config.SSLUserKey, p.postgresRootData["tls.key"]); err != nil {
			return err
		}

		p.createCerts = true
	case database.SSLModeALLOW, database.SSLModeDISABLE, database.SSLModePREFER:
		passwordSecret, err := p.client.GetV1Secret(
			ctx,
			p.configResource.PasswordSecret.Name,
			p.configResource.PasswordSecret.Namespace,
			p.logger,
		)
		if err != nil {
			return err
		}
		p.config.Password = string(passwordSecret.Data["password"])
	default:
		return errors.NewBadRequest("No such SSLmode")
	}
	p.conn = database.NewDBConnector(p.config.connString(), database.DBDriverPostgres, p.logger)

	if err := p.conn.Connect(ctx); err != nil {
		return err
	}
	p.conn.MapperFunc("postgres", common.SimpleMapper)
	return nil
}

func (p *Postgres) Close(ctx context.Context) {
	p.conn.Close(ctx)
}

func (p *Postgres) ProcessUser(ctx context.Context) error {
	if err := p.Connect(ctx); err != nil {
		return err
	}
	defer utils.DeleteFile(p.config.SSLUserCert)
	defer utils.DeleteFile(p.config.SSLUserKey)
	defer utils.DeleteFile(p.config.SSLCACert)
	defer p.Close(ctx)

	errGroup, ctx := errgroup.WithContext(ctx)
	goroutinesList := []processhandler{p.createUser, p.processUserTablePrivileges, p.processDBUserRoles}
	for _, fn := range goroutinesList {
		tmpFn := fn
		errGroup.Go(func() error {
			return tmpFn(ctx, false)
		})
	}
	return errGroup.Wait()
}

func (p *Postgres) DeleteUser(ctx context.Context) error {
	if err := p.Connect(ctx); err != nil {
		return err
	}
	defer utils.DeleteFile(p.config.SSLUserCert)
	defer utils.DeleteFile(p.config.SSLUserKey)
	defer utils.DeleteFile(p.config.SSLCACert)
	defer p.Close(ctx)

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	goroutinesList := []processhandler{p.processUserTablePrivileges, p.processDBUserRoles}
	for _, fn := range goroutinesList {
		tmpFn := fn
		errGroup.Go(func() error {
			return tmpFn(errGroupCtx, true)
		})
	}
	if err := errGroup.Wait(); err != nil {
		return err
	}
	return p.deleteUser(ctx)
}

func (p *Postgres) createUser(ctx context.Context, onDelete bool) error {
	sqlCreateUser := fmt.Sprintf(`CREATE USER %s`, EscapeLiteral(p.userResource.GetName()))
	if p.userResource.Spec.PasswordSecret != (authv1alpha1.Secret{}) {
		passwordSecret, err := p.client.GetV1Secret(
			ctx,
			p.userResource.Spec.PasswordSecret.Name,
			p.userResource.Spec.PasswordSecret.Namespace,
			p.logger,
		)
		if err != nil {
			return err
		}
		sqlCreateUser += fmt.Sprintf(" WITH PASSWORD %s", EscapeString(string(passwordSecret.Data["password"])))
	}

	if p.createCerts {
		p.generateCertSecretForUser(ctx)
	}
	return IgnoreAlreadyExists(p.conn.Exec(ctx, sqlCreateUser, database.DisableLogger))
}

func (p *Postgres) deleteUser(ctx context.Context) error {
	sqlDeleteUser := fmt.Sprintf(`DROP USER %s`, EscapeLiteral(p.userResource.GetName()))
	if p.createCerts {
		p.deleteCertSecretForUser(ctx)
	}
	return p.conn.Exec(ctx, sqlDeleteUser, database.DisableLogger)
}
func (p *Postgres) processUserTablePrivileges(ctx context.Context, onDelete bool) error {
	privByDBMap := make(map[string]map[authv1alpha1.Privilege]struct{})
	for _, priv := range p.userResource.Spec.Privileges {
		if priv.On != "" && priv.Database != "" {
			if _, f := privByDBMap[priv.Database]; !f {
				privByDBMap[priv.Database] = make(map[authv1alpha1.Privilege]struct{})
			}
			privByDBMap[priv.Database][priv] = struct{}{}
		}
	}

	dbList, err := p.getAllDatabases(ctx)
	if err != nil {
		return err
	}

	for _, dbname := range dbList {
		if err := p.processUserTablePrivilegesFromDB(ctx, dbname, privByDBMap[dbname], onDelete); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) processDBUserRoles(ctx context.Context, onDelete bool) error {

	dbPrivsMap, err := p.getDBUserRoles(ctx)
	if err != nil {
		return err
	}
	if err := p.revokeNotDefinedAndAssignDefinedRoles(ctx, dbPrivsMap, onDelete); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) processUserTablePrivilegesFromDB(ctx context.Context, dbname string, privMap map[authv1alpha1.Privilege]struct{}, onDelete bool) error {
	newconf := p.config.Copy()
	newconf.Dbname = dbname
	conn := NewPostgres(newconf, p.configResource, p.userResource, p.client, p.logger)

	if err := conn.Connect(ctx); err != nil {
		return err
	}
	defer conn.Close(ctx)

	var tablePrivs []authv1alpha1.Privilege
	err := conn.conn.Select(ctx, &tablePrivs,
		`SELECT privilege_type,
				table_catalog,
				table_name
		FROM information_schema.role_table_grants
		WHERE grantee = $1`,
		p.userResource.GetName())
	if err != nil {
		return err
	}
	p.logger.Info("DB user table privs", "TABLE_PRIVILEGES", tablePrivs)
	tablePrivsMap := make(map[authv1alpha1.Privilege]struct{})
	for _, priv := range tablePrivs {
		tablePrivsMap[priv] = struct{}{}
	}

	return p.revokeNotDefinedAndAssignDefined(ctx, conn.conn, tablePrivsMap, privMap, onDelete)
}

func (p *Postgres) getDBUserRoles(ctx context.Context) (map[authv1alpha1.Privilege]struct{}, error) {
	rolesList, err := p.userRolesQuery(ctx)
	if err != nil {
		return nil, err
	}
	dbPrivsList, err := p.userDBPrivsQuery(ctx)
	if err != nil {
		return nil, err
	}

	roles := make(map[authv1alpha1.Privilege]struct{})
	for _, role := range append(rolesList, dbPrivsList...) {
		roles[role] = struct{}{}
	}
	p.logger.Info(fmt.Sprintf("Getted roles for DB user %s: %v", p.userResource.GetName(), roles))
	return roles, nil
}

func (p *Postgres) userRolesQuery(ctx context.Context) ([]authv1alpha1.Privilege, error) {
	var rolesList []authv1alpha1.Privilege

	queryTemplate := `
	SELECT b.rolname as privilege_type
	FROM pg_catalog.pg_auth_members m
	JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
	WHERE m.member =
		(SELECT oid
		 FROM pg_catalog.pg_roles r
		 WHERE r.rolname = $1)`

	err := p.conn.Select(ctx, &rolesList, queryTemplate, p.userResource.GetName())
	if err != nil {
		return nil, err
	}
	return rolesList, nil
}

func (p *Postgres) userDBPrivsQuery(ctx context.Context) ([]authv1alpha1.Privilege, error) {
	var privList []authv1alpha1.Privilege

	queryTemplate := `
	WITH r AS
		(SELECT datname, (aclexplode(datacl)).grantee,
				(aclexplode(datacl)).privilege_type AS privs
		FROM pg_catalog.pg_database r)
	SELECT datname as table_catalog,
			privs as privilege_type
	FROM r
	WHERE r.grantee =
		(SELECT oid
		FROM pg_catalog.pg_roles r
		WHERE r.rolname = $1)`

	err := p.conn.Select(ctx, &privList, queryTemplate, p.userResource.GetName())
	if err != nil {
		return nil, err
	}
	return privList, nil
}

func (p *Postgres) revokeNotDefinedAndAssignDefinedRoles(ctx context.Context, dbPrivsMap map[authv1alpha1.Privilege]struct{}, onDelete bool) error {
	privMap := make(map[authv1alpha1.Privilege]struct{})
	for _, priv := range p.userResource.Spec.Privileges {
		if priv.On == "" {
			privMap[priv] = struct{}{}
		}
	}
	return p.revokeNotDefinedAndAssignDefined(ctx, p.conn, dbPrivsMap, privMap, onDelete)
}

func (p *Postgres) revokeNotDefinedAndAssignDefined(ctx context.Context, conn *database.DBconnector, dbPrivsMap, privMap map[authv1alpha1.Privilege]struct{}, onDelete bool) error {
	var queryList []database.Query
	userEsc := EscapeLiteral(p.userResource.GetName())

	if !onDelete {
		toCreate, toRevoke := IntersectDefinedPrivsWithDB(privMap, dbPrivsMap)
		revokeQueryList := getQueryListFromPrivsList([]string{"REVOKE %s", "FROM %s"}, toRevoke, userEsc)
		assignQueryList := getQueryListFromPrivsList([]string{"GRANT %s", "TO %s"}, toCreate, userEsc)
		queryList = append(revokeQueryList, assignQueryList...)
	} else {
		revokePrivList := make([]authv1alpha1.Privilege, len(privMap))
		i := 0
		for key := range privMap {
			revokePrivList[i] = key
			i++
		}
		queryList = getQueryListFromPrivsList([]string{"REVOKE %s", "FROM %s"}, revokePrivList, userEsc)
	}
	if len(queryList) > 0 {
		p.logger.Info("QUERY LIST for execute in transaction", "QUERY_LIST", queryList)
		if err := conn.ExecTx(ctx, queryList, []database.NamedQuery{}); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) getAllDatabases(ctx context.Context) ([]string, error) {
	databases := make([]string, 0)
	err := p.conn.Select(ctx, &databases, "SELECT datname FROM pg_database WHERE datistemplate = 'false' AND datname != 'postgres';")
	if err != nil {
		return nil, err
	}
	p.logger.Info("Getted databases", "DATABASES", databases)
	return databases, nil
}

func (p *Postgres) generateCertSecretForUser(ctx context.Context) error {

	if _, err := p.client.GetV1Secret(ctx, p.userResource.GetName(), p.userResource.GetNamespace(), p.logger); errors.IsNotFound(err) {
		p.logger.Info("Generating certificates for User")
		certData, err := p.generateDBCertificatesForUser(ctx)
		if err != nil {
			p.logger.Error(err, "Failed to generate new certificates for User '"+p.userResource.GetName()+"' in namespace '"+p.userResource.GetNamespace()+"'")
			return err
		}

		if err := p.client.CreateV1Secret(ctx, p.userResource, certData, p.logger); err != nil {
			p.logger.Error(err, "Failed to create new v1.Secret")
			return err
		}
		p.logger.Info("Successfully generated certificates for DB User")
		return nil
	} else if err != nil {
		return err
	}
	p.logger.Info("Certificates already generated")
	return nil
}

func (p *Postgres) deleteCertSecretForUser(ctx context.Context) error {
	secretResource, err := p.client.GetV1Secret(ctx, p.userResource.GetName(), p.userResource.GetNamespace(), p.logger)
	if errors.IsNotFound(err) {
		p.logger.Info("Certificates secret for DB user already deleted")
		return err
	} else if err != nil {
		return err
	}

	if err := p.client.DeleteV1Secret(ctx, secretResource, p.logger); err != nil {
		p.logger.Error(err, "Failed to delete v1.Secret for DB user")
		return err
	}
	p.logger.Info("Successfully deleted certificates secret for DB User")
	return nil

}

func (p *Postgres) generateDBCertificatesForUser(ctx context.Context) (map[string][]byte, error) {
	postgresCAKeySecret, err := p.client.GetV1Secret(ctx, p.configResource.SSLCredentials.CASecret.Name, p.configResource.SSLCredentials.CASecret.Namespace, p.logger)
	if err != nil {
		return nil, err
	}
	maps.Copy(postgresCAKeySecret.Data, p.postgresRootData)
	return GenPostgresCertFromCA(p.userResource.GetName(), postgresCAKeySecret.Data)
}
