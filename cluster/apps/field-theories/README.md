# Field Theories Blog Platform

Multi-component blog platform deployment consisting of the main blog site, BlueSky social integration, and Cloudflare Tunnel for external access.

## Overview

Field Theories is a personal blog built with Astro, featuring:
- **Main Blog Site**: Static site with dynamic content
- **BlueSky Sync (BlueSync)**: Automated posting and comment synchronization with BlueSky
- **Cloudflare Tunnel**: Secure public access via Cloudflare

## Architecture

```
┌─────────────────┐
│  Cloudflare     │  Public access via tunnel
└────────┬────────┘
         │
┌────────▼────────┐
│  Field Theories │  Astro blog site (port 4321)
│  Main Site      │  Exposed via Tailscale Funnel
└─────────────────┘

┌─────────────────┐
│  BlueSync       │  Rust service (port 3000)
│  API Server     │  ├─ Syncs RSS feed to BlueSky
└────────┬────────┘  └─ Syncs comments back
         │
┌────────▼────────┐
│  PostgreSQL     │  Shared database (postgres.postgres)
└─────────────────┘
```

## Access

- **Public**: `https://fieldtheories.blog` (via Cloudflare)
- **Tailscale Funnel**: `https://field-theories-homelab.feist-gondola.ts.net`
- **BlueSync API**: Internal only (port 3000)
- **BlueSky**: [@fieldtheories.blog](https://bsky.app/profile/fieldtheories.blog)

## Components

### 1. Main Blog Site (field-theories)

**Technology**: Astro static site generator
**Resources**:
- Memory: 64Mi request, 128Mi limit
- CPU: 50m request, 100m limit

**Features**:
- Tailscale Funnel for public access
- Auto-deployed on push via Flux image automation
- Read-only root filesystem for security

**Image**: `ghcr.io/t-eckert/field-theories/site`

### 2. BlueSky Sync (bluesync)

**Technology**: Rust Axum API server
**Resources**:
- Memory: 128Mi request, 256Mi limit
- CPU: 100m request, 200m limit

**Features**:
- Automatic RSS feed monitoring and posting to BlueSky
- Bidirectional comment synchronization
- Cron-based scheduling (configurable via env vars)
- Health check endpoint: `/health`
- PostgreSQL database for state management

**Image**: `ghcr.io/t-eckert/field-theories/bluesync`

**Required Secrets**:
```bash
BLUESKY_IDENTIFIER      # BlueSky account handle
BLUESKY_PASSWORD        # BlueSky app password
BLUESKY_MOCK           # Set to "true" for testing
DB_HOST                # postgres.postgres
DB_PORT                # 5432
DB_NAME                # bluesync
DB_USER                # postgres
DB_PASSWORD            # Database password
RSS_FEED_URL           # Blog RSS feed URL
API_PORT               # 3000
CORS_ORIGIN            # Allowed CORS origins
RSS_SYNC_CRON          # Cron schedule for RSS sync
COMMENT_SYNC_CRON      # Cron schedule for comment sync
```

### 3. Cloudflare Tunnel (cloudflared)

**Technology**: Cloudflare tunnel client
**Purpose**: Provides public HTTPS access without exposing ports

**Configuration**:
- Tunnel ID and credentials managed via secret
- Routes traffic to field-theories service

## Setup

### Prerequisites

- PostgreSQL database (uses existing `postgres.postgres`)
- BlueSky account with app password
- Cloudflare account with tunnel configured
- 1Password CLI for secrets management
- Flux CD with image automation enabled

### Initial Deployment

1. **Set up database**:
   ```bash
   # Database is automatically initialized via db-init-job
   kubectl apply -f cluster/apps/field-theories/namespace.yaml
   kubectl apply -f cluster/apps/field-theories/bluesync-db-init-job.yaml
   ```

2. **Configure secrets**:
   ```bash
   # BlueSky credentials
   kubectl create secret generic bluesync-secret -n field-theories \
     --from-literal=BLUESKY_IDENTIFIER="<handle>" \
     --from-literal=BLUESKY_PASSWORD="<app-password>" \
     --from-literal=BLUESKY_MOCK="false" \
     --from-literal=DB_HOST="postgres.postgres" \
     --from-literal=DB_PORT="5432" \
     --from-literal=DB_NAME="bluesync" \
     --from-literal=DB_USER="postgres" \
     --from-literal=DB_PASSWORD="<db-password>" \
     --from-literal=RSS_FEED_URL="https://fieldtheories.blog/rss.xml" \
     --from-literal=API_PORT="3000" \
     --from-literal=CORS_ORIGIN="*" \
     --from-literal=RSS_SYNC_CRON="0 * * * *" \
     --from-literal=COMMENT_SYNC_CRON="*/15 * * * *"

   # Cloudflare tunnel credentials
   kubectl create secret generic cloudflared-secret -n field-theories \
     --from-file=credentials.json=<path-to-tunnel-creds>
   ```

3. **Deploy services**:
   ```bash
   kubectl apply -f cluster/apps/field-theories/
   ```

4. **Verify deployment**:
   ```bash
   kubectl get pods -n field-theories
   kubectl logs -n field-theories -l app=bluesync
   kubectl logs -n field-theories -l app=field-theories
   ```

## Configuration

### Automatic Image Updates

Flux monitors for new images and automatically updates deployments:
- **Main Site**: Tags matching `YYYYMMDD-HHMM-sha-*` pattern
- **BlueSync**: Latest tag

Image policies in `fieldtheories-image-policy.yaml` and `bluesync-image-policy.yaml`

### Network Policy

The namespace has network policies configured to control traffic flow between components.

### Cron Schedules

**RSS Sync**: `0 * * * *` (hourly)
- Checks blog RSS feed for new posts
- Creates BlueSky posts for new articles
- Stores post mapping in database

**Comment Sync**: `*/15 * * * *` (every 15 minutes)
- Fetches replies/comments from BlueSky
- Stores in database for blog to query

## Maintenance

### Check Service Status

```bash
# All pods
kubectl get pods -n field-theories

# BlueSync logs
kubectl logs -n field-theories -l app=bluesync -f

# Main site logs
kubectl logs -n field-theories -l app=field-theories -f

# Cloudflared tunnel logs
kubectl logs -n field-theories -l app=cloudflared -f
```

### Database Access

```bash
# Connect to BlueSync database
task db:access DB=bluesync

# Or directly
kubectl exec -it -n postgres deployment/postgres -- psql -U postgres -d bluesync
```

### Restart Services

```bash
# Restart all
kubectl rollout restart deployment -n field-theories

# Restart specific component
kubectl rollout restart deployment/bluesync -n field-theories
kubectl rollout restart deployment/field-theories -n field-theories
kubectl rollout restart deployment/cloudflared -n field-theories
```

### Trigger Manual Sync

```bash
# Restart BlueSync to trigger immediate sync
kubectl rollout restart deployment/bluesync -n field-theories
```

### Check Image Automation Status

```bash
kubectl get imagepolicy -n field-theories
kubectl get imagerepository -n field-theories
```

## Troubleshooting

### BlueSync Not Posting to BlueSky

Check logs for errors:
```bash
kubectl logs -n field-theories -l app=bluesync --tail=100
```

Common issues:
- Invalid BlueSky credentials: Check `bluesync-secret`
- Database connection failure: Verify postgres.postgres is accessible
- RSS feed not reachable: Check `RSS_FEED_URL` setting
- BLUESKY_MOCK set to "true": Change to "false" for production

### Site Not Accessible

Check Tailscale Funnel status:
```bash
kubectl describe pod -n field-theories -l app=field-theories
kubectl get events -n field-theories --sort-by='.lastTimestamp'
```

Check Cloudflare tunnel:
```bash
kubectl logs -n field-theories -l app=cloudflared
```

### Database Issues

Test database connectivity:
```bash
kubectl run -it --rm pg-test --image=postgres:16-alpine --restart=Never -n field-theories -- \
  psql -h postgres.postgres -U postgres -d bluesync
```

Check database initialization:
```bash
kubectl get jobs -n field-theories
kubectl logs -n field-theories job/bluesync-db-init
```

### Image Updates Not Applying

Check Flux image automation:
```bash
kubectl get imagepolicy -n field-theories -o yaml
kubectl logs -n flux-system deployment/image-automation-controller
```

Force image rescan:
```bash
kubectl delete imagerepository -n field-theories --all
kubectl apply -f cluster/apps/field-theories/fieldtheories-image-repository.yaml
kubectl apply -f cluster/apps/field-theories/bluesync-image-repository.yaml
```

## Security Notes

- All services run as non-root user (UID 1000)
- Read-only root filesystems enforced
- All capabilities dropped from containers
- Seccomp profiles applied
- Secrets managed via Kubernetes secrets (not committed to git)
- BlueSky credentials use app passwords (not main password)
- Cloudflare tunnel provides zero-trust access without exposing ports
- Network policies restrict inter-pod communication

## Development

### Local Testing

Set `BLUESKY_MOCK=true` in bluesync-secret to prevent actual BlueSky posts during testing.

### Updating the Blog

1. Push changes to field-theories repository
2. GitHub Actions builds and pushes new image
3. Flux detects new image tag
4. Deployment automatically updated

### Updating BlueSync

1. Push changes to bluesync in field-theories repository
2. GitHub Actions builds and pushes new image
3. Flux detects latest tag
4. Deployment automatically updated

## References

- [Field Theories Blog](https://fieldtheories.blog)
- [BlueSky](https://bsky.app/profile/fieldtheories.blog)
- [Astro Documentation](https://docs.astro.build)
- [Cloudflare Tunnel Docs](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [Tailscale Funnel](https://tailscale.com/kb/1223/funnel/)
