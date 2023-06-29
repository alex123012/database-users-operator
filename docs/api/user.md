```yaml
---
apiVersion: databaseusersoperator.com/v1alpha1
kind: User
metadata:
  name: username
spec:
	# List of databases, where user needs to be created with configs for it.
  databases:
	    # The name of the Database CR to create user in, required.
    - name: database-cr-name
      # Reference to secret with password for user in the database, not required.
      passwordSecret:
        # Secret key with password, required.
        key: secret-key
        # Secret name and namespace, required.
        secret:
	        # Secret name, required.
          name: secret-name
	        # Secret namespace, required.
          namespace: secret-namespace

      # If operator would create data for user (for example for postgres with sslMode=="verify-full"),
      # it is reference to non-existed Secret, that will be created during user creation in the database, not required.
      createdSecret:
        # Secret name, required.
        name: future-created-secret-name
        # Secret namespace, required.
        namespace: future-created-secret-namespace
	    # List of references to Privileges CR, that will be applied to created user in the database, required.
      privileges:
        # Name of the Privileges CR, required.
      - name: privilege-cr-name-first
      - name: privilege-cr-name-second

    - name: another-database-cr-name
      passwordSecret:
        key: secret-key
        secret:
          name: secret-name
          namespace: secret-namespace
      createdSecret:
        name: future-created-secret-name
        namespace: future-created-secret-namespace
      privileges:
      - name: privilege-cr-name-first
      - name: privilege-cr-name-second

```
