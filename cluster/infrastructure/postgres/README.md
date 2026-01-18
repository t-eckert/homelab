# PostgreSQL Database Server

Shared PostgreSQL database server providing database services for multiple applications across the homelab.

## Overview

PostgreSQL 16 deployment serving as centralized database for:
- **Atuin**: Shell history sync database
- **BlueSync**: Field Theories BlueSky integration data
- **Grafana**: Dashboard and user data (datasource only)
- **Other applications**: General-purpose database server

## Architecture

```
┌─────────────────────────┐
│  PostgreSQL Server      │
│  Port: 5432             │
│  Version: 16-alpine     │
│  Storage: 10Gi PVC      │
└────────┬────────────────┘
         │
    ┌────▼──────────────┐
    │  Databases:       │
    │  ├─ homelab       │ (default)
    │  ├─ atuin         │
    │  └─ bluesync      │
    └───────────────────┘
```

## Access

- **Internal**: `postgres.postgres:5432`
- **From host**: Port-forward or use Taskfile commands
- **External**: Not exposed (internal only for security)

## Configuration

**Image**: `postgres:16-alpine`

**Resources**:
- Memory: 512Mi request, 1Gi limit
- CPU: 250m request, 500m limit

**Storage**:
- 10Gi PersistentVolumeClaim
- Mount: `/var/lib/postgresql/data`

**Environment**:
- `POSTGRES_PASSWORD`: Set via secret (from 1Password)
- `POSTGRES_USER`: postgres (default superuser)
- `POSTGRES_DB`: homelab (default database)

## Setup

### Prerequisites

- 1Password CLI for secret management
- Sufficient PV storage available (10Gi)

### Initial Deployment

1. **Create secret**:
   ```bash
   # Get password from 1Password
   export POSTGRES_PASSWORD=$(op read "op://Homelab/Homelab Postgres/password")

   # Create secret
   kubectl create secret generic postgres-credentials \
     -n postgres \
     --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD"
   ```

2. **Deploy PostgreSQL**:
   ```bash
   kubectl apply -f cluster/infrastructure/postgres/namespace.yaml
   kubectl apply -f cluster/infrastructure/postgres/
   ```

3. **Verify deployment**:
   ```bash
   kubectl get pods -n postgres
   kubectl logs -n postgres deployment/postgres
   ```

### Creating Application Databases

Databases are typically created via init jobs in application namespaces. Example:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: app-db-init
  namespace: app-namespace
spec:
  template:
    spec:
      containers:
      - name: db-init
        image: postgres:16-alpine
        command:
        - sh
        - -c
        - |
          PGPASSWORD=$DB_PASSWORD psql -h postgres.postgres -U postgres -c "CREATE DATABASE IF NOT EXISTS myapp;"
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: DB_PASSWORD
      restartPolicy: OnFailure
```

## Database Management

### Access Database via Taskfile

```bash
# Connect to default database
task db:access

# Connect to specific database
task db:access DB=atuin
task db:access DB=bluesync
```

### Get Connection String

```bash
# Get connection string for application
task db:conn DB=homelab
```

### Direct Access via kubectl

```bash
# Interactive psql session
kubectl exec -it -n postgres deployment/postgres -- \
  psql -U postgres -d homelab

# Run single query
kubectl exec -it -n postgres deployment/postgres -- \
  psql -U postgres -d homelab -c "SELECT version();"
```

### List All Databases

```bash
kubectl exec -it -n postgres deployment/postgres -- \
  psql -U postgres -c "\l"
```

### List Database Sizes

```bash
kubectl exec -it -n postgres deployment/postgres -- \
  psql -U postgres -c "
    SELECT
      pg_database.datname,
      pg_size_pretty(pg_database_size(pg_database.datname)) AS size
    FROM pg_database
    ORDER BY pg_database_size(pg_database.datname) DESC;
  "
```

## Backup and Restore

### Create Backup

```bash
# Backup specific database
kubectl exec -n postgres deployment/postgres -- \
  pg_dump -U postgres mydb > backup_mydb_$(date +%Y%m%d).sql

# Backup all databases
kubectl exec -n postgres deployment/postgres -- \
  pg_dumpall -U postgres > backup_all_$(date +%Y%m%d).sql
