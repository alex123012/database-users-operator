# How to add new database for operator

1. Create new module with the database name inside `/pkg/database` folder.
    > For example: `/pkg/database/mysql`

1. Implement `github.com/alex123012/database-users-operator/pkg/database.DatabaseImplementation` interface ([database.go](/pkg/database/database.go#L34)).
    > For example: [mysql.go](/pkg/database/mysql/mysql.go#L29)

1. Add tests for your code.
    > **NOTE**: You can use [connection.Connection interface](/pkg/database/connection/common.go#L30) to talk to DB for easy testing with [connection.FakeConnection](/pkg/database/connection/fake.go#L25). Refer to [postgresql tests](/pkg/database/postgresql/postgresql_test.go#L34) as example.

2. Add new database type with database name to [database_types.go](/api/v1alpha1/database_types.go#L27).
    > For example: `MySQL DatabaseType = "MySQL"`

3. Add Config type for your database.
    > Refer to [PostgreSQLConfig](/api/v1alpha1/database_types.go#L57)

4. Run `make generate manifests api-docs`

5. Add new database implementation to [newDatabase func](/pkg/database/database.go#L50)
    > Refer to another dbs defined in this func

6. Write tests for controller.
    > Refer to PostgreSQL tests for [DatabaseBinding](/controllers/databasebinding_controller_test.go) and [PrivilegesBinding](/controllers/privilegesbinding_controller_test.go)

7. To run tests use `make test`

8. Make PR to repo.

You can refer to [commit](https://github.com/alex123012/database-users-operator/commit/a328f5b7e64479193dd2e24b248cd8953495c1a2), where MySQL support was added