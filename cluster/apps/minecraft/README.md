# Minecraft Server

Scalable Minecraft server deployment with on-demand resource management.

## Overview

Minecraft Java Edition server that can be scaled up/down to manage resource usage on the homelab.

## Access

- **Server Address**: `minecraft-homelab.feist-gondola.ts.net:25565` (Tailscale)
- **Game Version**: Latest (auto-updating)
- **Server Type**: Vanilla

## Configuration

**Image**: `itzg/minecraft-server:latest`
**Port**: 25565 (TCP)
**Storage**: 5Gi PVC for world data

**Resources**:
- Memory: 2Gi request, 3Gi limit
- CPU: 1000m request, 2000m limit

**Server Settings**:
- EULA: Accepted
- Max Players: 20
- Difficulty: Normal
- Game Mode: Survival

## Scaling

The Minecraft server can be scaled down when not in use to save ~1Gi+ of memory:

### Scale Down (Stop Server)

```bash
task minecraft:scale-down
```

This sets replicas to 0, stopping the server but preserving world data.

### Scale Up (Start Server)

```bash
task minecraft:scale-up
```

This sets replicas to 1, starting the server back up.

### Check Status

```bash
task minecraft:status
```

Shows deployment status, pod status, and resource usage.

## World Management

### Backup World Data

```bash
# Backup world to local machine
kubectl exec -n minecraft deployment/minecraft -- tar czf - /data > minecraft-world-backup-$(date +%Y%m%d).tar.gz
```

### Restore World Data

```bash
# Restore from backup
cat minecraft-world-backup.tar.gz | kubectl exec -i -n minecraft deployment/minecraft -- tar xzf - -C /
kubectl rollout restart deployment/minecraft -n minecraft
```

### Access Server Files

```bash
# List files
kubectl exec -n minecraft deployment/minecraft -- ls -la /data

# View server properties
kubectl exec -n minecraft deployment/minecraft -- cat /data/server.properties

# View server logs
kubectl logs -n minecraft deployment/minecraft -f
```

## Configuration Updates

Edit server properties:

```bash
kubectl exec -it -n minecraft deployment/minecraft -- vi /data/server.properties
kubectl rollout restart deployment/minecraft -n minecraft
```

Or update via deployment environment variables in `deployment.yaml`.

## Maintenance

### Update Server

Server auto-updates to latest version on pod restart:

```bash
kubectl rollout restart deployment/minecraft -n minecraft
```

### View Server Logs

```bash
kubectl logs -n minecraft deployment/minecraft -f
```

### Execute Server Commands

```bash
kubectl exec -n minecraft deployment/minecraft -- rcon-cli <command>
# Examples:
# kubectl exec -n minecraft deployment/minecraft -- rcon-cli list
# kubectl exec -n minecraft deployment/minecraft -- rcon-cli say Hello players!
```

## Troubleshooting

### Server Won't Start

Check logs for errors:
```bash
kubectl logs -n minecraft deployment/minecraft --tail=100
```

Common issues:
- Insufficient memory: Check resource limits
- Corrupted world data: Restore from backup
- Version mismatch: Clear PVC and restart

### Can't Connect

```bash
# Check service
kubectl get svc -n minecraft

# Check Tailscale proxy
kubectl get pods -n tailscale | grep minecraft

# Test connectivity
kubectl exec -n minecraft deployment/minecraft -- nc -zv localhost 25565
```

### Performance Issues

```bash
# Check resource usage
kubectl top pod -n minecraft

# Consider increasing resources in deployment.yaml
```

## Resource Management

The Minecraft server is resource-intensive. Best practices:

1. **Scale down when not in use**: Use `task minecraft:scale-down`
2. **Monitor resource usage**: Use `task minecraft:status`
3. **Adjust player limits**: Edit `MAX_PLAYERS` in deployment
4. **Regular world cleanup**: Remove unused chunks

## Players

To add operators or whitelist players, exec into the pod:

```bash
kubectl exec -it -n minecraft deployment/minecraft -- rcon-cli op <playername>
kubectl exec -it -n minecraft deployment/minecraft -- rcon-cli whitelist add <playername>
```

## References

- [itzg/minecraft-server Docker Image](https://github.com/itzg/docker-minecraft-server)
- [Minecraft Wiki](https://minecraft.wiki/)
- [Server Properties Reference](https://minecraft.wiki/w/Server.properties)
