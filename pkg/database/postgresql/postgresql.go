package postgresql

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	authv1alpha1 "github.com/alex123012/k8s-database-users-operator/api/v1alpha1"
	"github.com/alex123012/k8s-database-users-operator/pkg/common"
	"github.com/alex123012/k8s-database-users-operator/pkg/database"
	"github.com/alex123012/k8s-database-users-operator/pkg/utils"
	"github.com/go-logr/logr"
)

func NewPostgresFromConfig(config *authv1alpha1.Config, userResource *authv1alpha1.User, client common.KubeInterface, logger logr.Logger) common.DatabaseInterface {
	configResource := &config.Spec.PostgreSQL
	newConfig := NewPostgresConfig(fmt.Sprintf("%s.%s.svc.cluster.local", configResource.Host, configResource.Namespace),
		configResource.Port,
		configResource.User, "",
		configResource.DatabaseName, configResource.SSLMode, "", "", "")
	return NewPostgres(newConfig, configResource, userResource, client, logger)
}

type Postgres struct {
	conn           *database.DBconnector
	config         *PostgresConfig
	logger         logr.Logger
	client         common.KubeInterface
	configResource *authv1alpha1.PostgreSQLConfig
	userResource   *authv1alpha1.User
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

		secretMap := postgresRootSecret.Data
		if err := utils.CreateFileFromBytes(p.config.SSLCACert, secretMap["ca.crt"]); err != nil {
			return err
		}
		// defer utils.DeleteFile(caCert)

		if err := utils.CreateFileFromBytes(p.config.SSLUserCert, secretMap["tls.crt"]); err != nil {
			return err
		}
		// defer utils.DeleteFile(clientCert)

		if err := utils.CreateFileFromBytes(p.config.SSLUserKey, secretMap["tls.key"]); err != nil {
			return err
		}
		// defer utils.DeleteFile(clientKey)

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
	defer p.Close(ctx)
	if err := p.createUser(ctx); err != nil {
		return err
	}
	if err := p.updatePrivileges(ctx); err != nil {
		return err
	}

	return nil
}

func (p *Postgres) DeleteUser(ctx context.Context) error {
	return nil
}
func (p *Postgres) createUser(ctx context.Context) error {
	sqlCreateUser := fmt.Sprintf(`CREATE USER %s`, EscapeLiteral(p.userResource.Spec.Name))
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
	return IgnoreAlreadyExists(p.conn.Exec(ctx, sqlCreateUser, database.DisableLogger))
}

func (p *Postgres) updatePrivileges(ctx context.Context) error {

	if err := p.processUserTablePrivileges(ctx); err != nil {
		return err
	}

	if err := p.processDBUserRoles(ctx); err != nil {
		return err
	}

	return nil
}

func (p *Postgres) processDBUserRoles(ctx context.Context) error {

	dbPrivsMap, err := p.getDBUserRoles(ctx)
	if err != nil {
		return err
	}
	if err := p.revokeNotDefinedAndAssignDefinedRoles(ctx, dbPrivsMap); err != nil {
		return err
	}
	return nil
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

	err := p.conn.Select(ctx, &privList, queryTemplate, p.userResource.Spec.Name)
	if err != nil {
		return nil, err
	}
	return privList, nil
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

	err := p.conn.Select(ctx, &rolesList, queryTemplate, p.userResource.Spec.Name)
	if err != nil {
		return nil, err
	}
	return rolesList, nil
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
	p.logger.Info(fmt.Sprintf("Getted roles for DB user %s: %v", p.userResource.Spec.Name, roles))
	return roles, nil
}

func (p *Postgres) revokeNotDefinedAndAssignDefinedRoles(ctx context.Context, dbPrivsMap map[authv1alpha1.Privilege]struct{}) error {
	privMap := make(map[authv1alpha1.Privilege]struct{})
	for _, priv := range p.userResource.Spec.Privileges {
		if priv.On == "" {
			privMap[priv] = struct{}{}
		}
	}
	return p.revokeNotDefinedAndAssignDefined(ctx, p.conn, dbPrivsMap, privMap)
}

