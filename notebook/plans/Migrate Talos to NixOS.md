# Migrate Talos Linux to NixOS

## Motivation

The Bee Link (Intel N150, 16 GB RAM, 1 TB NVMe) runs a single-node Kubernetes
cluster on Talos Linux. The Kubernetes control plane (etcd, apiserver, scheduler,
controller-manager, kubelet) plus ecosystem components (Flux, Tailscale operator,
kube-state-metrics, local-path-provisioner, generic-device-plugin) consume roughly
**1.1-1.4 CPU cores** and **2-2.6 Gi RAM** before any application workload runs.

Switching to NixOS eliminates this overhead. Services run as systemd units or
OCI containers managed by Podman, configured declaratively in a Nix flake.
Expected savings: ~1 core CPU, ~2 Gi RAM (~25-30% more headroom).

## Architecture: Before and After

### Before (Talos + Kubernetes)

```
Talos Linux (immutable, minimal)
├── etcd
├── kube-apiserver
├── kube-controller-manager
├── kube-scheduler
├── kubelet
│   ├── Flux controllers (6)
│   ├── Tailscale operator + 14 proxy pods
│   ├── Traefik
│   ├── local-path-provisioner
│   ├── generic-device-plugin
│   ├── kube-state-metrics
│   ├── Monitoring (prometheus, grafana, loki, promtail, node-exporter)
│   ├── Postgres
│   └── App pods (atuin, field-theories, jellyfin, n8n, ntfy, etc.)
```

### After (NixOS)

```
NixOS (declarative, flake-based)
├── tailscaled (single daemon, serve/funnel for routing)
├── postgresql (native NixOS service)
├── podman (rootless containers for apps)
│   ├── atuin
│   ├── field-theories + bluesync + cloudflared
│   ├── n8n
│   ├── ntfy
│   ├── copyparty
│   ├── fave
│   ├── the-weather
│   ├── umami
│   ├── uptime-kuma
│   ├── paperless stack (when enabled)
│   ├── minecraft (when enabled)
│   └── workbench (jupyter + debian/ssh)
├── prometheus + grafana + loki (native or containers)
└── sshd
```

## Inventory: Every Service and Its Migration Path

### Infrastructure

| Service | Current | NixOS Approach | Data Migration |
|---------|---------|----------------|----------------|
| PostgreSQL | `postgres:17-alpine` container, 30 Gi PVC | `services.postgresql` NixOS module | pg_dump / pg_restore |
| Tailscale | Operator + 14 proxy pods + ProxyClass | `services.tailscale` + `tailscale serve/funnel` | Re-auth node, reconfigure serve routes |
| Traefik | Helm chart, IngressRoutes | Eliminated — Tailscale serve replaces it | N/A |
| Flux (6 controllers) | GitOps for K8s manifests | Eliminated — NixOS flake + deploy-rs or nixos-rebuild | N/A |
| local-path-provisioner | Dynamic PVC provisioning | Eliminated — direct bind mounts to `/data/<service>/` | Copy PVC data from `/var/local-path-provisioner/` |
| generic-device-plugin | TUN device for Tailscale pods | Eliminated — Tailscale runs natively | N/A |

### Applications

