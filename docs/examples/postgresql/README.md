# PostgreSQL example

## Create a custom Namespace
It is a good practice to have all components run in a dedicated namespace. Let's run examples in `test-database-users-operator` namespace
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgresql/00-namespace.yaml
```

### Deploy postgres StatefulSet
**In fact, you can use any postgres installation, for example on bare metal or VMs. But for simplicity, we will run postgres in sts**

Deploy example manifests bundle with postgres `StatefulSet`, postgres `service`, secret with password for `postgres` user,
prepare-example `Job` and prepare-example-script `ConfigMap`:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgresql/01-statefulset.yaml
```

```text
statefulset.apps/postgresql-db created
job.batch/prepare-example created
configmap/prepare-example-script created
service/postgres created
secret/postgres created
```

Check postgres pod readiness:
```bash
kubectl get po -n test-database-users-operator -l app=postgresql-db
```

```text
NAME              READY   STATUS    RESTARTS   AGE
postgresql-db-0   1/1     Running   0          6s
```


Check init `Job` pod readiness:

```bash
kubectl get po -n test-database-users-operator -l job-name=prepare-example
```

```text
NAME                    READY   STATUS      RESTARTS   AGE
prepare-example-f2z4r   0/1     Completed   0          12s
```

Exec in to the postgres pod:
```bash
kubectl exec -ti -n test-database-users-operator postgresql-db-0 -- psql --user postgres
```
```text
psql (15.3 (Debian 15.3-1.pgdg120+1))
Type "help" for help.
```
Check database roles:
```sql
\du
```
> `some_role` was created by prepare-example `Job`
```text
                                   List of roles
 Role name |                         Attributes                         | Member of
-----------+------------------------------------------------------------+-----------
 postgres  | Superuser, Create role, Create DB, Replication, Bypass RLS | {}
 some_role | Cannot login                                               | {}
```

Connect to the database, created by prepare-example `Job` and check its table:
```sql
\c some_db
```
```text
You are now connected to database "some_db" as user "postgres".
```

```sql
\d
```

```text
           List of relations
 Schema |    Name    | Type  |  Owner
--------+------------+-------+----------
 public | some_table | table | postgres
(1 row)
```

### Deploy `Database` and `Privileges`

Exit from postgres pod and create `Database` resource:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgresql/02-database.yaml
```

```text
database.databaseusersoperator.com/postgres created
```

And `Privileges` resource:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgresql/03-privileges.yaml
```

```text
privileges.databaseusersoperator.com/some-privileges created
```

### Deploy `User`

Create `User` resource `john` with `postgres` `Database` and `some-privileges` `Privileges` references and `Secret` with password for it. It will "say" operator to create user `john` and assign to it privileges from `some-privileges` `Privileges` resource in previously created PostgreSQL DB:

```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgresql/04-user.yaml
```

```text
user.databaseusersoperator.com/john created
secret/postgres-john configured
```

Wait for user creation in the PostgreSQL database:
```bash
while kubectl get users.databaseusersoperator.com -n test-database-users-operator john -ojson | jq -e '.status.summary.ready != true' > /dev/null; do echo waiting for ready status of john User; done
```

Exec in to the postgres pod with `john` user and to `some_db` database:
```bash
kubectl exec -ti -n test-database-users-operator postgresql-db-0 -- psql --user john --dbname some_db
```

```text
psql (15.3 (Debian 15.3-1.pgdg120+1))
Type "help" for help.
```

Check database roles:
```sql
\du
```

```text
                                    List of roles
 Role name |                         Attributes                         |  Member of
-----------+------------------------------------------------------------+-------------
 john      |                                                            | {some_role}
 postgres  | Superuser, Create role, Create DB, Replication, Bypass RLS | {}
 some_role | Cannot login                                               | {}
```

> The `john` user should be member of the `some_role` role.


## Cleanup
Simply delete the `test-database-users-operator` namespace:
```bash
kubectl delete namespaces test-database-users-operator
```
```text
namespace "test-database-users-operator" deleted
```
