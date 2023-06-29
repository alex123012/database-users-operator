```yaml
---
apiVersion: databaseusersoperator.com/v1alpha1
kind: Database
metadata:
  name: postgres
  namespace: test-database-users-operator
spec:
	# Type of database to connect (Currently it is PostgreSQL and MySQL), required
  databaseType: PostgreSQL

	# Config for connecting for PostgreSQL compatible databases, not required.
	# required if databaseType equals to "PostgreSQL".
  postgreSQL:
    # Full DNS name/ip for database to use, required.
    # If K8S service is used to connect - provide full dns name
    # as <db-service-name>.<db-service-namespace>.svc.cluster.local
    # refer to --host flag in https://www.postgresql.org/docs/current/app-psql.html
    host: postgres-svc.postgres-namespace.svc.cluster.local

    # k8s-service/database port to connect to execute queries, defaults to 5432.
    # refer to --port flag in https://www.postgresql.org/docs/current/app-psql.html
    port: 5432

    # User that will be used to connect to database, defaults to "postgres".
    # It must have at least CREATEROLE privilege (if you won't provide superuser acess to users)
    # or database superuser role if you think you'll be needed to give some users database superuser privileges
    # refer to --username flag in https://www.postgresql.org/docs/current/app-psql.html
    # and https://www.postgresql.org/docs/current/sql-grant.html "GRANT on Roles"
    user: postgres



    # SSL mode that will be used to connect to PostgreSQL, defaults to "disable".
    # Posssible values: "disable", "allow", "prefer", "require", "verify-ca", "verify-full".
    # If SSL mode is "require", "verify-ca", "verify-full" - operator will generate K8S secret with
    # SSL bundle (CA certificate, user certificate and user key) for User CR
    # with the name and namespace, provided in the User CR spec.[].createdSecret.
    # see https://www.postgresql.org/docs/current/libpq-ssl.html
    sslMode: sslmode


    # Database name that will be used to connect to database, not required
    # refer to --dbname flag in https://www.postgresql.org/docs/current/app-psql.html
    databaseName: dbname


    # Secret with password for User to connect to database
    # If SSL Mode equals to "disable", "allow" or "prefer" field is required.
    # If SSL Mode equals to "require", "verify-ca" or "verify-full" - not required.
    # refer to --password flag in https://www.postgresql.org/docs/current/app-psql.html
    passwordSecret:
      # Secret data key with password for user
      key: password-key
      secret:
        # Secret name
        name: password-secret-name
        # Secret namespace
        namespace: password-secret-namespace

    # Secret with SSL CA certificate ("ca.crt" Secret data key), user certificate ("tls.crt" Secret data key) and user key ("tls.key" Secret data key).
    # If SSL Mode equals to "disable", "allow" or "prefer" field is not required.
    # If SSL Mode equals to "require", "verify-ca" or "verify-full" - required.
    # see https://www.postgresql.org/docs/current/libpq-ssl.html
    sslSecret:
      # Secret name
      name: ssl-secret-name
      # Secret namespace
      namespace: ssl-secret-namespace

    # Secret with CA key for creating users certificates
    # If SSL Mode equals to "disable", "allow" or "prefer" field is not required.
    # If SSL Mode equals to "require", "verify-ca" or "verify-full" - required.
    # see https://www.postgresql.org/docs/current/libpq-ssl.html
    sslCaKey:
      # Secret data key with CA key
      key: ssl-ca-key-data-key
      secret:
        # Secret name
        name: ssl-ca-key-name
        # Secret namespace
        namespace: ssl-ca-key-namespace

  # Config for connecting for MySQL compatible databases, not required.
	# required if DatabaseType equals to "MySQL".
  mySQL:
    # Full DNS name/ip for database to use, required.
    # If K8S service is used to connect - provide host
    # as <db-service-name>.<db-service-namespace>.svc.cluster.local
    # refer to --host flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html
    host: mysql-svc.mysql-namespace.svc.cluster.local

    # k8s-service/database port to connect to execute queries, defaults to 3306.
    # refer to --port flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html
    port: 3306

    # Database name that will be used to connect to database, not required.
    # see https://dev.mysql.com/doc/refman/8.0/en/connecting.html.
    databaseName: dbname

    # The MySQL user account to provide for the authentication process, defaults to "mysql".
    # It must have at least CREATE ROLE privilege (if you won't provide superuser acess to users)
    # or database superuser role if you think you'll be needed to give some users database superuser privileges
    # refer to --user flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html
    # and https://dev.mysql.com/doc/refman/8.0/en/privileges-provided.html#privileges-provided-guidelines "Privilege-Granting Guidelines"
    user: mysql

    # Secret with password for User to connect to database
    # refer to --password flag in https://dev.mysql.com/doc/refman/8.0/en/connection-options.html
    passwordSecret:
      # Secret data key with password for user
      key: password-key
      secret:
        # Secret name
        name: password-secret-name
        # Secret namespace
        namespace: password-secret-namespace

    # The hostname from which created users will connect
    # By default "*" will be used (So users would be "<user>@*")
    usersHostname: "*"
```