| Service | Image | Postgres DB | Persistent Data | Tailscale Hostname |
|---------|-------|-------------|-----------------|-------------------|
| atuin | `ghcr.io/atuinsh/atuin:latest` | `atuin` | None (stateless) | `atuin-homelab` (port 8888) |
| copyparty | `copyparty/dj:latest` | None | 1 Gi data | `copyparty` (port 80) |
| fave | `ghcr.io/t-eckert/fave:latest` | None | 1 Gi data (bookmarks.json) | `fave` (HTTPS ingress) |
| field-theories | `ghcr.io/t-eckert/field-theories/site:*` | None | None (static site) | Public via Cloudflare Tunnel |
| bluesync | `ghcr.io/t-eckert/field-theories/bluesync:*` | `bluesync` | None (stateless) | Internal only (port 3000) |
| cloudflared | `cloudflare/cloudflared:2025.11.1` | None | None | Tunnel to Cloudflare edge |
| ~~jellyfin~~ | ~~`jellyfin/jellyfin:10.11.5`~~ | ~~None~~ | ~~2 Gi config + 5 Gi media~~ | **Removed** — will run on dedicated hardware |
| ~~filebrowser~~ | ~~`filebrowser/filebrowser:latest`~~ | ~~None~~ | ~~Shares jellyfin media + 100 Mi DB~~ | **Removed** — only existed for Jellyfin |
| minecraft | `itzg/minecraft-server:latest` | None | 10 Gi world data | `minecraft` (port 25565) |
| n8n | `n8nio/n8n:latest` | None (SQLite) | 5 Gi data | `n8n` (port 80) |
| ntfy | `binwiederhier/ntfy` | None (SQLite) | 1 Gi cache | `ntfy` (HTTPS ingress) |
| paperless | `paperless-ngx:latest` + redis + gotenberg + tika | `paperless` | 65 Gi (data+media+consume+export) | `paperless` (port 8000) |
| the-weather | `ghcr.io/t-eckert/the-weather:latest` | None | None (stateless) | `the-weather` (HTTPS ingress) |
| umami | `umami:postgresql-v2.20.2` | `umami` | None (stateless) | `umami` (HTTPS + Funnel) |
| uptime-kuma | `louislam/uptime-kuma:2.0.2` | None (SQLite) | 4 Gi data | `uptime-kuma` (HTTPS ingress) |
| workbench | `jupyter/scipy-notebook:latest` + `python:3.13-bookworm` | None | 65 Gi (data+code+notebooks) | `jupyter` (HTTPS), `workbench` (SSH port 22) |

### Monitoring

| Service | Current | NixOS Approach |
|---------|---------|----------------|
| Prometheus | Container, 10 Gi PVC, 30d retention | `services.prometheus` NixOS module or Podman container |
| Grafana | Container, 2 Gi PVC | `services.grafana` NixOS module or Podman container |
| Loki | Container, 10 Gi PVC | Podman container |
| Promtail | DaemonSet, reads /var/log/pods | Replaced by Alloy or promtail reading journald + /var/log |
| Node Exporter | DaemonSet | `services.prometheus.exporters.node` NixOS module |
| kube-state-metrics | Reports K8s object states | Eliminated (no Kubernetes) |
| postgres-exporter | Sidecar on postgres pod | `services.prometheus.exporters.postgres` NixOS module |

## Secrets Inventory

These secrets must be migrated. On NixOS, use **agenix** or **sops-nix** for
encrypted secret management in the flake.

| Secret | Contents | Used By |
|--------|----------|---------|
| `postgres-secret` / `postgres-password` | Postgres superuser password | postgres, atuin, bluesync, paperless, grafana |
| `atuin-config` | DB URI, server config | atuin |
| `bluesync-secret` | Bluesky creds, DB config, RSS config | bluesync |
| `cloudflared-credentials` | Cloudflare tunnel credentials JSON | cloudflared |
| `fave-secret` | Auth password | fave |
| `the-weather-secret` | OpenWeather API key | the-weather |
| `umami-secret` | DB URL, app secret | umami |
| `paperless-secret` | Secret key | paperless |
| `paperless/postgres-credentials` | DB user/password | paperless |
| `spark-cli-config` | Anthropic API key, GitHub token, postgres password | spark/workbench |
| `ghcr-pull-secret` | GHCR container registry credentials | field-theories, bluesync |

## Data Volumes to Migrate

Total persistent data across all PVCs: **~280 Gi allocated** (actual usage likely much less).

| PVC | Size | Mount Path | Service |
|-----|------|-----------|---------|
| `postgres-storage` | 30 Gi | `/var/lib/postgresql/data` | postgres |
| `minecraft-data` | 10 Gi | `/data` | minecraft |
| `copyparty data` | 1 Gi | `/data` | copyparty |
| `fave data` | 1 Gi | `/data` | fave |
| `n8n data` | 5 Gi | `/home/node/.n8n` | n8n |
| `ntfy cache` | 1 Gi | `/var/cache/ntfy` | ntfy |
| `paperless-data` | 5 Gi | `/usr/src/paperless/data` | paperless |
| `paperless-media` | 50 Gi | `/usr/src/paperless/media` | paperless |
| `paperless-consume` | 5 Gi | `/usr/src/paperless/consume` | paperless |
| `paperless-export` | 5 Gi | `/usr/src/paperless/export` | paperless |
| `uptime-kuma data` | 4 Gi | `/app/data` | uptime-kuma |
| `workbench-data` | 50 Gi | shared | workbench |
| `workbench-code` | 5 Gi | shared | workbench |
| `workbench-notebooks` | 10 Gi | shared | workbench |
| `spark-tools-pvc` | 5 Gi | `/mnt/tools` | spark |
| `prometheus-pvc` | 10 Gi | `/prometheus` | prometheus |
| `grafana-pvc` | 2 Gi | `/var/lib/grafana` | grafana |
| `loki-storage` | 10 Gi | `/loki` | loki |

