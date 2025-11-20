# Talos Management Guide

## Cluster Overview

### Current Configuration
- **Hardware**: Bee Machine
- **Node**: talos-84a-hmg at 10.0.0.67
- **Role**: Control Plane (single-node cluster)
- **Talos Version**: v1.11.5
- **Kubernetes Version**: v1.33.0
- **Age**: 180 days
- **Network**: Flannel CNI

### Running Services
Your cluster currently hosts:
- **Monitoring Stack**: Prometheus, Grafana, Loki, Promtail
- **Web Services**: Field Theories blog, Umami analytics, Uptime Kuma
- **Infrastructure**: Postgres, Minecraft server, Ntfy notifications
- **Storage**: 13 persistent volumes (local-path-provisioner)
- **Networking**: Tailscale operator with 14 proxy pods
- **GitOps**: Flux controllers for automated deployments

## Certificate Issue and Resolution

### Problem
Your talosctl client is experiencing TLS certificate verification failures:
```
x509: certificate signed by unknown authority (possibly because of "x509: Ed25519 verification failure")
```

This prevents direct Talos API access for node management, service inspection, and system operations.

### Quick Fix: Regenerate Talos Config

You have three contexts in your talosconfig:
- `talos-default` → 10.0.0.67 (current)
- `talos-default homecluster` → 10.0.0.70 (old?)
- `talos-default homecluster-1` → 10.0.0.70 (old?)

To fix the certificate issue:

```bash
# Option 1: If you have the original talosconfig from cluster creation
# Copy it from your backup/secure location to ~/.talos/config

# Option 2: Generate a new admin kubeconfig and talosconfig
# You'll need physical/console access to the node to reset the PKI
talosctl --nodes 10.0.0.67 kubeconfig --force

# Option 3: Work around it for now using kubectl (see below)
```

### Working Around the Issue

Until certificates are fixed, use kubectl for cluster management:
```bash
# All cluster info is accessible via kubectl
kubectl get nodes -o wide
kubectl get pods -A
kubectl get svc -A
kubectl describe node talos-84a-hmg
kubectl logs -n kube-system <pod-name>
```

## Essential Talos Commands

Once certificates are working, these are your key Talos management commands:

### Cluster Health and Info
```bash
# Check overall cluster health
talosctl health --nodes 10.0.0.67

# Get Talos/Kubernetes versions
talosctl version --nodes 10.0.0.67

# View cluster membership
talosctl get members --nodes 10.0.0.67

# Interactive dashboard with metrics
talosctl dashboard --nodes 10.0.0.67
```

### Node Management
```bash
# View all running services
talosctl service --nodes 10.0.0.67

# Check specific service status
talosctl service kubelet status --nodes 10.0.0.67

# View service logs
talosctl logs kubelet --nodes 10.0.0.67

# View kernel logs
talosctl dmesg --nodes 10.0.0.67

# Check system resource usage
talosctl memory --nodes 10.0.0.67
talosctl processes --nodes 10.0.0.67
```

### Container Management
```bash
# List running containers
talosctl containers --nodes 10.0.0.67

# Container resource stats
talosctl stats --nodes 10.0.0.67

# View container images
talosctl image list --nodes 10.0.0.67
```

### System Operations
```bash
# Reboot node
talosctl reboot --nodes 10.0.0.67

# Shutdown node
talosctl shutdown --nodes 10.0.0.67

# Restart a service
talosctl restart kubelet --nodes 10.0.0.67
```

### Configuration
```bash
# View current machine config
talosctl get machineconfig --nodes 10.0.0.67 -o yaml

# Apply updated configuration
talosctl apply-config --nodes 10.0.0.67 --file updated-config.yaml

# Edit config interactively
talosctl edit machineconfig --nodes 10.0.0.67
```

### Upgrades
```bash
# Upgrade Talos OS
talosctl upgrade --nodes 10.0.0.67 --image ghcr.io/siderolabs/installer:v1.11.5 --preserve

# Upgrade Kubernetes
talosctl upgrade-k8s --nodes 10.0.0.67 --to 1.34.0
```

### Etcd Management
```bash
# Check etcd member status
talosctl etcd member list --nodes 10.0.0.67

# View etcd alarms (if any)
talosctl etcd alarm list --nodes 10.0.0.67

# Take etcd snapshot backup
talosctl etcd snapshot --nodes 10.0.0.67
```

## Resource Discovery

Talos has a resource-oriented API similar to Kubernetes. List available resource types:

