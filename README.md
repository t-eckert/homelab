# Homelab

This repo contains all of the scripts, definitions, and notes for my homelab.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Talos Linux K8s Cluster                             │
│                         Bee Machine (10.0.0.67)                             │
│                    Talos v1.11.5 • Kubernetes v1.34.2                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ System Components                                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Flux CD                    - GitOps continuous delivery                  │
│  • Local Path Provisioner     - Dynamic PV provisioning                     │
│  • Tailscale Operator         - Secure external access                      │
│  • Generic Device Plugin      - TUN device access for Tailscale             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ Applications & Services                                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Personal Projects                                                          │
│  ├─ Artemis                   - Job search application                      │
│  ├─ Field Theories            - Personal blog (Astro)                       │
│  │  └─ Cloudflared           - Cloudflare Tunnel                           │
│  └─ Field Theories BS Sync    - BlueSky integration (Rust)                  │
│                                                                             │
│  Media & File Management                                                    │
│  ├─ Jellyfin                  - Media server                                │
│  │  └─ FileBrowser           - Web-based file manager                      │
│  ├─ CopyParty                 - File sharing server                         │
│  └─ Fave                      - Bookmark manager                            │
│                                                                             │
│  Infrastructure & Utilities                                                 │
│  ├─ PostgreSQL                - Shared database server                      │
│  ├─ NTFY                      - Push notification service                   │
│  ├─ Uptime Kuma               - Uptime monitoring                           │
│  └─ Umami                     - Web analytics                               │
│                                                                             │
│  Monitoring Stack                                                           │
│  ├─ Grafana                   - Dashboards & visualization                  │
│  ├─ Prometheus                - Metrics collection                          │
│  ├─ Loki                      - Log aggregation                             │
│  ├─ Promtail                  - Log shipping                                │
│  └─ Kube State Metrics        - Kubernetes metrics                          │
│                                                                             │
│  Development & Gaming                                                       │
│  ├─ Workbench                                                               │
│  │  ├─ Debian                 - Development container                       │
│  │  └─ Jupyter Lab            - Interactive notebooks                       │
│  └─ Minecraft Server          - Game server (scaled on demand)              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ Separate Infrastructure                                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Home Assistant (10.0.0.195)  - Raspberry Pi 4 B                            │
└─────────────────────────────────────────────────────────────────────────────┘

Legend: StatefulSets use persistent storage • Tailscale provides secure ingress
```

## What I'm Running

### Cluster

This is a [Talos Linux](https://www.talos.dev/) cluster where I run most of my applications. The configuration for this cluster is in the `cluster/` directory, organized into:
- `cluster/apps/` - User-facing applications
- `cluster/system/` - Core infrastructure components
- `cluster/infrastructure/` - Shared backing services

- **Artemis**: An agentic app I wrote for finding job postings that match my résumé.
- [Copy Party](https://github.com/9001/copyparty): A neato file server.
- [Field Theories](https://fieldtheories.blog): My personal blog built in Astro.
- **Field Theories BlueSky Sync**: A lightweight Rust application for posting new blog posts to BlueSky and syncing back responses as comments on them. (STILL IN PROGRESS)
- [Flux](https://fluxcd.io/): A popular, lightweight CI operator for auto-updating the cluster based on Git config.
- [Jellyfin](https://jellyfin.org/): For media management
- [Minecraft Server](https://github.com/itzg/docker-minecraft-server): It's Minecraft! I scale this up and down because it's resource intensive-ish
- **Monitoring**: Grafana+Prometheus+Loki+Promtail monitoring setup
- [NTFY](https://ntfy.sh/): Self hosted push notifications
- **Postgres**: I use this Postgres server to back my apps on the homelab, run development databases, and 
- [Umami](https://umami.is/): Analytics platform
- [Uptime Kuma](https://github.com/louislam/uptime-kuma): An uptime monitoring application
- **Workbench**: A Debian instance and [Jupyter Lab](https://jupyter.org/) with a shared drive for personal data analysis projects

### Home Assistant

This runs on a dedicated Raspberry Pi 4 B.




