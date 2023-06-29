```yaml
---
apiVersion: databaseusersoperator.com/v1alpha1
kind: Privileges
metadata:
  name: some-privileges
  namespace: test-database-users-operator
# List of privileges, required.
privileges:
    # Table privilege.
  - database: some_db
    "on": some_table
    privilege: ALL PRIVILEGES
    # Database privilege.
  - database: postgres
    privilege: CONNECT
    # Role privilege.
  - privilege: some_role
```
