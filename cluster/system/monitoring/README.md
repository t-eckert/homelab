# Monitoring Stack

Comprehensive observability stack for the Talos Kubernetes homelab featuring metrics, logs, and visualization.

## Overview

Full monitoring solution consisting of:
- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Loki**: Log aggregation
- **Promtail**: Log shipping
- **Kube State Metrics**: Kubernetes object metrics
- **Node Exporter**: Node-level system metrics

## Architecture

```
┌──────────────┐
│   Grafana    │  Dashboards & Visualization (port 3000)
│  (Internal)  │  ├─ Prometheus datasource
└──────┬───────┘  └─ Loki datasource + PostgreSQL
       │
┌──────▼───────┐  ┌────────────┐  ┌──────────────┐
│  Prometheus  │  │    Loki    │  │  PostgreSQL  │
│  (Port 9090) │  │ (Port 3100)│  │  (External)  │
└──────┬───────┘  └─────┬──────┘  └──────────────┘
       │                │
┌──────▼────────────────▼──────┐
│  Metrics & Log Collectors    │
│  ├─ Kube State Metrics       │
│  ├─ Node Exporter (DaemonSet)│
│  └─ Promtail (DaemonSet)     │
└──────────────────────────────┘
```

## Access

- **Grafana**: Internal only - `http://grafana.monitoring:3000`
- **Prometheus**: `http://prometheus-homelab.feist-gondola.ts.net:9090` (Tailscale)
- **Loki**: Internal only - `http://loki.monitoring:3100`

## Components

### Grafana

**Purpose**: Unified dashboards for metrics and logs
**Port**: 3000
**Storage**: 2Gi PVC for dashboard storage

**Features**:
- Pre-configured Prometheus datasource
- Pre-configured Loki datasource
- Pre-configured PostgreSQL datasource
- Persistent dashboard storage

**Resources**:
- Memory: 256Mi request, 512Mi limit
- CPU: 100m request, 200m limit

### Prometheus

**Purpose**: Time-series metrics collection
**Port**: 9090
**Storage**: 10Gi PVC for metrics

**Features**:
- External access via Tailscale LoadBalancer
- Scrapes all Kubernetes services
- Stores metrics for 15 days (configurable)
- Service discovery for dynamic targets

**Resources**:
- Memory: 512Mi request, 1Gi limit
- CPU: 200m request, 500m limit

**Scrape Targets**:
- Kubernetes API server
- Kubelet metrics
- cAdvisor (container metrics)
- Kube State Metrics
- Node Exporter
- Application metrics endpoints

### Loki

**Purpose**: Log aggregation and storage
**Port**: 3100
**Storage**: 10Gi PVC for logs

**Features**:
- Label-based log indexing
- Efficient log compression
- Integration with Grafana
- Retention policy configured

**Resources**:
- Memory: 256Mi request, 512Mi limit
- CPU: 100m request, 200m limit

### Promtail

**Purpose**: Log collection from nodes
**Deployment**: DaemonSet (runs on all nodes)

**Features**:
- Collects logs from /var/log
- Labels logs with node and pod metadata
- Ships to Loki
- Uses Kubernetes service discovery

**Resources** (per node):
- Memory: 128Mi request, 256Mi limit
- CPU: 50m request, 100m limit

### Kube State Metrics

**Purpose**: Kubernetes object state metrics
**Port**: 8080

**Metrics**:
- Deployment status
- Pod states
- Node conditions
- Resource quotas
- PersistentVolume status

**Resources**:
- Memory: 128Mi request, 256Mi limit
- CPU: 50m request, 100m limit

### Node Exporter

**Purpose**: Node-level system metrics
**Deployment**: DaemonSet (runs on all nodes)
**Port**: 9100

**Metrics**:
- CPU usage
- Memory usage
- Disk I/O
- Network statistics
- Filesystem usage

**Resources** (per node):
- Memory: 64Mi request, 128Mi limit
- CPU: 50m request, 100m limit

## Setup

### Prerequisites

- Kubernetes cluster with PersistentVolume support
- PostgreSQL database (for Grafana PostgreSQL datasource)

### Initial Deployment

```bash
# Apply all monitoring components
kubectl apply -f cluster/system/monitoring/

# Verify deployment
kubectl get pods -n monitoring
kubectl get pvc -n monitoring
```

### Configuration

**Prometheus Configuration**: Defined in `prometheus-configmap.yaml`
- Scrape interval: 30s
- Evaluation interval: 30s
- Retention: 15 days

**Loki Configuration**: Defined in `loki-configmap.yaml`
- Retention: 168h (7 days)
- Chunk encoding: gzip

**Promtail Configuration**: Defined in `promtail-configmap.yaml`
- Scrapes /var/log/pods
- Labels: namespace, pod, container

