# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a Kubernetes homelab running on Talos Linux with Raspberry Pi hardware:
- Single-node control plane (Pi 4B 1GB, IP: 10.0.0.70)
- Separate Home Assistant instance (Pi 4B 4GB, IP: 10.0.0.195)
- Agent node (Pi 3B v2, IP: 10.0.0.232)

The cluster uses Tailscale for external access and local-path-provisioner for storage.

## Repository Structure

- `cluster/` - Kubernetes manifests organized by service
  - `apps/` - Application namespace definitions
  - `system/` - Core cluster components (storage, networking, device plugins)
  - `[service]/` - Service-specific deployments (minecraft, ntfy, uptime-kuma)
- `notebook/` - Technical documentation and setup guides
  - Documentation files use Title Case naming (e.g., `Home Assistant.md`, `Transfer Domain to Cloudflare.md`)
  - `notebook/plans/` - Write detailed implementation plans and procedures here using Title Case naming
- Service directories - Additional configuration files

## Common Commands

### Cluster Management
```bash
# Apply all namespaces
task cluster:apply-namespaces

# Apply specific service
kubectl apply -f cluster/[service]/

# Check service status
kubectl get svc -A | grep LoadBalancer
kubectl get pods -A
```

### Tailscale Troubleshooting
```bash
# Check operator logs
kubectl logs -n tailscale deployment/operator

# Check proxy pods
kubectl get pods -A | grep ts-

# Check service events
kubectl get events -A --sort-by='.lastTimestamp'
```

## Key Configuration Patterns

### Service Organization
Each service follows this structure:
- `namespace.yaml` - Namespace with PodSecurity labels for Tailscale compatibility
- `service.yaml` - LoadBalancer with Tailscale annotations
- `deployment.yaml` or `stateful-set.yaml` - Application workload

### Tailscale Integration
Services requiring external access need:
- `loadBalancerClass: tailscale`
- `tailscale.com/proxy-class: "tailscale-proxy"`
- `tailscale.com/hostname: "[service-name]"`

### Security Requirements
- Namespaces running Tailscale proxies need `pod-security.kubernetes.io/enforce: privileged`
- Use non-root containers where possible
- Drop capabilities and set seccomp profiles for security

## Important Notes

- The `tailscale-proxy` ProxyClass requires the generic device plugin for TUN device access
- StatefulSets are used for services requiring persistent storage (ntfy, uptime-kuma)
- Documentation in `notebook/` contains hardware specs and detailed setup procedures
- Local path provisioner is the default storage class for PVCs