func (p *Postgres) revokeNotDefinedAndAssignDefined(ctx context.Context, conn *database.DBconnector, dbPrivsMap, privMap map[authv1alpha1.Privilege]struct{}) error {
	var removeQueryList, assignQueryList []database.Query
	userEsc := EscapeLiteral(p.userResource.Spec.Name)
	toCreate, toRevoke := IntersectDefinedPrivsWithDB(privMap, dbPrivsMap)

	for _, priv := range toRevoke {
		revokeQuery := prepareStatementForPriv([]string{"REVOKE %s", "FROM %s"}, priv, userEsc)
		removeQueryList = append(removeQueryList, database.Query{Query: revokeQuery})
	}

	for _, priv := range toCreate {
		sqlGrant := prepareStatementForPriv([]string{"GRANT %s", "TO %s"}, priv, userEsc)
		assignQueryList = append(assignQueryList, database.Query{Query: sqlGrant})
	}

	if queryList := append(removeQueryList, assignQueryList...); len(queryList) > 0 {
		p.logger.Info("QUERY LIST", "query_list", queryList)
		if err := conn.ExecTx(ctx, queryList, []database.NamedQuery{}); err != nil {
			return err
		}
	}
	return nil
}

func prepareStatementForPriv(statement []string, priv authv1alpha1.Privilege, userEsc string) string {
	privEsc, onEsc, databaseEsc := EscapeLiteralWithoutQuotes(string(priv.Privilege)), EscapeLiteral(priv.On), EscapeLiteral(priv.Database)
	query := fmt.Sprintf(strings.Join(statement, " "), privEsc, userEsc)
	if priv.On != "" && priv.Database != "" {
		query = fmt.Sprintf(strings.Join([]string{statement[0], "ON %s", statement[1]}, " "), privEsc, onEsc, userEsc)
	}

	if priv.On == "" && priv.Database != "" {
		query = fmt.Sprintf(strings.Join([]string{statement[0], "ON DATABASE %s", statement[1]}, " "), privEsc, databaseEsc, userEsc)
	}
	return query
}

func (p *Postgres) processUserTablePrivileges(ctx context.Context) error {
	// map[string]map[authv1alpha1.Privilege]struct{}
	privMap := make(map[authv1alpha1.Privilege]struct{})
	for _, priv := range p.userResource.Spec.Privileges {
		if priv.On != "" && priv.Database != "" {
			privMap[priv] = struct{}{}
		}
	}

	dbList, err := p.getAllDatabases(ctx)
	if err != nil {
		return err
	}

	for _, dbname := range dbList {
		if err := p.processUserTablePrivilegesFromDB(ctx, dbname, privMap); err != nil {
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
	p.logger.Info("Getted databases", "databases", databases)
	return databases, nil
}

func (p *Postgres) processUserTablePrivilegesFromDB(ctx context.Context, dbname string, privMap map[authv1alpha1.Privilege]struct{}) error {
	newconf := p.config.Copy()
	newconf.Dbname = dbname
	conn := NewPostgres(newconf, p.configResource, p.userResource, p.client, p.logger)

	if err := conn.Connect(ctx); err != nil {
		return err
	}
	defer conn.Close(ctx)

	var tablePrivs []authv1alpha1.Privilege
	err := conn.conn.Select(ctx, &tablePrivs, "SELECT privilege_type, table_catalog, table_name from information_schema.role_table_grants where grantee = $1", p.userResource.Spec.Name)
	if err != nil {
		return err
	}
	p.logger.Info("DB user table privs", "table_privs", tablePrivs)
	tablePrivsMap := make(map[authv1alpha1.Privilege]struct{})
	for _, priv := range tablePrivs {
		tablePrivsMap[priv] = struct{}{}
	}

	return p.revokeNotDefinedAndAssignDefined(ctx, conn.conn, tablePrivsMap, privMap)
}
