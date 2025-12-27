# CPU Resource Optimization

## Current Situation

**CPU Allocation**: 3907m / 4000m (98% allocated)
**Actual CPU Usage**: 669m (16% of node capacity)
**Problem**: CPU requests are blocking new deployments despite low actual usage

## Top CPU Wasters (Request vs Actual Usage)

### Critical Offenders (>90% waste)

| Namespace | Pod | Requested | Actual | Waste | % Wasted |
|-----------|-----|-----------|--------|-------|----------|
| **kubernetes-dashboard** | api | 100m | 1m | 99m | 99% |
| **kubernetes-dashboard** | auth | 100m | 1m | 99m | 99% |
| **kubernetes-dashboard** | metrics-scraper | 100m | 1m | 99m | 99% |
| **kubernetes-dashboard** | web | 100m | 1m | 99m | 99% |
| **workbench** | debian | 100m | 0m | 100m | 100% |
| **umami** | umami | 100m | 3m | 97m | 97% |
| **jellyfin** | jellyfin | 100m | 2m | 98m | 98% |
| **monitoring** | prometheus | 250m | 17m | 233m | 93% |
| **monitoring** | kube-state-metrics | 100m | 4m | 96m | 96% |
| **monitoring** | node-exporter | 102m | 6m | 96m | 94% |
| **flux-system** | image-reflector | 100m | 13m | 87m | 87% |
| **flux-system** | helm-controller | 100m | 2m | 98m | 98% |
| **flux-system** | kustomize-controller | 100m | 2m | 98m | 98% |
| **flux-system** | notification-controller | 100m | 3m | 97m | 97% |
| **flux-system** | image-automation | 100m | 2m | 98m | 98% |

### Subtotal by Category

- **Kubernetes Dashboard**: 400m requested, 4m used → **396m waste**
- **Monitoring**: 502m requested, 27m used → **475m waste**
- **Flux Controllers**: 450m requested, 22m used → **428m waste**
- **Workbench**: 100m requested, 0m used → **100m waste**
- **Application Pods**: 300m+ requested, <10m used → **~290m waste**

**Total Recoverable**: ~1,689m (enough to deploy 16+ more Field Theories instances)

## Recommended Actions

### Immediate Wins (Reclaim 1,100m+)

1. **Reduce Kubernetes Dashboard requests** (400m → 50m, save 350m)
   ```yaml
   # cluster/kubernetes-dashboard/deployment.yaml
   resources:
     requests:
       cpu: 10m  # was 100m for each pod
   ```

2. **Reduce Monitoring stack requests** (502m → 150m, save 352m)
   ```yaml
   # Prometheus: 250m → 50m (save 200m)
   # Kube-state-metrics: 100m → 20m (save 80m)
   # Node-exporter: 102m → 30m (save 72m)
   ```

3. **Reduce Flux controllers** (450m → 150m, save 300m)
   ```yaml
   # Each controller: 100m → 30m
   # Image-reflector can stay at 50m since it uses 13m
   ```

4. **Delete or pause Workbench Debian pod** (save 100m)
   - Currently using 0m CPU
   - If not needed, delete entirely

### Priority Order

1. **Kubernetes Dashboard** - Easiest win, 4 pods using 1m each but requesting 100m
2. **Prometheus** - Single pod, 233m waste
3. **Flux Controllers** - 5 pods with high waste ratios
4. **Application pods** (Umami, Jellyfin) - If not heavily used, reduce requests

### Example Resource Adjustments

```yaml
# Low-usage services (dashboard, most apps)
resources:
  requests:
    cpu: 10m
  limits:
    cpu: 500m  # Allow bursting when needed

# Medium-usage services (monitoring, flux)
resources:
  requests:
    cpu: 25-50m
  limits:
    cpu: 1000m

# High-usage services (kube-apiserver, promtail)
resources:
  requests:
    cpu: 100m
  limits:
    cpu: 2000m
```

## Notes

- Tailscale proxies have no CPU requests (good!)
- Core K8s components (kube-apiserver, kube-controller-manager) are appropriately sized
- Most actual usage is well below requests, indicating over-provisioning
- With these changes, you could reduce total requests from 3907m to ~2200m (45% reduction)
