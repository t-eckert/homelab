# Home Assistant Setup

A plan for taking the Home Assistant install from "freshly installed" to a solid, well-maintained setup, following community best practices.

## Current State (2026-10-02)

- **Hardware**: Raspberry Pi 4, HA OS 18.3, HA Core 2026.9.4, running from a **256 GB SD card**
- **Access**: Tailscale add-on, `https://homeassistant.feist-gondola.ts.net`
- **Integrations**: Sonos (Living Room One SL), Google Cast (Family Room TV), Brother printer (IPP), TP-Link BE65 router (UPnP), Met.no weather, Raspberry Pi, Bluetooth, Assist MCP server
- **Floors/areas**: Basement, Ground, First Floor, Second Floor, with 13 areas (set up 2026-10-02)
- **Gaps**:
  - ~~No backups have ever run~~ (daily encrypted backups to R2 since 2026-10-02)
  - No automations, scripts, or scenes
  - The `person.thomas_eckert` entity has no device tracker, so presence is `unknown`
  - Only one user (Thomas); no person/user for the second household member
  - ~~The unit system is imperial~~ (switched to Metric 2026-10-02)
  - Most devices are not assigned to an area
  - No HACS (the SSH & Web Terminal add-on was installed 2026-10-02)

## Phase 1: Foundation (do first)

1. **Automatic backups**: Settings → System → Backups. Turn on daily automatic backups with encryption, keep about 7 copies, and save the emergency kit (the encryption key) in 1Password. Add at least one **off-device** location:
   - Network storage (SMB/NFS share) on another machine in the house, and/or
   - A cloud target (Home Assistant Cloud, Google Drive, OneDrive, Backblaze B2 or S3 backup integrations)
2. **Get off the SD card**: SD cards are the most common cause of Pi HA failures, because the database writes wear them out. Choose one:
   - Boot the Pi 4 from a USB SSD (restore from a backup onto it), or
   - Move to a mini PC / Home Assistant Green and restore from a backup.

   Do this *after* backups work, since a backup restore is the migration path.
3. ~~**Units and locale**: set the unit system to Metric.~~ Done 2026-10-02.
4. **Companion app** on each phone (iOS/Android): this provides presence (fixes the `unknown` person), push notifications, actionable notifications, and phone sensors (battery, Wi-Fi, focus mode).
5. **Users and people**: create a user for the second household member (non-admin) and link their phone through the Companion app. Optionally make a separate admin account for yourself and a day-to-day non-admin one.
6. **Assign devices to areas**: Family Room TV, router, Pi. Consider adding a "Family Room" area if that TV lives somewhere other than the Living Room.

## Phase 2: Tooling Folks Find Useful

- **Add-ons**
  - *Studio Code Server* or *File editor*, for editing YAML in the browser
  - *Advanced SSH & Web Terminal*, for shell access and `ha` CLI
    - **Done 2026-10-02**: `ssh homeassistant` (user `hassio`, over Tailscale), using the 1Password key "Home Assistant". The host entry lives in dotfiles `nix/home/ssh.nix`. vi/vim/nvim and the `ha` CLI are available.
- **HACS** (Home Assistant Community Store): the de-facto way to get community integrations and dashboard cards. Popular picks:
  - Dashboard: Mushroom cards, Bubble Card, card-mod, auto-entities, mini-graph-card
  - Integrations: Adaptive Lighting (once there are smart lights), Powercalc
- **Config in Git**: keep `configuration.yaml`, `automations.yaml`, and a `packages/` directory in a git repo (the Git pull add-on, or push from Studio Code Server). This matches how the rest of the homelab is managed with Flux.

## Phase 3: Tie Into the Homelab

- **Notifications via ntfy**: the cluster already runs ntfy. Use the ntfy integration (or a REST notify) so HA alerts land in the same place as other homelab alerts.
- **Metrics**: enable the Prometheus integration and scrape it from the cluster's monitoring stack for long-term graphs (HA's own recorder should keep only around 10 days, to spare the disk).
- **Uptime Kuma** (deferred): Kuma runs on Fly (`uptime-kuma-fly/`) and is not on the tailnet, so it can't poll HA. Use **Push monitors** instead:
  - *Heartbeat*: an HA automation pushes every 5 min; Kuma alerts after about 10 min of silence.
  - *Backup*: HA pushes after each successful automatic backup; Kuma alerts after about 25 h of silence.
  - Wiring: `rest_command` entries with push URLs in `secrets.yaml`, plus two automations. The push URLs go in 1Password.

## Phase 4: Devices and Automations

This depends on goals. The common next steps are:

- **A radio for local, cloud-free devices**: a Zigbee/Thread coordinator such as Home Assistant Connect ZBT-2 or an SMLIGHT SLZB-06 (network-attached, which is good for a Pi in a closet). It lets you use cheap, reliable sensors (door/window, motion, temperature, leak) and smart plugs/bulbs.
  - ZHA (built in) is the simplest choice; Zigbee2MQTT + Mosquitto has broader device support.
- **Basement sensors**: leak sensors near the water heater/furnace in the HVAC room, and temperature/humidity in the storage room (dehumidifier automation).
- **Thermostat integration**, if the thermostat is smart (Ecobee, Nest, etc.)
- **Good first automations**:
  - Notify on water leak
  - Garage door left open after dark
  - Lights on at sunset when someone is home
  - Pause Sonos when the doorbell rings
  - Printer toner low → add to the shopping list
- **Dashboards**: use the *Sections* view with one section per floor/area, and keep a separate mobile-first dashboard.
- **Energy dashboard**: if smart plugs or a whole-home energy monitor are added later.

## Decisions (2026-10-02)

- **Existing devices**: one TP-Link bulb, possibly not on Wi-Fi yet. The goal is a foundation to expand on later.
- **Goals**: convenience and energy use.
- **Backups**: off-site to the cloud.
- **Hardware**: keep the Pi 4 and move it to a USB SSD eventually.

### Implications

- **TP-Link bulb**: put it on Wi-Fi with the Kasa or Tapo app, then add the *TP-Link Smart Home* integration, which controls it locally.
- **Energy**: TP-Link energy-monitoring plugs (Kasa KP115 / Tapo P110) use the same integration and feed the Energy dashboard per device. A whole-home monitor can come later.
- **Done**: unit system switched to Metric; weather entity set to °C / hPa / km/h / mm / km.
- **Done**: backups. The *Cloudflare R2* integration (not *AWS S3*, which rejects non-AWS endpoints) writes to the `homeassistant-backups` R2 bucket (Standard class). Backups run daily at 04:45, are encrypted, keep 7 copies, and go to both R2 and the Pi's local storage. Credentials and the **backup encryption key** are in 1Password: `Private/Home Assistant R2 Backups`.
  - Open: the Supervisor still raises `no_current_backup`, because HA's backup manager makes `partial`-type backups.