## Migration Plan

### Phase 0: Preparation (before touching the server)

**0.1 — Build the NixOS configuration locally**

Create a new Git repository (or a `nixos/` directory in this repo) with a Nix flake
that defines the entire server configuration:

```
nixos/
├── flake.nix
├── flake.lock
├── hosts/
│   └── bee/
│       ├── configuration.nix    # Main system config
│       ├── hardware.nix         # Hardware-specific (generated)
│       ├── networking.nix       # Tailscale, firewall
│       ├── storage.nix          # Filesystem layout, mount points
│       ├── postgres.nix         # PostgreSQL config
│       ├── monitoring.nix       # Prometheus, Grafana, Loki, exporters
│       └── containers/          # Per-service Podman container definitions
│           ├── atuin.nix
│           ├── field-theories.nix
│           ├── n8n.nix
│           ├── ntfy.nix
│           ├── copyparty.nix
│           ├── fave.nix
│           ├── the-weather.nix
│           ├── umami.nix
│           ├── uptime-kuma.nix
│           ├── paperless.nix
│           ├── minecraft.nix
│           └── workbench.nix
├── secrets/
│   ├── secrets.nix              # agenix secret declarations
│   └── *.age                    # Encrypted secret files
└── deploy.nix                   # deploy-rs configuration
```

Key decisions for the flake:
- Use **Podman** (rootless where possible) via `virtualisation.oci-containers`
- Use **agenix** for secrets (age-encrypted, decrypted at activation)
- Use **deploy-rs** for remote deployments from local machine
- PostgreSQL runs as a native NixOS service (not containerized) for performance
- Node exporter and postgres exporter run as native NixOS services

**0.2 — Set up the secret management**

1. Generate an age key for the Bee Link (or use its SSH host key)
2. Encrypt all secrets from the inventory above with agenix
3. Verify decryption works in a local NixOS VM test

**0.3 — Design the storage layout**

```
/data/
├── postgres/              # PostgreSQL data directory
├── atuin/                 # (stateless, no dir needed)
├── copyparty/             # File uploads
├── fave/                  # bookmarks.json
├── field-theories/        # (stateless)
├── minecraft/             # World data
├── n8n/                   # SQLite + workflows
├── ntfy/                  # Cache DB
├── paperless/
│   ├── data/
│   ├── media/
│   ├── consume/
│   └── export/
├── uptime-kuma/           # SQLite DB
├── workbench/
│   ├── data/
│   ├── code/
│   └── notebooks/
└── monitoring/
    ├── prometheus/
    ├── grafana/
    └── loki/
```

**0.4 — Write Tailscale serve/funnel configuration**

Replace the Tailscale operator + proxy pod pattern with `tailscale serve`:

```bash
# HTTPS ingress services (replace Tailscale Ingress)
tailscale serve --bg --https=443 --set-path=/ http://localhost:8080   # fave
# ... etc

# For services needing their own hostname, use Tailscale with
# multiple serve configurations or run as separate tailscale nodes
# using tsnet or tailscale serve on different ports
```

**Important limitation**: A single Tailscale node can only serve one set of
ports. The current setup has 14+ separate Tailscale identities (one per proxy
pod). On NixOS, you have options:

- **Option A — Single Tailscale node, reverse proxy**: Run Caddy or Nginx as a
  reverse proxy. Tailscale exposes ports 80/443. The reverse proxy routes by
  subdomain (e.g., `ntfy.bee.ts.net`, `n8n.bee.ts.net`). This requires
  enabling Tailscale MagicDNS HTTPS certs.

