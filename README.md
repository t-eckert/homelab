# Homelab

This repo contains all of the scripts, definitions, and notes for my homelab.

## What I'm Running

### Cluster

This is a [Talos Linux](https://www.talos.dev/) cluster where I run most of my applications. The configuration for this cluster is in the `cluster/` directory.

- **Artemis**: An agentic app I wrote for finding job postings that match my résumé.
- [Copy Party](https://github.com/9001/copyparty): A neato file server.
- [Field Theories](https://fieldtheories.blog): My personal blog built in Astro.
- **Field Theories BlueSky Sync**: A lightweight Rust application for posting new blog posts to BlueSky and syncing back responses as comments on them. (STILL IN PROGRESS)
- [Flux](https://fluxcd.io/): A popular, lightweight CI operator for auto-updating the cluster based on Git config.
- [Minecraft Server](https://github.com/itzg/docker-minecraft-server): It's Minecraft! I scale this up and down because it's resource intensive-ish
- **Monitoring**: Grafana+Prometheus+Loki+Promtail monitoring setup
- [NTFY](https://ntfy.sh/): Self hosted push notifications
- **Postgres**: I use this Postgres server to back my apps on the homelab, run development databases, and 
- [Umami](https://umami.is/): Analytics platform
- [Uptime Kuma](https://github.com/louislam/uptime-kuma): An uptime monitoring application
- **Workbench**: A Debian instance and [Jupyter Lab](https://jupyter.org/) with a shared drive for personal data analysis projects

### Home Assistant

This runs on a dedicated Raspberry Pi 4 B.




