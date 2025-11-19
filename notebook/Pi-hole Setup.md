# Pi-hole Setup

Pi-hole is deployed in the Kubernetes cluster with Tailscale integration for remote DNS access.

## Architecture

- **Namespace**: `pihole`
- **Deployment Type**: StatefulSet with persistent storage
- **Image**: `pihole/pihole:2024.07.0`
- **Storage**: 2x 1Gi PVCs (pihole config and dnsmasq config)
- **Access**: Tailscale LoadBalancer services

## Services

### Web Interface
- **Service**: `pihole-web`
- **Tailscale Hostname**: `pihole`
- **Port**: 80 (HTTP)
- **Access**: Connect to Tailscale, then navigate to `http://pihole` in your browser

### DNS Service
- **Services**: `pihole-dns-tcp` and `pihole-dns-udp`
- **Tailscale Hostname**: `pihole-dns`
- **Port**: 53 (TCP/UDP)
- **Access**: Use Tailscale IP as DNS server when connected to Tailscale network

## Deployment

### Initial Setup

```bash
# Apply the namespace
kubectl apply -f cluster/pihole/namespace.yaml

# Apply the services
kubectl apply -f cluster/pihole/service.yaml

# Deploy Pi-hole
kubectl apply -f cluster/pihole/stateful-set.yaml
```

### Verify Deployment

```bash
# Check pod status
kubectl get pods -n pihole

# Check services
kubectl get svc -n pihole

# Check Tailscale endpoints
kubectl get svc -A | grep pihole

# View logs
kubectl logs -n pihole -f statefulset/pihole
```

## Configuration

### Environment Variables

The following environment variables are configured in `stateful-set.yaml`:

- **TZ**: `America/New_York` - Set to your timezone
- **FTLCONF_webserver_api_password**: `changeme` - **Change this before deployment!**
- **FTLCONF_dns_upstreams**: `1.1.1.1;1.0.0.1` - Cloudflare DNS (customize as needed)

### Update Admin Password

**IMPORTANT**: Before deploying, edit `cluster/pihole/stateful-set.yaml` and change the `FTLCONF_webserver_api_password` value from `changeme` to a secure password.

## Using Pi-hole via Tailscale

### Get Tailscale DNS IP

After deployment, get the Tailscale IP address for the DNS service:

```bash
# Get the LoadBalancer ingress IP
kubectl get svc -n pihole pihole-dns-tcp -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
```

Or check in the Tailscale admin console for the device named `pihole-dns`.

### Configure Devices

#### macOS/Linux
1. Connect to Tailscale
2. Open network settings
3. Set DNS server to the Pi-hole Tailscale IP address

#### iOS/Android
1. Connect to Tailscale
2. Go to Tailscale app settings
3. Override local DNS with the Pi-hole Tailscale IP

#### Tailscale Global DNS (Recommended)
1. Go to Tailscale admin console: https://login.tailscale.com/admin/dns
2. Add nameserver with the Pi-hole Tailscale IP
3. All devices on your Tailnet will automatically use Pi-hole when connected

## Storage

Pi-hole uses two persistent volumes:
- `/etc/pihole` - Main configuration, gravity database, custom lists
- `/etc/dnsmasq.d` - DNS configuration files

Both volumes use the `local-path` storage class and are backed up to the node's local storage.

## Maintenance

### Update Pi-hole Image

```bash
# Edit the stateful set to update the image version
kubectl edit statefulset pihole -n pihole

# Or update the YAML and reapply
kubectl apply -f cluster/pihole/stateful-set.yaml
```

### Restart Pi-hole

```bash
kubectl rollout restart statefulset pihole -n pihole
```

### Access Logs

```bash
kubectl logs -n pihole -f statefulset/pihole
```

### Backup Configuration

Pi-hole configuration is stored in PVCs. To backup:

```bash
# Get the PVC mount paths on the node
kubectl get pv

# Access the node and backup the local-path storage directories
```

## Troubleshooting

### DNS Not Working

1. Check pod is running:
   ```bash
   kubectl get pods -n pihole
   ```

2. Check Tailscale services are assigned IPs:
   ```bash
   kubectl get svc -n pihole
   ```

3. Verify Tailscale operator logs:
   ```bash
   kubectl logs -n tailscale deployment/operator
   ```

4. Test DNS resolution:
   ```bash
   dig @<pihole-tailscale-ip> google.com
   ```

### Web Interface Not Accessible

1. Verify the web service has a Tailscale hostname:
   ```bash
   kubectl get svc -n pihole pihole-web
   ```

2. Check Tailscale devices in admin console
3. Try accessing via IP instead of hostname

### Permission Issues

The namespace uses `privileged` pod security to allow Tailscale proxy pods to function correctly.

## Security Notes

- Change the default admin password before deployment
- Pi-hole is only accessible via Tailscale (not exposed locally)
- Consider using Tailscale ACLs to restrict which devices can use Pi-hole
- Upstream DNS servers are set to Cloudflare (1.1.1.1) - adjust as needed for privacy preferences

## Resources

- Official Pi-hole Docker documentation: https://github.com/pi-hole/docker-pi-hole
- Tailscale Kubernetes operator: https://tailscale.com/kb/1236/kubernetes-operator
