# Update Kubernetes

Guide the user through updating Kubernetes on the Talos homelab cluster. This command handles upgrading the Kubernetes version while preserving cluster state and workloads.

## Instructions for Claude

When this command is invoked, follow these steps:

### 1. Pre-Upgrade Assessment

First, check the current state:

```bash
# Check current Kubernetes version
kubectl version

# Check current Talos version (must be compatible)
talosctl version --nodes 10.0.0.67

# Check cluster health before upgrade
talosctl health --nodes 10.0.0.67

# List all running workloads (to verify after upgrade)
kubectl get pods -A
```

Inform the user of:
- Current Kubernetes version (client and server)
- Current Talos version (ensure compatibility)
- Cluster health status
- Number of running workloads

### 2. Check Available Kubernetes Versions

```bash
# Check available Kubernetes versions for your Talos version
# Talos v1.11.x supports Kubernetes 1.32.x - 1.34.x
curl -s https://api.github.com/repos/kubernetes/kubernetes/releases | grep -o '"tag_name": "v[^"]*"' | head -10
```

**Kubernetes Version Compatibility (as of knowledge cutoff):**
- Talos v1.11.x → Kubernetes v1.32.x, v1.33.x, v1.34.x
- Talos v1.10.x → Kubernetes v1.31.x, v1.32.x, v1.33.x

**Important notes:**
- Only upgrade one minor version at a time (e.g., 1.32.x → 1.33.x)
- Always use the latest patch version (e.g., 1.33.5 instead of 1.33.0)
- Ensure Talos version supports the target Kubernetes version
- The current cluster is on v1.33.0, can upgrade to latest v1.33.x or v1.34.x

### 3. Dry-Run Kubernetes Upgrade

Always perform a dry-run first to check for issues:

```bash
# Dry-run upgrade to see what will happen
talosctl --nodes 10.0.0.67 upgrade-k8s --to <K8S_VERSION> --dry-run
```

The dry-run will show:
- Which images will be pulled
- Control plane components to be updated
- Any potential compatibility warnings

### 4. Execute Kubernetes Upgrade

If dry-run looks good, proceed with the upgrade:

```bash
# Execute the upgrade
talosctl --nodes 10.0.0.67 upgrade-k8s --to <K8S_VERSION>
```

**What happens during upgrade:**
1. Pre-pulls new Kubernetes images
2. Updates control plane components (kube-apiserver, kube-controller-manager, kube-scheduler)
3. Updates kube-proxy on all nodes
4. Updates kubelet on all nodes
5. Re-applies bootstrap manifests
6. Waits for components to become healthy

**Monitoring the upgrade:**
```bash
# Watch control plane pods restart
kubectl get pods -n kube-system --watch

# Check control plane component status
kubectl get componentstatuses

# Monitor API server health
kubectl get --raw /healthz

# View kubelet logs if needed
talosctl logs kubelet --follow --nodes 10.0.0.67
```

### 5. Post-Upgrade Verification

After the upgrade completes, verify everything:

```bash
# Verify new Kubernetes version
kubectl version

# Check all nodes are ready and on new version
kubectl get nodes -o wide

# Check all system pods are running
kubectl get pods -n kube-system

# Verify workloads are healthy
kubectl get pods -A

# Check for any pod issues
kubectl get pods -A | grep -v Running | grep -v Completed

# Verify services are accessible
kubectl get svc -A | grep LoadBalancer

# Test DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup kubernetes.default
```

### 6. Verify Critical Services

Check that critical homelab services are still functioning:

```bash
# Check Tailscale operator
kubectl get pods -n tailscale
kubectl logs -n tailscale deployment/operator --tail=20

# Check Flux GitOps
kubectl get pods -n flux-system
flux check

# Check monitoring stack
kubectl get pods -n monitoring

# Check application services
kubectl get pods -n uptime-kuma
kubectl get pods -n ntfy
kubectl get pods -n minecraft

# Verify persistent volumes are still mounted
kubectl get pv
kubectl get pvc -A
```

### 7. Update Documentation

After successful upgrade, update the following files:
- `CLAUDE.md` - Update Kubernetes version in Architecture Overview (line 9)
- `notebook/Talos Management Guide.md` - Update Kubernetes version (line 10)

### Important Notes to Share

**Before upgrading:**
- Talos manages Kubernetes as a system service
- Upgrades are rolling and minimize disruption
- Control plane updates happen first, then worker components
- Single-node cluster will experience brief API server restarts

**During upgrade:**
- API server may be briefly unavailable during control plane update
- Existing pods continue running during the upgrade
- Network connectivity should remain stable
- StatefulSets and persistent volumes are preserved

**After upgrade:**
- Some pods may restart to pick up new kubelet features
- CoreDNS and kube-proxy will be on new versions
- Check for deprecated API versions in your manifests
- Review Kubernetes release notes for breaking changes

### Troubleshooting

**If upgrade fails or hangs:**

```bash
# Check upgrade progress
talosctl get kubernetescontrolplaneresources --nodes 10.0.0.67

# View kubelet status
talosctl service kubelet status --nodes 10.0.0.67

# Check for image pull issues
talosctl logs kubelet --nodes 10.0.0.67 | grep -i "error\|failed"

# View control plane pod logs
kubectl logs -n kube-system kube-apiserver-talos-84a-hmg
kubectl logs -n kube-system kube-controller-manager-talos-84a-hmg
kubectl logs -n kube-system kube-scheduler-talos-84a-hmg

# Check etcd health
talosctl etcd member list --nodes 10.0.0.67
kubectl get pods -n kube-system | grep etcd
```

**Common issues:**

1. **Image pull failures**: Check internet connectivity and container registry access
2. **API server not ready**: Check etcd health and control plane logs
3. **Nodes not updating**: Verify kubelet service is running
4. **Pods stuck in pending**: Check node resources and taints

**Rollback (if necessary):**

Kubernetes upgrades via Talos don't have automatic rollback. If you need to rollback:

```bash
# Downgrade to previous version (only if upgrade failed)
talosctl --nodes 10.0.0.67 upgrade-k8s --to <PREVIOUS_VERSION>
```

### Version History Reference

Keep track of upgrades in documentation:
- Current: v1.33.0
- Previous: (track when you upgrade)

## Workflow

1. **Assess current state**: Show versions and cluster health

2. **Check compatibility**: Verify Talos version supports target Kubernetes version

3. **Choose target version**:
   - Recommend latest patch of current minor version (e.g., 1.33.x → 1.33.5)
   - Or next minor version if user wants feature upgrade (e.g., 1.33.x → 1.34.0)

4. **Confirm with user**: Get explicit approval for target version

5. **Dry-run**: Execute dry-run and show results

6. **Execute upgrade**: If dry-run passes, proceed with real upgrade

7. **Monitor progress**: Watch component updates in real-time

8. **Verify cluster**: Run comprehensive post-upgrade checks

9. **Verify services**: Ensure all workloads are healthy

10. **Update docs**: Update version numbers in documentation

11. **Provide summary**: List what was upgraded and any issues encountered

## Safety Checklist

Before upgrading, ensure:
- [ ] Talos version is compatible with target Kubernetes version
- [ ] Cluster health check passes
- [ ] You understand the release notes for the target version
- [ ] Critical workloads have been noted for post-upgrade verification
- [ ] You have recent etcd backups (optional but recommended)

After upgrading, verify:
- [ ] All nodes show new Kubernetes version
- [ ] All system pods are running
- [ ] All application pods are running
- [ ] Services are accessible
- [ ] Persistent volumes are mounted
- [ ] Tailscale connectivity works
- [ ] Monitoring stack is operational
