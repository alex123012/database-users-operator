# How to add new database for operator

1. Create new module with the database name inside `/controllers/database` folder.
    > For example: `/controllers/database/mysql`

1. Implement `github.com/alex123012/database-users-operator/controllers/database.DatabaseImplementation` interface ([database.go](/controllers/database/database.go#L33)).
    > For example: [mysql.go](/controllers/database/mysql/mysql.go#L24)

1. Add tests for your code.
    > **NOTE**: You can use [connection.Connection interface](/controllers/database/connection/common.go#L30) to talk to DB for easy testing with [FakeConnection](/controllers/database/connection/fake.go#L24). Refer to [postgresql tests](/controllers/database/postgresql/postgresql_test.go#L34) as example.

1. Add new database type with database name to [database_types.go](/api/v1alpha1/database_types.go#L27).
    > For example: `MySQL DatabaseType = "MySQL"`

1. Add Config type for your database.
    > Refer to [PostgreSQLConfig](/api/v1alpha1/database_types.go#L53)

1. Run `make generate manifests api-docs `

1. Add new database implementation to [newDatabase func](/controllers/database/database.go#L50)
    > Refer to another dbs defined in this func

1. Write tests for controller.
    > Refer to PostgreSQL tests for [DatabaseBinding](/controllers/databasebinding_controller_test.go) and [PrivilegesBinding](/controllers/privilegesbinding_controller_test.go)

1. To run tests use `make test`

1. Make PR to repo.

You can refer to [commit](https://github.com/alex123012/database-users-operator/commit/a328f5b7e64479193dd2e24b248cd8953495c1a2), where MySQL support was added