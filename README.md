### **In active development**
# Database Users Kubernetes Operator

Kubernetes operator to create and manage users and roles for various SQL and NoSQL databases (currently supports PostgreSQL, CockroachDB). This repository contains a [custom controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) and [custom resource definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) designed for the lifecycle (creation, update privileges, deletion) of a different databases users/roles.
# Features
* Currently supports PostgreSQL (roles, table privileges, database privileges) and CockroachDB (roles and table privileges, database privileges not working properly)
* Create users/roles and assign privileges to them in databases
* Change users/roles privileges in databases in runtime
* Delete user/role in databases when custom resource is deleted
# Prerequisites

1. Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).
1. Properly configured `kubectl`
1. `curl`

# Database Users Operator Installation

Apply `database-users-operator` installation manifest. The simplest way - directly from `github`.

## **In case you are OK to install operator into `database-users-operator-system` namespace**

just run:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/deploy/manifests.yaml
```

## **In case you would like to customize installation parameters**,

Download the fully bundled manifests and customize them yourself
```bash
curl -so database-users-operator-manifests.yaml https://raw.githubusercontent.com/alex123012/database-users-operator/main/deploy/manifests.yaml
```
And apply:
```bash
kubectl apply -f database-users-operator-manifests.yaml
```

It will install **database-users-operator** into `database-users-operator-system` namespace and will watch custom resources like a
```yaml
apiVersion: auth.alex123012.com/v1alpha1
kind: Config
```
and
```yaml
apiVersion: auth.alex123012.com/v1alpha1
kind: User
```
in all available namespaces.

## Operator installation process
```text
namespace/database-users-operator-system created
customresourcedefinition.apiextensions.k8s.io/configs.auth.alex123012.com created
customresourcedefinition.apiextensions.k8s.io/users.auth.alex123012.com created
serviceaccount/database-users-operator-controller-manager created
clusterrole.rbac.authorization.k8s.io/database-users-operator-manager-role created
clusterrolebinding.rbac.authorization.k8s.io/database-users-operator-manager-rolebinding created
service/database-users-operator-webhook-service created
deployment.apps/database-users-operator-controller-manager created
certificate.cert-manager.io/database-users-operator-serving-cert created
issuer.cert-manager.io/database-users-operator-selfsigned-issuer created
mutatingwebhookconfiguration.admissionregistration.k8s.io/database-users-operator-mutating-webhook-configuration created
validatingwebhookconfiguration.admissionregistration.k8s.io/database-users-operator-validating-webhook-configuration created
```

Check `database-users-operator-system` is running:
```bash
kubectl get pods -n database-users-operator-system
```
```text
NAME                                                          READY   STATUS    RESTARTS   AGE
database-users-operator-controller-manager-777dcc4765-nb76m   1/1     Running   0          36s
```
# Examples

There are several ready-to-use [User and Config examples](docs/examples). Below are a few to start with.

## Create Custom Namespace
It is a good practice to have all components run in dedicated namespaces. Let's run examples in `test-database-users-operator` namespace
```bash
kubectl create namespace test-database-users-operator
```
```text
namespace/test-database-users-operator created
```

## Postgres example

### Deploy postgres statefulset
**In fact, you can use any postgres installation, for example on bare metal or VMs. But for simplicity, we will run postgres in sts**

Deploy example manifest bundle with PostgreSQL sts, postgres service, secret with password for `postgres` user and secret with password for `john` user:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgres-sts.yaml
```

```text
statefulset.apps/postgresql-db created
service/postgres created
secret/postgres created
secret/postgres-john created
```

Check postgres pod readiness:
```bash
kubectl get pods -n test-database-users-operator
```

```text
NAME              READY   STATUS    RESTARTS   AGE
postgresql-db-0   1/1     Running   0          2m39s
```

Exec in to pod:
```bash
kubectl exec -ti -n test-database-users-operator postgresql-db-0 -- psql --user postgres
```
```text
psql (15.0 (Debian 15.0-1.pgdg110+1))
Type "help" for help.
```
check default users and their privileges
```sql
\du
```
```text
                                   List of roles
 Role name |                         Attributes                         | Member of
-----------+------------------------------------------------------------+-----------
 postgres  | Superuser, Create role, Create DB, Replication, Bypass RLS | {}
```

Create `test` database:
```sql
CREATE DATABASE test;
```
```text
CREATE DATABASE
```
connect to it
```sql
\c test
```
```text
You are now connected to database "test" as user "postgres".
```
create table `persons`:
```sql
CREATE TABLE Persons (
    PersonID int,
    LastName varchar(255),
    FirstName varchar(255),
    Address varchar(255),
    City varchar(255)
);
```
```text
CREATE TABLE
```

Now exit from postgres pod and create example config and user resources:
```bash
kubectl apply -f https://raw.githubusercontent.com/alex123012/database-users-operator/main/docs/examples/postgres-user-example.yaml
```

CR for Config
```yaml
---
apiVersion: auth.alex123012.com/v1alpha1
kind: Config
metadata:
  name: postgres # name to use in .spec.databaseConfig field in User CR
  namespace: test-database-users-operator # namespace to use in .spec.databaseConfig field in User CR
spec:
  databaseType: PostgreSQL # Database type (currently supported is only PostgreSQL)
  postgreSQL: # Config for databaseType 'PostgreSQL'
    # Full name of service with namespace and internal cluster domain
    # or external domain/ip for database
    host: postgres.test-database-users-operator.svc.cluster.local
    # Database port to connect
    port: 5432
    # User with privileges to create and update roles
    user: postgres
    # SSL mode to use (refer to https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-PROTECTION)
    sslMode: disable
    # Secret with key 'password' for user provided in field .spec.user
    passwordSecret:
      name: postgres
      namespace: test-database-users-operator
    # SSL secrets config (will be used
    # if .spec.sslMode == ("verify-ca" || "required" || "verify-full"))
    # sslSecrets:
    #   userSecret:
    #     name: postgres-user-keypair
    #     namespace: test-database-users-operator
    #   caSecret:
    #     name: postgres-ca-key
    #     namespace: test-database-users-operator
```

