```yaml
---
apiVersion: databaseusersoperator.com/v1alpha1
kind: User
metadata:
  name: john
  namespace: test-database-users-operator
passwordSecret:
  secret:
    name: postgres-john
    namespace: test-database-users-operator
  key: password
```
