### **In active development**
# Database Users Kubernetes Operator

Kubernetes operator to create and manage users and roles for various SQL and NoSQL databases (currently supports PostgreSQL, MySQL and CockroachDB). This repository contains a [custom controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) and [custom resource definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) designed for the lifecycle (creation, update privileges, deletion) of a different databases users/roles.

# Features
* Currently supports roles, table privilegesa and database privileges for `PostgreSQL`, `MySQL` and `CockroachDB`.
* Create users/roles and assign privileges to them in databases.
* Change users/roles privileges in databases in runtime.
* Delete user/role in databases when custom resource is deleted.

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

It will install **database-users-operator** into `database-users-operator-system` namespace.

## Operator installation process
```text
namespace/database-users-operator-system created
customresourcedefinition.apiextensions.k8s.io/databases.databaseusersoperator.com created
customresourcedefinition.apiextensions.k8s.io/privileges.databaseusersoperator.com created
customresourcedefinition.apiextensions.k8s.io/users.databaseusersoperator.com created
serviceaccount/database-users-operator-controller-manager created
role.rbac.authorization.k8s.io/database-users-operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/database-users-operator-manager-role created
clusterrole.rbac.authorization.k8s.io/database-users-operator-metrics-reader created
clusterrole.rbac.authorization.k8s.io/database-users-operator-proxy-role created
rolebinding.rbac.authorization.k8s.io/database-users-operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/database-users-operator-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/database-users-operator-proxy-rolebinding created
service/database-users-operator-controller-manager-metrics-service created
deployment.apps/database-users-operator-controller-manager created
```

Check `database-users-operator-system` is running:
```bash
kubectl get pods -n database-users-operator-system
```

```text
NAME                                                          READY   STATUS    RESTARTS   AGE
database-users-operator-controller-manager-777dcc4765-nb76m   1/1     Running   0          36s
```
# Documentation

Review [docs/](docs/) folder for more information.

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

Create issue or PR and tag `@alex123012`.

To run e2e tests locally:
```bash
kind create cluster --name e2e-tests --image kindest/node:v1.2
6.6
make prepare-kind
make run-e2e
```

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
mkdir -p "${HOME}/.cockroach-certs/"
for key in $(kubectl get secrets ${secret_name} -oyaml | yq '.data | keys | .[]'); do kubectl get secrets ${secret_name} -oyaml | key=$key yq '.data[strenv(key)]' | base64 -d | tee "${HOME}/.cockroach-certs/"$(if [[ $key == "tls.key" ]]; then echo "client.${user}.key"; elif [[ $key == "tls.crt" ]]; then echo "client.${user}.crt"; else echo "ca.crt"; fi); done
```

# TODO
- [x] Add E2E tests.
- [x] Create status updates for user CR.
- [ ] Add webhook validation for config and user CR (partially done).
- [x] Create events for user CR.
- [ ] Auto delete user from DB on `database` entry remove from User CR.
- [ ] Add prometheus metrics and alerts.