```bash
# Get all resource definitions
talosctl get rd --nodes 10.0.0.67

# Example: Get network configuration
talosctl get nodeaddresses --nodes 10.0.0.67
talosctl get nodeipconfig --nodes 10.0.0.67

# Get hardware info
talosctl get systeminfo --nodes 10.0.0.67
```

## Cluster Management via kubectl

Your cluster is working great via kubectl. Here are useful commands:

### Cluster Status
```bash
# Node information
kubectl get nodes -o wide
kubectl describe node talos-84a-hmg

# Cluster component health
kubectl get componentstatuses
kubectl get --raw /healthz
```

### Application Management
```bash
# View all workloads
kubectl get pods,deployments,statefulsets -A

# Check Tailscale proxies
kubectl get pods -n tailscale

# View service endpoints
kubectl get endpoints -A
```

### Storage
```bash
# Check persistent volumes
kubectl get pv
kubectl get pvc -A

# View storage class
kubectl get storageclass
kubectl describe storageclass local-path
```

### Monitoring and Logs
```bash
# Pod logs
kubectl logs -n monitoring prometheus-865dfc6f78-dcc29

# Follow logs
kubectl logs -f -n flux-system source-controller-54bc45dc6-mt7nh

# Previous container logs (after restart)
kubectl logs --previous -n <namespace> <pod>
```

## Troubleshooting

### Common Issues

#### Tailscale Proxy Pods Not Starting
```bash
# Check operator logs
kubectl logs -n tailscale deployment/operator

# Verify generic device plugin (needed for /dev/net/tun)
kubectl get pods -n kube-system | grep generic-device-plugin

# Check pod events
kubectl describe pod -n tailscale <proxy-pod-name>
```

#### Storage Issues
```bash
# Check provisioner logs
kubectl logs -n local-path-storage deployment/local-path-provisioner

# Verify pending PVCs
kubectl get pvc -A | grep Pending

# Check node disk space via kubectl
kubectl debug node/talos-84a-hmg -it --image=busybox -- df -h
```

#### Network Problems
```bash
# Check CNI pods
kubectl get pods -n kube-system | grep flannel

# View CoreDNS status
kubectl get pods -n kube-system | grep coredns
kubectl logs -n kube-system <coredns-pod>
```

### Talos-Specific Debugging

Once certificates work:

```bash
# Capture network traffic
talosctl pcap --nodes 10.0.0.67 --interface eth0

# View mounts
talosctl mounts --nodes 10.0.0.67

# Check network connections
talosctl netstat --nodes 10.0.0.67

# Disk usage
talosctl usage --nodes 10.0.0.67

# Read arbitrary files
talosctl read /proc/meminfo --nodes 10.0.0.67
```

## Backup and Recovery

### Critical Backups

1. **Talos Configuration**
   ```bash
   # Backup current config
   talosctl get machineconfig -o yaml > talos-config-backup.yaml
   ```

2. **Etcd Snapshots**
   ```bash
   # Take snapshot
   talosctl etcd snapshot --nodes 10.0.0.67
   ```

3. **Kubernetes Resources**
   ```bash
   # Backup all resources (GitOps helps here!)
   kubectl get all -A -o yaml > k8s-backup.yaml

   # Backup specific namespaces
   kubectl get all,pvc,configmap,secret -n uptime-kuma -o yaml > uptime-kuma-backup.yaml
   ```

### Recovery Scenarios

#### Node Replacement
If you need to rebuild the node:
1. Boot Bee Machine with Talos installer
2. Apply your backed-up machine config
3. Point talosctl to new IP
4. Bootstrap or join cluster

#### Data Recovery
Your PVCs use local-path storage at `/opt/local-path-provisioner/` on the node. This data persists across pod restarts but not node rebuilds.

## Next Steps

1. **Fix Certificate Issue**: Priority 1 to regain full Talos API access
2. **Update Documentation**: Your CLAUDE.md mentions 10.0.0.70, but actual IP is 10.0.0.67
3. **System Maintenance**:
   - Talos OS is up to date at v1.11.5 ✅
   - Kubernetes v1.33.0 is current
4. **Automate Backups**: Set up regular etcd snapshots and config backups
5. **Add Monitoring**: Connect Talos metrics to your Prometheus instance

## References

- [Talos Documentation](https://www.talos.dev/latest/)
- [Talos CLI Reference](https://www.talos.dev/latest/reference/cli/)
- [Kubernetes on Talos](https://www.talos.dev/latest/kubernetes-guides/configuration/)