- **Option B — Multiple Tailscale nodes via tsnet**: Some services embed a
  Tailscale node using tsnet. More complex but preserves the current
  per-service hostname model.

- **Option C — Tailscale Serve with path-based routing**: A single hostname
  with different paths. Doesn't work well for most apps.

**Recommended: Option A** — a single Caddy reverse proxy with Tailscale HTTPS.
Consolidates 14 proxy pods into 1 lightweight process. Services that need
non-HTTP ports (postgres:5432, minecraft:25565, workbench SSH:22) use Tailscale
`serve --tcp`.

**0.5 — Test in a VM**

Before touching the real hardware:
1. Build a NixOS VM image from the flake (`nix build .#nixosConfigurations.bee.config.system.build.vm`)
2. Verify all systemd services start
3. Verify Podman containers pull and run
4. Verify PostgreSQL starts and accepts connections
5. Verify Caddy reverse proxy routes correctly
6. Verify monitoring stack scrapes targets

### Phase 1: Backup Everything

**1.1 — Dump PostgreSQL databases**

```bash
# From local machine via Tailscale
pg_dump -h postgres.feist-gondola.ts.net -U postgres -d homelab > homelab.sql
pg_dump -h postgres.feist-gondola.ts.net -U postgres -d atuin > atuin.sql
pg_dump -h postgres.feist-gondola.ts.net -U postgres -d bluesync > bluesync.sql
pg_dump -h postgres.feist-gondola.ts.net -U postgres -d paperless > paperless.sql
pg_dump -h postgres.feist-gondola.ts.net -U postgres -d umami -U umami > umami.sql
```

**1.2 — Copy all PVC data off the node**

```bash
# SSH into the Talos node (or use talosctl cp)
# PVC data lives under /var/local-path-provisioner/ on the node

# From local machine, rsync everything over Tailscale
rsync -avz --progress \
  root@homelab.feist-gondola.ts.net:/var/local-path-provisioner/ \
  ./backup/pvc-data/
```

Note: Talos doesn't have SSH by default. Options:
- Use `talosctl cp` to copy files out
- Use `talosctl read` for individual files
- Mount the NVMe on another machine
- Use the workbench debian container (which has SSH) to tar and transfer data

**1.3 — Export Grafana dashboards**

```bash
# Use Grafana API via Tailscale
curl -s "https://grafana.feist-gondola.ts.net/api/search" | \
  jq -r '.[].uid' | while read uid; do
    curl -s "https://grafana.feist-gondola.ts.net/api/dashboards/uid/$uid" \
      > "backup/grafana-dashboards/${uid}.json"
  done
```

**1.4 — Save Tailscale ACLs and DNS configuration**

Document the current Tailscale network configuration, Funnel settings, and any
ACL rules that reference the homelab services.

**1.5 — Record current Cloudflare Tunnel config**

The tunnel ID is `d870bd1a-445f-4fd8-9a14-2e9b08589018`. Save the credentials
JSON and tunnel configuration for re-use on NixOS.

**1.6 — Verify backups**

- Restore postgres dumps to a local postgres instance and spot-check
- Verify Grafana dashboard JSON is valid
- Verify PVC data files are complete (check file counts, sizes)

### Phase 2: Install NixOS

**2.1 — Create NixOS installer USB**

```bash
# Build a custom installer with Tailscale pre-configured for remote access
nix build .#nixosConfigurations.installer.config.system.build.isoImage
```

Or download the standard NixOS minimal ISO.

**2.2 — Install NixOS on the Bee Link**

1. Connect monitor + keyboard to the Bee Link
2. Boot from USB
3. Partition the NVMe:
   ```
   /dev/nvme0n1p1  512M   EFI System Partition (FAT32)  /boot
   /dev/nvme0n1p2  16G    swap
   /dev/nvme0n1p3  rest   ext4 or btrfs                 /
   ```
   Consider **btrfs** for snapshots and compression (good for a homelab where
   you want easy rollback).
4. Run `nixos-generate-config` to produce `hardware.nix`
5. Copy the flake onto the machine and run `nixos-install --flake .#bee`
6. Reboot into NixOS

**2.3 — Verify base system**

