# Hardware

## Compute

### Bee Link Mini PC

Hostname: `homelab`  
Local IP: `10.0.0.67`  
Tailscale Address: `https://homelab.fiest-gondola.ts.net`  
CPU: Intel Twin Lake N150 Processor  
Memory: 16 GB  
Storage: 1 TB  
Network: WiFi 6, LAN 1000M  
Input: 100-240V, 1.9A 50/60Hz  

### Raspberry Pi 4 B

Hostname: `homeassistant`  
Tailscale Address: `https://homeassistant.feist-gondola.ts.net` (Tailscale add-on `share_homeassistant`; plain HTTP on the LAN is port 80, not 8123)  
wlan0 MAC: `d8:3a:dd:a1:fa:06`  
IP (last seen): `192.168.68.62` on the home Wi-Fi (Wi-Fi only, no Ethernet)  
OS: Home Assistant OS 18.3 (`haos_rpi4-64`), flashed 2026-10-02 to a SanDisk 256 GB SDXC  
Role: Home Assistant  
RAM: 4 GB  
ARCH: ARMv8 64 bit  

### Raspberry Pi 4 B

IP: `10.0.0.70`  
OS: [OpenWebRX+ 1.2.102](https://github.com/luarvique/openwebrx/releases)
Role: Software Defined Radio Client  
RAM: 1 GB  
ARCH: ARMv8 64 bit  

### Raspberry Pi 3 B v2

IP: `10.0.0.232`  
OS: Raspbian  
Role: Linux play environment  
RAM: 1 GB  
ARCH: ARMv8 64 bit  

## Network

### TP-Link TL-SG105 (current)

Type: 5-port unmanaged gigabit switch  
Role: Stopgap switch for the Pis until the mini rack is set up  
Form factor: Fanless desktop (not rack-mounted)  
Notes: Plug-and-play, nothing to configure. Will become a spare once the mini-rack switch takes over.

### Upgrade path (planned)

When the 10" mini rack is set up (also housing a future Framework Desktop), replace the TL-SG105 with a rack switch that has 2.5G+ ports and an SFP+ cage — the Framework Desktop's onboard NIC is 5GbE (Realtek RTL8126), so plain gigabit would bottleneck it. Candidates:

- **MikroTik CRS310-8G+2S+IN** (~$220) — 8× 2.5GbE + 2× SFP+, fanless, no PoE.
- **TP-Link TL-SG2210XMP-M2** (~$250) — 8× 2.5G PoE + 2× SFP+, fanless; PoE could power Pi 5 HATs.

Gotcha: 10" racks are shallow — confirm switch depth fits the specific rack before buying.
