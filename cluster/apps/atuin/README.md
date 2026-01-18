# Atuin Shell History Sync Server

Atuin server deployment for syncing shell history across machines.

## Overview

Atuin is a shell history sync server that provides:
- Encrypted shell history synchronization
- Full-text search across command history
- Statistics and analytics
- Cross-platform support (bash, zsh, fish)

## Access

- **Tailscale**: `http://atuin-homelab.feist-gondola.ts.net:8888`
- **Internal**: `http://atuin.atuin:8888` (from within cluster)

## Setup

### Prerequisites

- PostgreSQL database (uses existing `postgres.postgres`)
- 1Password CLI for secrets management
- Tailscale operator configured in cluster

### Initial Deployment

1. **Configure secrets**:
   ```bash
   # Get password from 1Password
   export POSTGRES_PASSWORD=$(op read "op://Homelab/Homelab Postgres/password")

   # Update secret.yaml
   sed -i.bak "s|PASSWORD|$POSTGRES_PASSWORD|g" cluster/atuin/secret.yaml
   sed -i.bak "s|REPLACE_WITH_PASSWORD|$POSTGRES_PASSWORD|g" cluster/atuin/db-init-job.yaml
   rm cluster/atuin/*.bak
   ```

2. **Apply manifests**:
   ```bash
   kubectl apply -f cluster/atuin/namespace.yaml
   kubectl apply -f cluster/atuin/secret.yaml
   kubectl apply -f cluster/atuin/db-init-job.yaml
   kubectl apply -f cluster/atuin/deployment.yaml
   kubectl apply -f cluster/atuin/service.yaml
   ```

3. **Verify deployment**:
   ```bash
   kubectl get all -n atuin
   kubectl logs -n atuin deployment/atuin
   ```

## Configuration

### Server Settings

- **Open Registration**: Enabled
- **Max History Length**: 8192 entries
- **Page Size**: 100 entries
- **Port**: 8888

### Resources

- **Requests**: 128Mi memory, 100m CPU
- **Limits**: 512Mi memory, 500m CPU

### Database

- **Database Name**: `atuin`
- **Host**: `postgres.postgres:5432`
- **User**: `postgres`

## Client Setup

To use Atuin with this server:

1. **Install Atuin client**:
   ```bash
   # macOS
   brew install atuin

   # Linux
   bash <(curl --proto '=https' --tlsv1.2 -sSf https://setup.atuin.sh)
   ```

2. **Configure client**:
   ```bash
   # Set server URL
   atuin config set sync_address http://atuin-homelab.feist-gondola.ts.net:8888

   # Register account
   atuin register -u <username> -e <email> -p <password>

   # Or login if account exists
   atuin login -u <username> -p <password>
   ```

3. **Enable shell integration**:
   ```bash
   # For zsh (add to ~/.zshrc)
   eval "$(atuin init zsh)"

   # For bash (add to ~/.bashrc)
   eval "$(atuin init bash)"
   ```

4. **Sync history**:
   ```bash
   # Upload existing history
   atuin sync
   ```

## Maintenance

### Check Server Status

```bash
kubectl get pods -n atuin
kubectl logs -n atuin deployment/atuin
```

### Access Database

```bash
# Connect to PostgreSQL
kubectl exec -it -n postgres deployment/postgres -- psql -U postgres -d atuin
```

### View Tailscale Status

```bash
kubectl get svc -n atuin atuin-external
```

### Restart Server

```bash
kubectl rollout restart deployment/atuin -n atuin
```

## Troubleshooting

### Pod Crashes on Startup

Check logs for errors:
```bash
kubectl logs -n atuin -l app=atuin
```

Common issues:
- Database connection failure: Verify postgres.postgres is accessible
- Config file errors: Check secret.yaml configuration
- Permission issues: Ensure namespace has privileged pod security

### Database Connection Issues

Test database connectivity:
```bash
kubectl run -it --rm pg-test --image=postgres:16-alpine --restart=Never -n atuin -- \
  psql -h postgres.postgres -U postgres -d atuin
```

### Tailscale Not Working

Check Tailscale service:
```bash
kubectl describe svc -n atuin atuin-external
kubectl get pods -n tailscale
```

## Security Notes

- Server uses secure PostgreSQL connection with password authentication
- Secrets managed via Kubernetes secrets (not committed to git)
- Shell history is encrypted end-to-end by Atuin clients
- Tailscale provides secure access without exposing to public internet
- Pod security context enforces non-root user and restricted capabilities

## References

- [Atuin Documentation](https://atuin.sh)
- [Atuin GitHub](https://github.com/atuinsh/atuin)
- [Atuin Server Configuration](https://docs.atuin.sh/self-hosting/server-setup/)