## Accessing Dashboards

### Port Forward to Grafana

```bash
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

Then open `http://localhost:3000` in your browser.

### Access Prometheus UI

Via Tailscale: `http://prometheus-homelab.feist-gondola.ts.net:9090`

Or port forward:
```bash
kubectl port-forward -n monitoring svc/prometheus 9090:9090
```

## Common Queries

### Prometheus Queries

**CPU Usage by Pod**:
```promql
sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)
```

**Memory Usage by Namespace**:
```promql
sum(container_memory_usage_bytes) by (namespace)
```

**Pod Restart Count**:
```promql
kube_pod_container_status_restarts_total
```

### Loki Queries

**Logs from specific namespace**:
```logql
{namespace="field-theories"}
```

**Error logs across cluster**:
```logql
{} |= "error" | json
```

## Maintenance

### Check Status

```bash
# All monitoring pods
kubectl get pods -n monitoring

# PVC usage
kubectl get pvc -n monitoring

# Service endpoints
kubectl get endpoints -n monitoring
```

### View Logs

```bash
# Prometheus
kubectl logs -n monitoring -l app=prometheus

# Grafana
kubectl logs -n monitoring -l app=grafana

# Loki
kubectl logs -n monitoring -l app=loki

# Promtail (specific node)
kubectl logs -n monitoring -l app=promtail --tail=50
```

### Restart Components

```bash
# Restart Prometheus
kubectl rollout restart deployment/prometheus -n monitoring

# Restart Grafana
kubectl rollout restart deployment/grafana -n monitoring

# Restart Loki
kubectl rollout restart deployment/loki -n monitoring

# Restart Promtail (DaemonSet)
kubectl rollout restart daemonset/promtail -n monitoring
```

### Storage Management

```bash
# Check PVC usage
kubectl exec -n monitoring deployment/prometheus -- df -h /prometheus
kubectl exec -n monitoring deployment/loki -- df -h /loki

# If storage is full, adjust retention or expand PVC
kubectl edit pvc prometheus-pvc -n monitoring
```

## Troubleshooting

### Prometheus Not Scraping Targets

Check targets in Prometheus UI:
```
http://prometheus-homelab.feist-gondola.ts.net:9090/targets
```

Common issues:
- ServiceMonitor not created: Check `prometheus-configmap.yaml` for target configs
- Network policy blocking: Verify namespace network policies
- Pod not exposing metrics: Check pod has metrics endpoint and port labeled

### Grafana Can't Connect to Datasources

```bash
# Check datasource configuration
kubectl get configmap -n monitoring grafana-datasources -o yaml

# Test connectivity from Grafana pod
kubectl exec -it -n monitoring deployment/grafana -- \
  wget -O- http://prometheus.monitoring:9090/-/healthy
```

### Loki Not Receiving Logs

Check Promtail:
```bash
# Promtail logs
kubectl logs -n monitoring -l app=promtail --tail=100

# Verify Promtail is running on all nodes
kubectl get pods -n monitoring -l app=promtail -o wide
```

Test Loki connectivity:
```bash
kubectl exec -it -n monitoring daemonset/promtail -- \
  wget -O- http://loki.monitoring:3100/ready
```

### High Memory Usage

Monitor resource usage:
```bash
kubectl top pods -n monitoring
```

Adjust resource limits if needed:
```bash
kubectl edit deployment/prometheus -n monitoring
kubectl edit deployment/grafana -n monitoring
```

### Persistent Volume Issues

```bash
# Check PV status
kubectl get pv | grep monitoring

# Check PVC binding
kubectl get pvc -n monitoring

# Describe for events
kubectl describe pvc -n monitoring
```

## Security Notes

- All services run as non-root users
- RBAC configured for Kube State Metrics and Promtail
- Read-only root filesystems where possible
- Grafana accessible only internally (no LoadBalancer)
- Prometheus exposed via Tailscale for secure remote access
- No authentication configured on Prometheus (rely on network security)

## Performance Tuning

### Reduce Prometheus Retention

Edit `prometheus-configmap.yaml`:
```yaml
global:
  retention.time: 7d  # Reduce from 15d
```

### Adjust Scrape Intervals

```yaml
scrape_configs:
  - job_name: 'kubernetes-pods'
    scrape_interval: 60s  # Increase from 30s
```

### Reduce Log Retention in Loki

Edit `loki-configmap.yaml`:
```yaml
limits_config:
  retention_period: 72h  # Reduce from 168h
```

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Loki Documentation](https://grafana.com/docs/loki/latest/)
- [Kube State Metrics](https://github.com/kubernetes/kube-state-metrics)
- [Node Exporter](https://github.com/prometheus/node_exporter)
