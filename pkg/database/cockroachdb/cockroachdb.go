package cockroachdb

// import (
// 	"context"
// 	"fmt"
// 	"strings"

// 	authv1alpha1 "github.com/alex123012/k8s-database-users-operator/api/v1alpha1"
// 	"github.com/alex123012/k8s-database-users-operator/pkg/database/postgresql"
// 	"github.com/jackc/pgx/v5"
// )

// type Postgres struct {
// 	conn   *pgx.Conn
// 	config *postgresql.PostgresConfig
// }

// func NewPostgres(config *PostgresConfig) *Postgres {
// 	return &Postgres{
// 		config: config,
// 	}
// }
// func (p *Postgres) Connect(ctx context.Context) error {
// 	conf, err := p.config.GetConfig()
// 	if err != nil {
// 		return err
// 	}
// 	p.conn, err = pgx.ConnectConfig(ctx, conf)
// 	return err
// }

// func (p *Postgres) Close(ctx context.Context) {
// 	p.conn.Close(ctx)
// }

// func (p *Postgres) ProcessUser(ctx context.Context, userResource *authv1alpha1.User, password string) error {
// 	if err := p.createUser(ctx, userResource.Spec.Name, password); err != nil {
// 		return err
// 	}
// 	if err := p.updatePrivileges(ctx, userResource.Spec.Name, userResource.Spec.Privileges); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (p *Postgres) createUser(ctx context.Context, username, password string) error {
// 	sqlCreateUser := fmt.Sprintf(`CREATE USER %s`, EscapeLiteral(username))
// 	if password != "" {
// 		sqlCreateUser += fmt.Sprintf(" WITH PASSWORD %s", EscapeString(password))
// 	}
// 	_, err := p.conn.Exec(ctx, sqlCreateUser)
// 	return IgnoreAlreadyExists(err)
// }

// func (p *Postgres) updatePrivileges(ctx context.Context, username string, privileges []authv1alpha1.Privilege) error {
// 	privMap := make(map[authv1alpha1.Privilege]struct{})
// 	for _, priv := range privileges {
// 		privMap[priv] = struct{}{}
// 	}
// 	tablePrivsList, err := p.getUserTablePrivileges(ctx, username)
// 	if err != nil {
// 		return err
// 	}
// 	rolesList, err := p.getUserRolePrivileges(ctx, username)
// 	if err != nil {
// 		return err
// 	}
// 	if err := p.removeNotDefined(ctx, append(tablePrivsList, rolesList...), privMap, username); err != nil {
// 		return err
// 	}

// 	if err := p.assignDefinedRoles(ctx, privileges, username); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (p *Postgres) assignDefinedRoles(ctx context.Context, privileges []authv1alpha1.Privilege, username string) error {
// 	resultQueryList := []string{"BEGIN"}
// 	userEsc := EscapeLiteral(username)
// 	for _, priv := range privileges {
// 		privEsc, onEsc := EscapeLiteral(string(priv.Privilege)), EscapeLiteral(priv.On)
// 		sqlGrant := fmt.Sprintf(`GRANT %s TO %s`, privEsc, userEsc)
// 		if priv.On != "" {
// 			sqlGrant = fmt.Sprintf(`GRANT %s ON %s TO %s`, privEsc, onEsc, userEsc)
// 		}
// 		resultQueryList = append(resultQueryList, sqlGrant)
// 	}
// 	resultQuery := strings.Join(append(resultQueryList, "END;"), ";")
// 	// fmt.Println(resultQuery)
// 	if _, err := p.conn.Exec(ctx, resultQuery); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (p *Postgres) removeNotDefined(ctx context.Context, dbPrivsList []authv1alpha1.Privilege, CRPrivsMap map[authv1alpha1.Privilege]struct{}, username string) error {
// 	removeQueryList := []string{"BEGIN"}
// loop:
// 	for _, priv := range dbPrivsList {
// 		checkList := []authv1alpha1.Privilege{priv}

// 		privUpper := authv1alpha1.PrivilegeType(strings.ToUpper(string(priv.Privilege)))
// 		_, f := authv1alpha1.PrivilegeTypeMap[privUpper]
// 		privRule := f && !strings.HasPrefix(strings.ToLower(string(priv.Privilege)), "all")

// 		onRule := priv.On != "" && !strings.HasSuffix(priv.On, ".*")

