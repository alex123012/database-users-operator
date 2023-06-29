# API Reference

## Packages
- [databaseusersoperator.com/v1alpha1](#databaseusersoperatorcomv1alpha1)


## databaseusersoperator.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the  v1alpha1 API group


### Resource Types
- [Database](#database)
- [Privileges](#privileges)
- [User](#user)



#### Database



Database is the Schema for the databases API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `databaseusersoperator.com/v1alpha1`
| `kind` _string_ | `Database`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[DatabaseSpec](#databasespec)_ |  |


#### DatabaseRef





_Appears in:_
- [UserSpec](#userspec)

| Field | Description |
| --- | --- |
| `name` _string_ | The name of the Database CR to create user in, required. |
| `passwordSecret` _[Secret](#secret)_ | Reference to secret with password for user in the database, not required. |
| `createdSecret` _[NamespacedName](#namespacedname)_ | If operator would create data for user (for example for postgres with sslMode=="verify-full"), it is reference to non-existed Secret, that will be created during user creation in the database, not required. |
| `privileges` _[Name](#name) array_ | List of references to Privileges CR, that will be applied to created user in the database, required. |


#### DatabaseSpec



DatabaseSpec defines the desired state of Database.

_Appears in:_
- [Database](#database)

| Field | Description |
| --- | --- |
| `databaseType` _DatabaseType_ | Type of database to connect (Currently it is PostgreSQL and MySQL), required |
| `postgreSQL` _[PostgreSQLConfig](#postgresqlconfig)_ | Config for connecting for PostgreSQL compatible databases, not required. required if DatabaseType equals to "PostgreSQL". |
| `mySQL` _[MySQLConfig](#mysqlconfig)_ | Config for connecting for MySQL compatible databases, not required. required if DatabaseType equals to "MySQL". |


#### MySQLConfig





_Appears in:_
- [DatabaseSpec](#databasespec)

| Field | Description |
| --- | --- |
| `host` _string_ | Full DNS name/ip for database to use, required. If K8S service is used to connect - provide host as <db-service-name>.<db-service-namespace>.svc.cluster.local refer to --host flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html |
| `port` _integer_ | k8s-service/database port to connect to execute queries, defaults to 3306. refer to --port flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html |
| `databaseName` _string_ | Database name that will be used to connect to database, not required. see https://dev.mysql.com/doc/refman/8.0/en/connecting.html. |
| `user` _string_ | The MySQL user account to provide for the authentication process, defaults to "mysql". It must have at least CREATE ROLE privilege (if you won't provide superuser acess to users) or database superuser role if you think you'll be needed to give some users database superuser privileges refer to --user flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html and https://dev.mysql.com/doc/refman/8.0/en/privileges-provided.html#privileges-provided-guidelines "Privilege-Granting Guidelines" |
| `passwordSecret` _[Secret](#secret)_ | Secret with password for User to connect to database refer to --password flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html |
| `usersHostname` _string_ | The hostname from which created users will connect By default "*" will be used (So users would be "<user>@*") |


#### Name





_Appears in:_
- [DatabaseRef](#databaseref)

| Field | Description |
| --- | --- |
| `name` _string_ | resource name |


#### NamespacedName





_Appears in:_
- [DatabaseRef](#databaseref)
- [PostgreSQLConfig](#postgresqlconfig)
- [Secret](#secret)

| Field | Description |
| --- | --- |
| `namespace` _string_ | resource namespace |
| `name` _string_ | resource name |


#### PostgreSQLConfig



PostgreSQLConfig is config that will be used by operator to connect to PostgreSQL compatible databases.

_Appears in:_
- [DatabaseSpec](#databasespec)

| Field | Description |
| --- | --- |
| `host` _string_ | Full DNS name/ip for database to use, required. If K8S service is used to connect - provide full dns name as <db-service-name>.<db-service-namespace>.svc.cluster.local refer to --host flag in https://www.postgresql.org/docs/current/app-psql.html |
| `port` _integer_ | k8s-service/database port to connect to execute queries, defaults to 5432. refer to --port flag in https://www.postgresql.org/docs/current/app-psql.html |
| `user` _string_ | User that will be used to connect to database, defaults to "postgres". It must have at least CREATEROLE privilege (if you won't provide superuser acess to users) or database superuser role if you think you'll be needed to give some users database superuser privileges refer to --username flag in https://www.postgresql.org/docs/current/app-psql.html and https://www.postgresql.org/docs/current/sql-grant.html "GRANT on Roles" |
| `sslMode` _[PostgresSSLMode](#postgressslmode)_ | SSL mode that will be used to connect to PostgreSQL, defaults to "disable". Posssible values: "disable", "allow", "prefer", "require", "verify-ca", "verify-full". If SSL mode is "require", "verify-ca", "verify-full" - operator will generate K8S secret with SSL bundle (CA certificate, user certificate and user key) for User CR with same name as User CR. see https://www.postgresql.org/docs/current/libpq-ssl.html |
| `databaseName` _string_ | Database name that will be used to connect to database, not required refer to --dbname flag in https://www.postgresql.org/docs/current/app-psql.html |
| `sslSecret` _[NamespacedName](#namespacedname)_ | Secret with SSL CA certificate ("ca.crt" key), user certificate ("tls.crt" key) and user key ("tls.key" key). If SSL Mode equals to "disable", "allow" or "prefer" field is not required. If SSL Mode equals to "require", "verify-ca" or "verify-full" - required. see https://www.postgresql.org/docs/current/libpq-ssl.html |
| `sslCaKey` _[Secret](#secret)_ | Secret with CA key for creating users certificates If SSL Mode equals to "disable", "allow" or "prefer" field is not required. If SSL Mode equals to "require", "verify-ca" or "verify-full" - required. see https://www.postgresql.org/docs/current/libpq-ssl.html |
| `passwordSecret` _[Secret](#secret)_ | Secret with password for User to connect to database If SSL Mode equals to "disable", "allow" or "prefer" field is required. If SSL Mode equals to "require", "verify-ca" or "verify-full" - not required. refer to --password flag in https://www.postgresql.org/docs/current/app-psql.html |


#### PostgresSSLMode

_Underlying type:_ `string`



_Appears in:_
- [PostgreSQLConfig](#postgresqlconfig)



#### PrivilegeSpec



PrivilegesSpec defines the desired state of Privileges.

_Appears in:_
- [Privileges](#privileges)

| Field | Description |
| --- | --- |
| `privilege` _PrivilegeType_ | Privilege is role name or PrivilegeType, required. |
| `on` _string_ | In database object to give privileges to, not required. |
| `database` _string_ | If Privilege is database specific - this field will be used to determine which db to use, not required. |


#### Privileges



Privileges is the Schema for the privileges API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `databaseusersoperator.com/v1alpha1`
| `kind` _string_ | `Privileges`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `privileges` _[PrivilegeSpec](#privilegespec) array_ | List of privileges, required. |


#### Secret



Secret is a reference for kubernetes secret.

_Appears in:_
- [DatabaseRef](#databaseref)
- [MySQLConfig](#mysqlconfig)
- [PostgreSQLConfig](#postgresqlconfig)

| Field | Description |
| --- | --- |
| `secret` _[NamespacedName](#namespacedname)_ | Secret is secret name and namespace |
| `key` _string_ | Kubernetes secret key with data |


#### StatusSummary





_Appears in:_
- [UserStatus](#userstatus)

| Field | Description |
| --- | --- |
| `ready` _boolean_ |  |
| `message` _string_ |  |


#### User



User is the Schema for the users API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `databaseusersoperator.com/v1alpha1`
| `kind` _string_ | `User`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[UserSpec](#userspec)_ |  |


#### UserSpec



UserSpec defines the desired state of User.

_Appears in:_
- [User](#user)

| Field | Description |
| --- | --- |
| `databases` _[DatabaseRef](#databaseref) array_ | List of databases, where user needs to be created with configs for it. |