CR for user:
```yaml
---
apiVersion: auth.alex123012.com/v1alpha1
kind: User
metadata:
  name: john # This name will be used for user(role) in database
  namespace: test-database-users-operator
spec:
  passwordSecret:
    # secret with key 'password' containing user password to assign
    name: postgres-john
    namespace: test-database-users-operator
  privileges: # List of applyable privileges
    # This privilege will be applied on database test
    # like `GRANT CREATE ON DATABASE test TO john`
    - privilege: CREATE
      database: test
    # This privilege will be applied on table persons in database test
    # like `GRANT INSERT ON persons TO john`
    - privilege: INSERT
      "on": persons
      database: test
    # This privilege will be applied as role for user
    # like `GRANT postgres TO john`
    - privilege: postgres
  # List of database configs to use (refer to above yaml manifest)
  databaseConfigs:
    - name: postgres
      namespace: test-database-users-operator
```
```text
config.auth.alex123012.com/postgres created
user.auth.alex123012.com/john created
```

### Connect to Postgres Database and check user privileges

Now exec in to postgres pod with psql another time and check user creation and applied privileges:
```bash
kubectl exec -ti -n test-database-users-operator postgresql-db-0 -- psql --user postgres --dbname test
```
```text
psql (15.0 (Debian 15.0-1.pgdg110+1))
Type "help" for help.
```

Roles:
```sql
\du john
```
```sql
            List of roles
 Role name | Attributes | Member of
-----------+------------+------------
 john      |            | {postgres}
```
Table privileges (query for more pretty result, you can consider using ```\z persons```):
```sql
SELECT privilege_type,
    table_catalog,
    table_name
FROM information_schema.role_table_grants
WHERE grantee = 'john';
```
```text
 privilege_type | table_catalog | table_name
----------------+---------------+------------
 INSERT         | test          | persons
(1 row)
```

Get user database privileges (query for more pretty result, you can consider using ```\l test```)
```sql
WITH tab1 AS (
  SELECT datname, (aclexplode(datacl)).grantee,
      (aclexplode(datacl)).privilege_type
  FROM pg_catalog.pg_database r
), tab2 AS (
  SELECT rolname, oid
  FROM pg_catalog.pg_roles
  WHERE rolname = 'john'
)
SELECT datname as database_name,
      rolname as role_name,
      privilege_type
FROM tab1 t1 INNER JOIN tab2 t2
ON t1.grantee = t2.oid;
```
```text
 database_name | role_name | privilege_type
---------------+-----------+----------------
 test          | john      | CREATE
(1 row)
```

### User deletetion
delete user CR:
```bash
kubectl delete --namespace test-database-users-operator users.auth.alex123012.com john
```
```text
user.auth.alex123012.com "john" deleted
```

And then once more check user privileges:
```bash
kubectl exec -ti -n test-database-users-operator postgresql-db-0 -- psql --user postgres --dbname test
```
```
psql (15.0 (Debian 15.0-1.pgdg110+1))
Type "help" for help.

test=# \du john
           List of roles
 Role name | Attributes | Member of
-----------+------------+-----------

test=# \z persons
                                  Access privileges
 Schema |  Name   | Type  |     Access privileges     | Column privileges | Policies
--------+---------+-------+---------------------------+-------------------+----------
 public | persons | table | postgres=arwdDxt/postgres |                   |
(1 row)

test=# \l test
                                              List of databases
 Name |  Owner   | Encoding |  Collate   |   Ctype    | ICU Locale | Locale Provider |   Access privileges
------+----------+----------+------------+------------+------------+-----------------+-----------------------
 test | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | =Tc/postgres         +
      |          |          |            |            |            |                 | postgres=CTc/postgres
(1 row)
```

All privileges and the user himself are removed

### Cleanup
Simply delete test namespace:
```bash
kubectl delete namespaces test-database-users-operator
```
```text
namespace "test-database-users-operator" deleted
```

Refer to [docs/examples/](docs/examples/) directory to check another DB types (CockroachDB) config and user CR
# Development

## Running on the cluster
* Install the CRDs into the cluster:

```sh
make install
```

* Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

* Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

* Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/database-users-operator:tag
```

* Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/database-users-operator:tag
```

## Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

## Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

# Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

## How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

## Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

# License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

# Helper
```bash
# Bash command for retrieving SSL certificates for user from default CockroachDB installation with operator
user=john
secret_name=john
for key in $(kubectl get secrets ${secret_name} -oyaml | yq '.data | keys | .[]'); do kubectl get secrets ${secret_name} -oyaml | key=$key yq '.data[strenv(key)]' | base64 -d | tee tmp/$(if [[ $key == "tls.key" ]]; then echo "client.${user}.key"; elif [[ $key == "tls.crt" ]]; then echo "client.${user}.crt"; else echo "ca.crt"; fi); done
```

# TODO
- [x] Auto remove user from all dbs listed in databaseConfig when User CR deleted
- [ ] Add webhook validation for config and user CR
- [ ] Fix CockroachDB database privileges
- [ ] Create events for user CR
- [ ] Create status updates for user CR
- [ ] Auto delete user from DB on databaseConfig entry remove from User CR
- [ ] Check compability with different postgres versions (only checked with PostgreSQL 15 and CockroachDB 22.1.9)
- [ ] Add MySQL support
- [ ] Add prometheus metrics and alerts
