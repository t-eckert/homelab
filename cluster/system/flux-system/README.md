# Flux CD GitOps System

Flux CD deployment for continuous delivery and automated image updates in the homelab cluster.

## Overview

Flux CD provides:
- **GitOps**: Cluster state managed via Git repository
- **Automated Deployments**: Automatically applies changes from Git
- **Image Automation**: Monitors container registries and updates deployments
- **Reconciliation**: Ensures cluster matches Git state

## Architecture

```
┌─────────────────┐
│  Git Repository │  Source of truth
│  (this repo)    │  cluster/ directory
└────────┬────────┘
         │
┌────────▼───────────────┐
│  Flux Controllers      │
│  ├─ Source Controller  │  Syncs Git repo
│  ├─ Kustomize Ctrl     │  Applies manifests
│  ├─ Helm Controller    │  (not used)
│  └─ Image Controllers  │  Updates images
└────────┬───────────────┘
         │
┌────────▼────────┐
│  Cluster State  │  Running workloads
└─────────────────┘
```

## Components

### Source Controller
Monitors Git repository for changes every 1 minute.

### Kustomize Controller
Reconciles cluster state every 10 minutes.
Path: `./cluster` (root of manifests)

### Image Automation Controller
- Updates deployment image tags
- Creates Git commits
- Pushes to main branch

### Image Reflector Controller
- Scans container registries
- Tracks latest image tags based on policies

## Configuration

### Git Repository Sync

**Manifest**: `gotk-sync.yaml`
- **URL**: `ssh://git@github.com/t-eckert/homelab`
- **Branch**: `main`
- **Interval**: 1 minute
- **Path**: `./cluster`

### Image Update Automation

**Manifest**: `image-update-automation.yaml`
- **Interval**: 5 minutes
- **Path**: `./cluster`
- **Commit Author**: fluxcdbot
- **Strategy**: Setters (using image policy markers)

## Image Policies

Services with automated image updates have:

1. **ImageRepository**: Monitors container registry
2. **ImagePolicy**: Defines tag selection strategy
3. **Image Policy Marker**: In deployment YAML

Example:
```yaml
image: ghcr.io/user/app:v1.2.3 # {"$imagepolicy": "namespace:policy-name"}
```

## Managing Flux

### Check Flux Status

```bash
# All Flux resources
kubectl get all -n flux-system

# Kustomizations
kubectl get kustomization -n flux-system

# Git repository sync
kubectl get gitrepository -n flux-system

# Image policies
kubectl get imagepolicy -A
```

### View Flux Logs

```bash
# Source controller (Git sync)
kubectl logs -n flux-system deployment/source-controller -f

# Kustomize controller (reconciliation)
kubectl logs -n flux-system deployment/kustomize-controller -f

# Image automation (updates)
kubectl logs -n flux-system deployment/image-automation-controller -f

# Image reflector (registry scanning)
kubectl logs -n flux-system deployment/image-reflector-controller -f
```

### Force Reconciliation

```bash
# Force Git sync
flux reconcile source git flux-system

# Force kustomization
flux reconcile kustomization flux-system

# Force image repository scan
flux reconcile image repository <name> -n <namespace>
```

### Suspend/Resume Automation

```bash
# Suspend image automation
flux suspend image update flux-system

# Resume image automation
flux resume image update flux-system
```

## Image Update Workflow

1. New image pushed to registry (e.g., ghcr.io)
2. **Image Reflector** scans registry (per ImageRepository)
3. **Image Policy** evaluates tags against rules
4. If newer image found, **Image Automation** updates deployment YAML
5. **Git commit** created with updated image tag
6. **Pushed to main** branch
7. **Source Controller** detects change
8. **Kustomize Controller** applies updated manifest
9. **Deployment** rolls out new image

## Adding Image Automation to Service

1. **Create ImageRepository**:
   ```yaml
   apiVersion: image.toolkit.fluxcd.io/v1beta2
   kind: ImageRepository
   metadata:
     name: myapp
     namespace: myapp
   spec:
     image: ghcr.io/user/myapp
     interval: 5m
   ```

2. **Create ImagePolicy**:
   ```yaml
   apiVersion: image.toolkit.fluxcd.io/v1beta2
   kind: ImagePolicy
   metadata:
     name: myapp
     namespace: myapp
   spec:
     imageRepositoryRef:
       name: myapp
     policy:
       semver:
         range: '>=1.0.0'
   ```

3. **Add marker to deployment**:
   ```yaml
   spec:
     containers:
     - name: myapp
       image: ghcr.io/user/myapp:v1.0.0 # {"$imagepolicy": "myapp:myapp"}
   ```

## Troubleshooting

### Flux Not Syncing

Check Git repository status:
```bash
kubectl describe gitrepository flux-system -n flux-system
```

Common issues:
- SSH key invalid: Check deploy key in GitHub
- Repository not accessible: Verify URL
- Branch doesn't exist: Check branch name

### Image Updates Not Working

Check image policies:
```bash
kubectl get imagepolicy -A
kubectl describe imagepolicy <name> -n <namespace>
```

Common issues:
- ImageRepository not scanning: Check credentials/access
- Policy not selecting correct tags: Review policy rules
- Marker syntax incorrect: Verify `{"$imagepolicy": "..."}` format
- Image automation suspended: Check `flux get image update`

### Kustomization Failing

View errors:
```bash
kubectl describe kustomization flux-system -n flux-system
```

Common issues:
- Invalid YAML: Check manifests for syntax errors
- Resource conflicts: Duplicate names or conflicts
- Dependencies not met: Check resource ordering

### Git Push Failures

Check image automation logs:
```bash
kubectl logs -n flux-system deployment/image-automation-controller
```

Common issues:
- No write access to repository: Check deploy key permissions
- Branch protected: Adjust branch protection rules
- Conflicts: Manual intervention needed

## Manual Image Updates

If automation isn't enabled, use the update script:

```bash
# Check for available updates
task images:check

# Update and commit locally
task images:update

# Update, commit, and push
task images:update-and-push
```

## Security

- Deploy key stored in `flux-system` secret (not in Git)
- Read-only Git access for Source Controller
- Write access for Image Automation (via different key)
- No secrets committed to repository

## Maintenance

### Update Flux Components

```bash
# Using Flux CLI
flux install --export > cluster/system/flux-system/gotk-components.yaml

# Or automatic update via Flux itself (recommended)
flux create source helm flux \
  --url=https://fluxcd-community.github.io/helm-charts \
  --namespace=flux-system

flux create helmrelease flux \
  --source=HelmRepository/flux \
  --chart=flux2 \
  --namespace=flux-system
```

### Backup Flux Configuration

```bash
# Export all Flux resources
flux export source git --all > flux-git-sources.yaml
flux export kustomization --all > flux-kustomizations.yaml
flux export image repository --all > flux-image-repos.yaml
flux export image policy --all > flux-image-policies.yaml
```

## Best Practices

1. **Always commit manifests to Git** before applying manually
2. **Use image policies** for automated updates
3. **Monitor Flux logs** for reconciliation errors
4. **Tag images with semantic versioning** for better policy control
5. **Test changes in separate branch** before merging to main

## References

- [Flux Documentation](https://fluxcd.io/docs/)
- [Image Automation Guide](https://fluxcd.io/docs/guides/image-update/)
- [Flux CLI](https://fluxcd.io/docs/cmd/)
- [Troubleshooting](https://fluxcd.io/docs/troubleshooting/)
