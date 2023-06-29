# CockroachDB example

## Create a custom Namespace
It is a good practice to have all components run in a dedicated namespace. Let's run examples in `test-database-users-operator` namespace
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/00-namespace.yaml
```

### Deploy CockroachDB StatefulSet
**In fact, you can use any CockroachDB installation, for example on bare metal or VMs. But for simplicity, we will run CockroachDB in sts**

First set up certificates for cockroach and load them into Kubernetes cluster as Secrets using the commands below:
```bash
# Setup
mkdir certs
mkdir my-safe-directory

# Create CA certificate and key
cockroach cert create-ca --certs-dir=certs --ca-key=my-safe-directory/ca.key
kubectl create secret -n test-database-users-operator generic cockroachdb.ca.key --from-file=my-safe-directory/ca.key

# Create certs for root user
cockroach cert create-client root --certs-dir=certs --ca-key=my-safe-directory/ca.key
kubectl create secret -n test-database-users-operator generic cockroachdb.client.root --from-literal="tls.crt=$(cat certs/client.root.crt)" --from-literal="tls.key=$(cat certs/client.root.key)" --from-literal="ca.crt=$(cat certs/ca.crt)"

# Create certs for cockroach nodes
cockroach cert create-node --certs-dir=certs --ca-key=my-safe-directory/ca.key localhost 127.0.0.1 cockroachdb-public cockroachdb-public.test-database-users-operator cockroachdb-public.test-database-users-operator.svc.cluster.local *.cockroachdb *.cockroachdb.test-database-users-operator *.cockroachdb.test-database-users-operator.svc.cluster.local
kubectl create secret generic cockroachdb.node -n test-database-users-operator --from-file=certs
```

Deploy example manifests bundle with CockroachDB `StatefulSet`:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/01-statefulset.yaml
```

```text
serviceaccount/cockroachdb created
role.rbac.authorization.k8s.io/cockroachdb created
rolebinding.rbac.authorization.k8s.io/cockroachdb created
service/cockroachdb-public created
service/cockroachdb created
poddisruptionbudget.policy/cockroachdb-budget created
statefulset.apps/cockroachdb created
```

Initialize `CockroachDB`:
```
kubectl exec -it -n test-database-users-operator cockroachdb-0 -- /cockroach/cockroach init --certs-dir=/cockroach/cockroach-certs
```
```text
Cluster successfully initialized
```

Check CockroachDB pods readiness:
```bash
kubectl get po -n test-database-users-operator -l app=cockroachdb
```

```text
NAME            READY   STATUS    RESTARTS   AGE
cockroachdb-0   1/1     Running   0          60s
cockroachdb-1   1/1     Running   0          60s
cockroachdb-2   1/1     Running   0          60s
```

Deploy manifests with CockroachDB setup `Job` - `prepare-example`:
```
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/02-job.yaml
```

```text
job.batch/prepare-example created
configmap/prepare-example-script created
```

Check `Job` pod readiness:

```bash
kubectl get po -n test-database-users-operator -l job-name=prepare-example
```

```text
NAME                    READY   STATUS      RESTARTS   AGE
prepare-example-vqnfn   0/1     Completed   0          19s
```

Exec in to the `CocroachDB` pod:
```bash
kubectl exec -it -n test-database-users-operator cockroachdb-0 -- cockroach sql --certs-dir=/cockroach/cockroach-certs
```
```text
#
# Welcome to the CockroachDB SQL shell.
# All statements must be terminated by a semicolon.
# To exit, type: \q.
#
# Server version: CockroachDB CCL v23.1.4 (aarch64-unknown-linux-gnu, built 2023/06/16 21:18:51, go1.19.4) (same version as client)
# Cluster ID: 4ada3fc0-0d4f-485a-9f58-5abde9584123
#
# Enter \? for a brief introduction.
#
```
Check database roles:
```sql
\du
```
> `some_role` was created by prepare-example `Job`
```text
List of roles:
  Role name |                   Attributes                    | Member of
------------+-------------------------------------------------+------------
  root      | Superuser, Create role, Create DB               | {admin}
  admin     | Superuser, Create role, Create DB               | {}
  some_role | Cannot login                                    | {}
  node      | Superuser, Create role, Create DB, Cannot login | {}
(4 rows)                                             | {}
```

Connect to the database, created by prepare-example `Job` and check its table:
```sql
\c some_db
```
```text
using new connection URL: postgresql://root@localhost:26257/some_db?application_name=%24+cockroach+sql&connect_timeout=15&sslcert=%2Fcockroach%2Fcockroach-certs%2Fclient.root.crt&sslkey=%2Fcockroach%2Fcockroach-certs%2Fclient.root.key&sslmode=verify-full&sslrootcert=%2Fcockroach%2Fcockroach-certs%2Fca.crt
```

```sql
\d
```

```text
List of relations:
  Schema |      Name       | Type  | Owner |   Table
---------+-----------------+-------+-------+-------------
  public | some_table      | table | root  | NULL
  public | some_table_pkey | index | root  | some_table
(2 rows)
```

### Deploy `Database` and `Privileges`

Exit from `CockroachDB` pod and create `Database` resource:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/03-database.yaml
```

```text
database.databaseusersoperator.com/cockroachdb created
```

And `Privileges` resource:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/04-privileges.yaml
```

```text
privileges.databaseusersoperator.com/some-privileges created
```

### Deploy `User`

Create `User` resource `john` with `cockroachdb` `Database` and `some-privileges` `Privileges` references and `createdSecret` with name and namespace for `Secret` where operator will store certificates for connection. It will "say" operator to create user `john` and assign to it privileges from `some-privileges` `Privileges` resource in previously created `CockroachDB` DB and create certificates for this user:

```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/05-user.yaml
```

```text
user.databaseusersoperator.com/john created
```

Wait for user creation in the `CockroachDB` database:
```bash
while kubectl get users.databaseusersoperator.com -n test-database-users-operator john -ojson | jq -e '.status.summary.ready != true' > /dev/null; do echo waiting for ready status of john User; done
```

Create pod with `john` certificates to connect to `CockroachDB`:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/cockroachdb/06-pod.yaml
```
```text
pod/cockroachdb-john created
```

Exec in to the `cockroachdb-john` pod with `john` user and to `some_db` database:
```bash
kubectl exec -it -n test-database-users-operator cockroachdb-john -- cockroach sql --certs-dir=/cockroach/cockroach-certs -u john -d some_db --host cockroachdb-public
```

```text
#
# Welcome to the CockroachDB SQL shell.
# All statements must be terminated by a semicolon.
# To exit, type: \q.
#
# Server version: CockroachDB CCL v23.1.4 (aarch64-unknown-linux-gnu, built 2023/06/16 21:18:51, go1.19.4) (same version as client)
# Cluster ID: 4ada3fc0-0d4f-485a-9f58-5abde9584123
#
# Enter \? for a brief introduction.
#
```

Check database roles:
```sql
\du
```

```text
List of roles:
  Role name |                   Attributes                    |  Member of
------------+-------------------------------------------------+--------------
  root      | Superuser, Create role, Create DB               | {admin}
  john      |                                                 | {some_role}
  admin     | Superuser, Create role, Create DB               | {}
  some_role | Cannot login                                    | {}
  node      | Superuser, Create role, Create DB, Cannot login | {}
(5 rows)                                               | {}
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