- SSH in (NixOS has sshd by default if enabled in config)
- Verify Tailscale connects: `tailscale status`
- Verify filesystem layout: `ls /data/`
- Verify Podman works: `podman run --rm hello-world`

### Phase 3: Restore Data

**3.1 — Restore PostgreSQL**

```bash
# PostgreSQL is running as a NixOS service
sudo -u postgres createdb atuin
sudo -u postgres createdb bluesync
sudo -u postgres createdb paperless
sudo -u postgres createdb umami
sudo -u postgres createuser umami  # with password

sudo -u postgres psql homelab < homelab.sql
sudo -u postgres psql atuin < atuin.sql
sudo -u postgres psql bluesync < bluesync.sql
sudo -u postgres psql paperless < paperless.sql
sudo -u postgres psql umami < umami.sql
```

**3.2 — Restore application data**

```bash
# Copy backed-up PVC data into the new /data/ layout
rsync -av backup/pvc-data/n8n/ /data/n8n/
rsync -av backup/pvc-data/ntfy-cache/ /data/ntfy/
rsync -av backup/pvc-data/uptime-kuma/ /data/uptime-kuma/
rsync -av backup/pvc-data/workbench-data/ /data/workbench/data/
rsync -av backup/pvc-data/workbench-code/ /data/workbench/code/
rsync -av backup/pvc-data/workbench-notebooks/ /data/workbench/notebooks/
rsync -av backup/pvc-data/copyparty/ /data/copyparty/
rsync -av backup/pvc-data/fave/ /data/fave/
rsync -av backup/pvc-data/minecraft/ /data/minecraft/
rsync -av backup/pvc-data/paperless-data/ /data/paperless/data/
rsync -av backup/pvc-data/paperless-media/ /data/paperless/media/
# ... etc

# Fix ownership
chown -R 1000:1000 /data/jellyfin/
# ... set per-service as needed
```

**3.3 — Restore Grafana dashboards and monitoring data**

```bash
rsync -av backup/pvc-data/grafana/ /data/monitoring/grafana/
rsync -av backup/pvc-data/prometheus/ /data/monitoring/prometheus/
rsync -av backup/pvc-data/loki/ /data/monitoring/loki/
```

### Phase 4: Start Services and Validate

**4.1 — Start services in dependency order**

The NixOS configuration should define service dependencies, but for the initial
bringup, verify in this order:

1. **Tailscale** — `tailscale status` shows connected
2. **PostgreSQL** — `psql -U postgres -d homelab -c 'SELECT 1'`
3. **Caddy** (reverse proxy) — `systemctl status caddy`
4. **Stateless apps first**: the-weather, fave, atuin, umami, field-theories
5. **Stateful apps**: n8n, ntfy, uptime-kuma, copyparty
6. **Heavy apps**: workbench, paperless (if enabled)
7. **Monitoring**: prometheus, grafana, loki, node-exporter

**4.2 — Configure Tailscale serve/funnel**

```bash
# Set up Caddy as the HTTPS frontend
# Caddy config (in NixOS module) handles TLS via Tailscale

# TCP passthrough for non-HTTP services
tailscale serve --bg tcp:5432 tcp://localhost:5432    # postgres
tailscale serve --bg tcp:25565 tcp://localhost:25565  # minecraft
tailscale serve --bg tcp:22 tcp://localhost:2222      # workbench SSH
```

**4.3 — Validate each service**

For each service, verify:
- [ ] Container/service is running (`systemctl status` / `podman ps`)
- [ ] Data is intact (login, check content)
- [ ] Tailscale hostname resolves and connects
- [ ] Monitoring target is scraped by Prometheus

**4.4 — Restore Cloudflare Tunnel**

```bash
# cloudflared runs as a Podman container or systemd service
# Use the same tunnel ID and credentials
cloudflared tunnel --config /etc/cloudflared/config.yml run
```

Verify `fieldtheories.blog` resolves to the field-theories container.

**4.5 — Restore Grafana datasources and dashboards**

Import the saved dashboard JSON files via the Grafana API or by placing them
in the provisioning directory.

### Phase 5: Establish GitOps Workflow

Replace Flux with a NixOS-native deployment workflow:

**Option A — deploy-rs (recommended)**

```bash
# From local machine
deploy .#bee
# Builds the NixOS config, copies closure to server, activates
```

