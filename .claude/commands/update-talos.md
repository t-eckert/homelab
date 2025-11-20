# Update Talos

Guide the user through updating Talos Linux on the homelab cluster. This includes updating the talosctl CLI and upgrading the Talos OS on the cluster nodes.

## Instructions for Claude

When this command is invoked, follow these steps:

### 1. Pre-Update Assessment

First, check the current versions:

```bash
# Check current talosctl version
talosctl version

# Check cluster health before upgrade
talosctl health

# Check available releases
curl -s https://api.github.com/repos/siderolabs/talos/releases | grep -o '"tag_name": "[^"]*"' | head -5
```

Inform the user of:
- Current talosctl client version
- Current Talos OS version on cluster
- Latest available stable versions (as of last knowledge: v1.10.8, v1.11.5)
- Whether cluster is healthy

### 2. Update talosctl CLI

Provide instructions based on the user's OS:

**For macOS (using Homebrew):**
```bash
brew upgrade siderolabs/tap/talosctl
```

**For Linux/macOS (manual install):**
```bash
# Download latest version (replace VERSION with target version)
curl -LO https://github.com/siderolabs/talos/releases/download/VERSION/talosctl-darwin-amd64

# For Linux:
# curl -LO https://github.com/siderolabs/talos/releases/download/VERSION/talosctl-linux-amd64

# Make executable and move to PATH
chmod +x talosctl-*
sudo mv talosctl-* /usr/local/bin/talosctl
```

Verify the update:
```bash
talosctl version
```

### 3. Upgrade Talos OS on Cluster

**Important considerations to communicate:**
- Talos upgrades should follow minor version paths (e.g., 1.10.x → 1.11.x)
- Always use the latest patch version in each minor release
- The upgrade is rolling and non-disruptive
- Kubernetes version upgrades are separate

**Pre-upgrade check:**
```bash
# Dry-run to see what will happen
talosctl upgrade --nodes <NODE_IP> \
  --image ghcr.io/siderolabs/installer:v<VERSION> \
  --preserve
```

**Execute upgrade:**
```bash
# Upgrade the node
talosctl upgrade --nodes 10.0.0.67 \
  --image ghcr.io/siderolabs/installer:v<VERSION> \
  --preserve

# For debugging, add: --wait --debug
```

**Monitor upgrade progress:**
```bash
# Watch kernel logs during upgrade
talosctl dmesg --follow --nodes 10.0.0.67
```

**Post-upgrade verification:**
```bash
# Verify new version
talosctl version --nodes 10.0.0.67

# Check cluster health
talosctl health --nodes 10.0.0.67

# Verify Kubernetes nodes
kubectl get nodes -o wide
```

### 4. Upgrade Kubernetes (Optional)

If the user also wants to upgrade Kubernetes, explain that this is separate:

**Check current Kubernetes version:**
```bash
kubectl version --short
```

**Dry-run Kubernetes upgrade:**
```bash
talosctl --nodes 10.0.0.67 upgrade-k8s --to <K8S_VERSION> --dry-run
```

**Execute Kubernetes upgrade:**
```bash
talosctl --nodes 10.0.0.67 upgrade-k8s --to <K8S_VERSION>
```

This process:
1. Pre-pulls images
2. Updates control plane components
3. Updates kube-proxy
4. Updates kubelet on all nodes
5. Re-applies bootstrap manifests

**Verify Kubernetes upgrade:**
```bash
kubectl version
kubectl get nodes
```

### 5. Update Documentation

After successful upgrades, update the following files:
- `CLAUDE.md` - Update Talos and Kubernetes versions
- `notebook/Talos Management Guide.md` - Update version numbers

### Important Notes to Share

- **Automatic rollback**: If an upgrade fails to boot, Talos automatically rolls back
- **Single-node cluster**: The upgrade will cause brief API unavailability
- **Control plane**: Talos protects etcd quorum during upgrades
- **Preserve flag**: `--preserve` keeps ephemeral data during upgrade
- **Stage flag**: Use `--stage` if having issues with file locks

### Troubleshooting

If upgrade fails:
```bash
# Check upgrade status
talosctl get upgradestatus --nodes 10.0.0.67

# View service logs
talosctl logs machined --nodes 10.0.0.67

# Force rollback if needed (automatic usually)
talosctl rollback --nodes 10.0.0.67
```

## Workflow

1. Ask the user what they want to upgrade:
   - Just the talosctl CLI?
   - Talos OS on the cluster?
   - Both Talos OS and Kubernetes?

2. Show current versions and available versions

3. Confirm target versions with user

4. Execute upgrades step by step, showing output

5. Verify each upgrade before proceeding to next

6. Update documentation files

7. Provide summary of what was upgraded
