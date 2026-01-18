# Homelab Scripts

Automation scripts for managing the homelab Kubernetes cluster.

## Image Update Automation

### Overview

The `update-images.sh` script provides automated container image updates based on Flux ImagePolicy resources. This is a workaround for Flux ImageUpdateAutomation not automatically committing changes to the repository.

### Usage

#### Via Task (Recommended)

```bash
# Check for available updates (dry-run)
task images:check

# Update images and commit locally
task images:update

# Update, commit, and push to remote
task images:update-and-push

# Show status of all ImagePolicy resources
task images:status
```

#### Direct Script Execution

```bash
# Dry-run mode
DRY_RUN=true ./scripts/update-images.sh

# Update and commit
./scripts/update-images.sh

# Update, commit, and push
AUTO_PUSH=true ./scripts/update-images.sh
```

### How It Works

1. **Queries Flux ImagePolicy resources** across all namespaces
2. **Checks readiness** - skips policies that aren't ready
3. **Finds deployment manifests** with `{"$imagepolicy": "namespace:name"}` markers
4. **Compares versions** - current image tag vs latest from policy
5. **Updates manifests** when newer images are available
6. **Creates git commits** with detailed change summaries

### Requirements

- `kubectl` - Kubernetes CLI
- `jq` - JSON processor
- `git` - Version control
- Access to the Kubernetes cluster with Flux installed
- Flux ImagePolicy resources configured

### Example Output

```bash
$ task images:check
INFO: Fetching ImagePolicy resources...
INFO: Processing field-theories/field-theories -> ghcr.io/t-eckert/field-theories/site:20260117-2124-sha-27bf745
INFO: Updating ./cluster/field-theories/deployment.yaml
INFO:   FROM: ghcr.io/t-eckert/field-theories/site:20260115-0509-sha-f048662
INFO:   TO:   ghcr.io/t-eckert/field-theories/site:20260117-2124-sha-27bf745
WARNING: [DRY RUN] Would update ./cluster/field-theories/deployment.yaml

SUCCESS: Updated 1 file(s)
WARNING: DRY RUN mode - no changes committed
```

### Configuration

Environment variables:

- `DRY_RUN` - Set to `true` to preview changes without modifying files (default: `false`)
- `AUTO_PUSH` - Set to `true` to automatically push commits to remote (default: `false`)
- `CLUSTER_DIR` - Path to cluster manifests directory (default: `./cluster`)

### Integration with Flux

This script complements Flux's existing image automation:

- **ImageRepository** - Scans container registries for new images
- **ImagePolicy** - Determines which image tag to use based on policy
- **This Script** - Updates manifests and commits changes (replaces ImageUpdateAutomation)

The script respects Flux's `{"$imagepolicy": "namespace:name"}` marker format, ensuring compatibility with Flux's update detection system.

### Troubleshooting

**No files found with marker**
- Ensure deployment manifests have the correct `{"$imagepolicy": "namespace:name"}` marker
- Check that the ImagePolicy name and namespace match exactly

**ImagePolicy not ready**
- Run `task images:status` to see policy status
- Check ImageRepository and ImagePolicy resources with `flux get images all`

**Script can't find images**
- Verify ImagePolicy has `status.latestRef.name` and `status.latestRef.tag` set
- Check that the ImageRepository is scanning successfully