The workflow becomes:
1. Edit Nix files in the repo
2. `git commit && git push`
3. `deploy .#bee` (or set up a GitHub Action to deploy on push)

**Option B — nixos-rebuild over SSH**

```bash
nixos-rebuild switch --flake .#bee --target-host root@homelab.feist-gondola.ts.net
```

Simpler but less robust (no automatic rollback on failure).

**Image updates**: Without Flux ImagePolicy, set up a simple script or GitHub
Action that checks GHCR for new tags on `field-theories`, `bluesync`,
`fave`, `the-weather` and updates the pinned image tags in the Nix config.

### Phase 6: Cleanup

- [ ] Remove old Talos configuration from the repo (or archive it)
- [ ] Update `CLAUDE.md` to reflect the NixOS setup
- [ ] Remove the Tailscale node for the old Talos machine from Tailscale admin
- [ ] Remove stale Tailscale proxy devices (14 of them) from the admin console
- [ ] Update Taskfile commands for the new setup
- [ ] Update `notebook/Hardware.md` with NixOS details

## Example NixOS Container Definition

Here's what atuin looks like as a Podman container in NixOS:

```nix
# containers/atuin.nix
{ config, ... }:
{
  virtualisation.oci-containers.containers.atuin = {
    image = "ghcr.io/atuinsh/atuin:latest";
    cmd = [ "atuin" "server" "start" ];
    ports = [ "8888:8888" ];
    environment = {
      ATUIN_HOST = "0.0.0.0";
      ATUIN_PORT = "8888";
      ATUIN_OPEN_REGISTRATION = "true";
      ATUIN_MAX_HISTORY_LENGTH = "8192";
      ATUIN_PAGE_SIZE = "100";
    };
    environmentFiles = [
      config.age.secrets.atuin-db-uri.path  # Contains ATUIN_DB_URI=...
    ];
  };
}
```

## Example Caddy Reverse Proxy Configuration

```nix
# networking.nix (partial)
services.caddy = {
  enable = true;
  virtualHosts = {
    "ntfy.bee.ts.net".extraConfig = ''
      reverse_proxy localhost:8080
    '';
    "fave.bee.ts.net".extraConfig = ''
      reverse_proxy localhost:8080
    '';
    # ... one entry per service
  };
};
```

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Data loss during migration | High | Full backup in Phase 1, verify before wiping |
| Extended downtime | Medium | Build and test the full NixOS config in a VM first |
| Tailscale hostname changes | Medium | Document all current hostnames, update clients |
| Missing secret or config | Medium | Detailed inventory above, checklist validation |
| NixOS learning curve | Low | Declarative config is well-documented, rollback is easy |
| Podman compatibility issues | Low | All images are standard OCI, Podman is Docker-compatible |
| Losing Flux image auto-updates | Low | Replace with a cron job or GitHub Action |

## Timeline Estimate

| Phase | Duration | Notes |
|-------|----------|-------|
| Phase 0: Preparation | 2-4 days | Build and test NixOS config in VM |
| Phase 1: Backup | 1-2 hours | Mostly automated |
| Phase 2: Install NixOS | 30-60 min | Physical access needed |
| Phase 3: Restore Data | 1-2 hours | rsync + pg_restore |
| Phase 4: Validate | 2-4 hours | Manual verification of each service |
| Phase 5: GitOps Setup | 1-2 hours | deploy-rs or nixos-rebuild |
| Phase 6: Cleanup | 30 min | |
| **Total downtime** | **~4-8 hours** | Phases 2-4 require the server to be offline |

## Decision Points

Before starting, decide on:

1. **Reverse proxy**: Caddy (recommended, auto-TLS, simple config) vs Nginx
2. **Secrets management**: agenix (recommended, SSH key-based) vs sops-nix (GPG/age-based)
3. **Filesystem**: btrfs (snapshots, compression) vs ext4 (simpler, proven)
4. **Container runtime**: Podman rootless (recommended, more secure) vs Docker
5. **Deploy method**: deploy-rs (recommended, atomic with rollback) vs nixos-rebuild
6. **Monitoring**: Keep current stack (Prometheus + Grafana + Loki) or simplify
7. **Tailscale routing**: Single node + Caddy (recommended) vs multiple tsnet nodes