```

### Restore from Backup

```bash
# Restore database
cat backup_mydb.sql | kubectl exec -i -n postgres deployment/postgres -- \
  psql -U postgres -d mydb
```

### Automated Backups

Consider setting up a CronJob for regular backups:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: postgres
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16-alpine
            command:
            - sh
            - -c
            - pg_dumpall -U postgres > /backups/backup_$(date +\%Y\%m\%d).sql
            volumeMounts:
            - name: backups
              mountPath: /backups
          volumes:
          - name: backups
            persistentVolumeClaim:
              claimName: postgres-backups
          restartPolicy: OnFailure
```

## Monitoring

### Check Database Status

```bash
# Connection count
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Active queries
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "SELECT pid, usename, application_name, state, query FROM pg_stat_activity WHERE state = 'active';"
```

### Storage Usage

```bash
# Check PVC usage
kubectl exec -n postgres deployment/postgres -- df -h /var/lib/postgresql/data

# Table sizes in database
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -d mydb -c "
    SELECT
      schemaname,
      tablename,
      pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    LIMIT 10;
  "
```

### Resource Usage

```bash
kubectl top pod -n postgres
```

## Maintenance

### Restart PostgreSQL

```bash
kubectl rollout restart deployment/postgres -n postgres
```

### Vacuum Database

```bash
# Analyze and vacuum
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -d mydb -c "VACUUM ANALYZE;"
```

### Update Statistics

```bash
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -d mydb -c "ANALYZE;"
```

## Troubleshooting

### Connection Refused

Check if pod is running:
```bash
kubectl get pods -n postgres
kubectl logs -n postgres deployment/postgres
```

Check if service is correctly configured:
```bash
kubectl get svc -n postgres
kubectl describe svc postgres -n postgres
```

### Out of Connections

Check max connections:
```bash
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "SHOW max_connections;"
```

List current connections:
```bash
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "SELECT * FROM pg_stat_activity;"
```

### Storage Full

Check PVC usage:
```bash
kubectl exec -n postgres deployment/postgres -- df -h /var/lib/postgresql/data
```

Expand PVC:
```bash
kubectl edit pvc postgres-pvc -n postgres
# Increase storage size
```

Clean up old data or databases:
```bash
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "DROP DATABASE old_database;"
```

### Performance Issues

Enable logging for slow queries by editing deployment:
```yaml
env:
- name: POSTGRES_LOG_MIN_DURATION_STATEMENT
  value: "1000"  # Log queries taking >1s
```

Check for long-running queries:
```bash
kubectl exec -n postgres deployment/postgres -- \
  psql -U postgres -c "
    SELECT pid, now() - query_start as duration, query
    FROM pg_stat_activity
    WHERE state = 'active'
    ORDER BY duration DESC;
  "
```

## Security Notes

- Password managed via Kubernetes secret (not in git)
- Service is ClusterIP only (not exposed externally)
- No network policies currently configured
- Consider adding network policies to restrict access to specific namespaces
- Run as postgres user (UID 999)
- Database files stored on PVC with appropriate permissions

## Client Databases

### Atuin

- **Database**: `atuin`
- **Purpose**: Shell history synchronization
- **Schema**: Managed by Atuin server
- **Init**: `cluster/apps/atuin/db-init-job.yaml`

### BlueSync

- **Database**: `bluesync`
- **Purpose**: Field Theories BlueSky integration
- **Schema**: Post mappings, comment data
- **Init**: `cluster/apps/field-theories/bluesync-db-init-job.yaml`

### Adding New Databases

1. Create database via init job in application namespace
2. Configure application to connect to `postgres.postgres:5432`
3. Use Kubernetes secrets for credentials
4. Document in this README

## Performance Tuning

### Increase Shared Buffers

```yaml
env:
- name: POSTGRES_SHARED_BUFFERS
  value: "256MB"
```

### Adjust Work Memory

```yaml
env:
- name: POSTGRES_WORK_MEM
  value: "16MB"
```

### Connection Pooling

Consider deploying PgBouncer for connection pooling if needed.

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/16/)
- [PostgreSQL Docker Image](https://hub.docker.com/_/postgres)
- [Kubernetes StatefulSet Best Practices](https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/)
