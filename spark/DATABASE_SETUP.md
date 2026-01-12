# PostgreSQL Database Setup for Spark

## Overview

Spark uses a shared PostgreSQL instance with a dedicated `spark` user that has limited permissions. Each spark gets its own database within this instance.

## Prerequisites

- PostgreSQL instance running (postgres-homelab.feist-gondola.ts.net)
- Access to postgres superuser account
- kubectl access to homelab cluster

## Initial Setup

### 1. Create Dedicated Spark User

Connect to your PostgreSQL instance as the postgres superuser:

```bash
# Using kubectl port-forward if postgres is in cluster
kubectl port-forward -n postgres svc/postgres 5432:5432

# Or connect directly via Tailscale
psql "postgresql://postgres:<password>@postgres-homelab.feist-gondola.ts.net:5432/homelab"
```

Create the spark user with limited permissions:

```sql
-- Create the spark user
CREATE USER spark WITH PASSWORD 'your_secure_password_here';

-- Grant permission to create databases (required for spark create)
ALTER USER spark CREATEDB;

-- Grant connect to the homelab database
GRANT CONNECT ON DATABASE homelab TO spark;

-- Verify the user was created
\du spark
```

The spark user should show:
- `Create DB` privilege enabled
- NOT a Superuser
- NOT able to create roles

### 2. Update Spark CLI Configuration

Update the Kubernetes secret with the spark user credentials:

```bash
# Edit the secret
kubectl edit secret spark-cli-config -n spark
```

Update the base64-encoded values:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: spark-cli-config
  namespace: spark
stringData:
  ANTHROPIC_API_KEY: "sk-ant-..."
  POSTGRES_PASSWORD: "your_spark_user_password"  # Spark user password, not postgres password
  GITHUB_TOKEN: "github_pat_..."
```

Or use the secret template and load script:

```bash
cd cluster/spark

# Copy template
cp secret.yaml.template secret.yaml

# Edit with actual values
vim secret.yaml

# Apply
kubectl apply -f secret.yaml

# Load environment variables for CLI use
source load-env.sh
```

### 3. Test Connection

Test that the spark user can connect and create databases:

```bash
# Set environment variables
export POSTGRES_PASSWORD="your_spark_user_password"
export POSTGRES_HOST="postgres-homelab.feist-gondola.ts.net"
export POSTGRES_PORT="5432"
export POSTGRES_USER="spark"
export POSTGRES_DB="homelab"

# Test connection (should work)
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c '\l'

# Test database creation (should work)
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c 'CREATE DATABASE test_spark;'

# Verify database exists
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c '\l' | grep test_spark

# Clean up test
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c 'DROP DATABASE test_spark;'
```

## Security Considerations

### What the Spark User CAN Do

- Create new databases
- Connect to the homelab database
- Drop databases it created

### What the Spark User CANNOT Do

- Access other users' databases (unless explicitly granted)
- Create new PostgreSQL users
- Modify system catalogs
- Change server configuration
- Access postgres superuser privileges

### Password Management

1. **Generate Strong Password**: Use a password manager or:
   ```bash
   openssl rand -base64 32
   ```

2. **Store Securely**: Keep password in:
   - Kubernetes secret (base64 encoded)
   - 1Password or other password manager
   - Never commit to git

3. **Rotate Regularly**: Update password quarterly:
   ```sql
   ALTER USER spark PASSWORD 'new_password';
   ```
   Then update Kubernetes secret.

## Troubleshooting

### Permission Denied Errors

If you see `permission denied to create database`:

```sql
-- As postgres superuser, grant CREATEDB
ALTER USER spark CREATEDB;
```

### Connection Refused

Check:
1. PostgreSQL is running: `kubectl get pods -n postgres`
2. Service is exposed: `kubectl get svc -n postgres`
3. Tailscale is connected: `tailscale status`
4. Password is correct (no special characters causing issues)

### Database Already Exists

If spark creation fails with "database already exists":

```bash
# List all databases
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c '\l'

# Drop the existing database
spark delete <spark-name>

# Or manually:
psql "postgresql://spark:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}" -c 'DROP DATABASE "spark-name";'
```

## Migration from Postgres User

If you're currently using the postgres superuser and want to migrate to the spark user:

1. Create the spark user as described above
2. Update the Kubernetes secret with new credentials
3. Reload environment: `source load-env.sh`
4. Test new configuration: `spark create --test` (if test flag exists)
5. Existing sparks will continue to work (they use their own database connections)
6. New sparks will use the spark user

No need to recreate existing spark instances - they connect to their individual databases directly.

## References

- PostgreSQL User Management: https://www.postgresql.org/docs/current/user-manag.html
- Database Creation Privileges: https://www.postgresql.org/docs/current/sql-createdatabase.html
- Kubernetes Secrets: https://kubernetes.io/docs/concepts/configuration/secret/