// 		var tmpOn string
// 		if onRule {
// 			tmp := strings.Split(priv.On, ".")
// 			tmpOn = strings.Join(tmp[:len(tmp)-1], ".") + ".*"
// 			checkList = append(checkList, authv1alpha1.Privilege{
// 				Privilege: priv.Privilege,
// 				On:        tmpOn,
// 			})
// 		}
// 		if privRule {
// 			checkList = append(checkList, authv1alpha1.Privilege{
// 				Privilege: authv1alpha1.ALLPRIVILEGES,
// 				On:        priv.On,
// 			})
// 		}

// 		if privRule && onRule {
// 			checkList = append(checkList, authv1alpha1.Privilege{
// 				Privilege: authv1alpha1.ALLPRIVILEGES,
// 				On:        tmpOn,
// 			})
// 		}
// 		for _, rule := range checkList {
// 			if _, f := CRPrivsMap[rule]; f {
// 				// fmt.Printf("%v\n", rule)
// 				continue loop
// 			}
// 		}

// 		escPriv, escUser := EscapeLiteral(string(priv.Privilege)), EscapeLiteral(username)
// 		removeQuery := fmt.Sprintf("REVOKE %s FROM %s", escPriv, escUser)
// 		if priv.On != "" {
// 			removeQuery = fmt.Sprintf("REVOKE %s ON %s FROM %s", escPriv, EscapeLiteral(priv.On), escUser)
// 		}
// 		removeQueryList = append(removeQueryList, removeQuery)
// 	}
// 	if len(removeQueryList) > 1 {
// 		resultQuery := strings.Join(append(removeQueryList, "END;"), ";")
// 		// fmt.Printf("%v\n", resultQuery)
// 		if _, err := p.conn.Exec(ctx, resultQuery); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }
// func (p *Postgres) getUserRolePrivileges(ctx context.Context, username string) ([]authv1alpha1.Privilege, error) {
// 	var rolesList []string
// 	queryTemplate := `
// 	SELECT
// 	ARRAY(SELECT b.rolname
// 	  FROM pg_catalog.pg_auth_members m
// 	  JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
// 	  WHERE m.member = r.oid) as memberof
// 	FROM pg_catalog.pg_roles r where r.rolname = %s`

// 	err := p.conn.QueryRow(ctx, fmt.Sprintf(queryTemplate, EscapeString(username))).Scan(&rolesList)
// 	if err != nil {
// 		return nil, err
// 	}
// 	roles := make([]authv1alpha1.Privilege, len(rolesList))
// 	for i := range roles {
// 		roles[i] = authv1alpha1.Privilege{
// 			Privilege: authv1alpha1.PrivilegeType(rolesList[i]),
// 		}
// 	}
// 	return roles, nil
// }

// func (p *Postgres) getUserTablePrivileges(ctx context.Context, username string) ([]authv1alpha1.Privilege, error) {
// 	dbList, err := p.getAllDatabases(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	queryList := make([]string, len(dbList))
// 	for i := range dbList {
// 		queryList[i] = fmt.Sprintf(
// 			"SELECT privilege_type, table_catalog, table_name from %s.information_schema.table_privileges where grantee = %s",
// 			dbList[i], EscapeString(username))
// 	}
// 	query := strings.Join(queryList, " UNION ")

// 	rows, err := p.conn.Query(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	privs := make([]authv1alpha1.Privilege, 0)
// 	for rows.Next() {
// 		values, err := rows.Values()
// 		if err != nil {
// 			return nil, err
// 		}
// 		privs = append(privs, authv1alpha1.Privilege{
// 			Privilege: authv1alpha1.PrivilegeType(values[0].(string)),
// 			On:        values[1].(string) + "." + values[2].(string),
// 		})
// 	}
// 	return privs, nil
// }

// func (p *Postgres) getAllDatabases(ctx context.Context) ([]string, error) {
// 	rows, err := p.conn.Query(ctx, "SELECT datname FROM pg_database;")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	databases := make([]string, 0)
// 	for rows.Next() {
// 		values, err := rows.Values()
// 		if err != nil {
// 			return nil, err
// 		}
// 		databases = append(databases, EscapeLiteral(values[0].(string)))
// 	}
// 	return databases, nil
// }

// func (p *Postgres) ProcessCertificates(ctx context.Context, userResource *authv1alpha1.User, caCert, caKey []byte) (map[string][]byte, error) {
// 	return nil, nil
// }
