# Raspberry Pi Provisioning

How to image a Pi with Raspberry Pi OS, get it onto the network headless, and record it in
[Hardware.md](./Hardware.md) so it stays identifiable.

For Talos on a Pi (the Kubernetes path) see [Setting Up Talos on Raspberry Pi](./Setting%20Up%20Talos%20on%20Raspberry%20Pi.md)
instead. Talos has no SSH and is configured with `talosctl`.

## Why this document exists

Hardware.md originally recorded each machine by **IP address**. Changing routers invalidated
every one of them at once, and with two visually identical Pi 4 Bs — one 1 GB, one 4 GB —
there was no way left to tell which was which without booting them.

The fix is to key on identifiers that don't change:

- **Serial number** — burned into the SoC
- **MAC addresses** — `eth0` and `wlan0`, assigned at manufacture
- **Board revision code** — encodes the model and RAM size

IP address is still worth noting, but as a *last seen* value, not an identity.

## Imaging

Use the Raspberry Pi Imager GUI (`/Applications/Raspberry Pi Imager.app`). It has a CLI at
`Contents/MacOS/rpi-imager`, but the OS-customisation dialog is what makes headless Wi-Fi
work, and that's GUI-only.

1. **Device:** the specific model — Imager tailors the bootloader to it
2. **OS:** Raspberry Pi OS Lite (64-bit) unless there's a reason to want a desktop
3. **Storage:** confirm with `diskutil list` first; the SD reader usually shows as an
   *internal, physical* disk, not external

In **Edit Settings**:

| Field | Notes |
|---|---|
| Hostname | Becomes `<hostname>.local` over mDNS — how you'll find the Pi |
| Username / password | |
| SSH | *Allow public-key authentication only*, paste `~/.ssh/id_ed25519.pub` |
| Wi-Fi SSID | Must match exactly, including spaces and apostrophes |
| Wi-Fi password | |
| **Wireless LAN country** | **Required.** See below. |

### Two ways this silently fails

**Missing country code.** Without a regulatory domain, `wlan0` stays rfkill-blocked on 5 GHz.
If the AP is 5 GHz-only, or the Pi gets steered onto a 5 GHz band, the board boots with no
network and no way to tell you why. Set the country (**CA** here) every time.

**Nearly identical SSIDs.** Check the exact name against the router before typing it — a
household often accumulates several similar ones, and a typo looks the same as a hardware
failure from the outside.

Confirm the customisation was written before ejecting:

```bash
grep -o firstrun /Volumes/bootfs/cmdline.txt
```

## Finding the Pi after first boot

First boot takes a minute or two — it resizes the root filesystem and reboots itself.

```bash
ping -c2 <hostname>.local
arp -a | grep -iE "dc:a6:32|e4:5f:01|b8:27:eb"   # Raspberry Pi OUIs
```

If the Mac is on more than one network, check each subnet — a Pi on Ethernet won't be on the
same one as a Pi on Wi-Fi.

## Recording identity

```bash
task pi:identity HOST=<hostname>.local
```

This prints a Markdown block ready to paste into Hardware.md:

```
### Raspberry Pi 4 Model B Rev 1.1

Serial: `10000000abcdef12`
MAC (eth0): `dc:a6:32:xx:xx:xx`
MAC (wlan0): `dc:a6:32:xx:xx:xx`
Board revision: `a03111`
RAM: 1 GB
...
```

It cross-checks `MemTotal` against the revision code and warns if they disagree. The first
nibble of a Pi 4 B revision encodes RAM: `a` = 1 GB, `b` = 2 GB, `c` = 4 GB, `d` = 8 GB.
`MemTotal` always reads a little low because of the GPU carve-out, so the revision code is the
more reliable of the two.

**Then put a physical label on the board** — serial last-4 and RAM size. Everything above is
bookkeeping; the label is what actually prevents a repeat.

## RTL-SDR receivers

```bash
sudo apt update && sudo apt install -y rtl-sdr librtlsdr-dev
```

**Blacklist the DVB driver.** The kernel's `dvb_usb_rtl28xxu` claims the dongle on sight, and
every SDR tool then fails with `usb_claim_interface error -6`. This is the most common
RTL-SDR problem by a wide margin:

```bash
sudo tee /etc/modprobe.d/blacklist-rtlsdr.conf <<'EOF'
blacklist dvb_usb_rtl28xxu
blacklist rtl2832
blacklist rtl2830
EOF
sudo reboot
```

Verify against the hardware, not the config:

```bash
rtl_test -t     # expect a device found, plus the tuner type
```

Serve it on the network so other machines can use the dongle:

```bash
sudo tee /etc/systemd/system/rtl-tcp.service <<'EOF'
[Unit]
Description=RTL-SDR TCP server
After=network-online.target

[Service]
ExecStart=/usr/bin/rtl_tcp -a 0.0.0.0 -p 1234
Restart=always

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl enable --now rtl-tcp
```

Notes:

- The **RTL-SDR Blog V4** needs the `rtl-sdr-blog` driver fork for HF. Stock `librtlsdr` is
  fine above ~28 MHz, so ADS-B at 1090 MHz works either way, but AM/shortwave won't.
- Give the Pi a genuine **3 A** supply. An underpowered Pi 4 with a dongle attached drops
  samples intermittently, which reads as a software bug.

## Home Assistant OS

HAOS is flashed as its own image (*Other specific-purpose OS → Home Assistant*), not installed
on top of Raspberry Pi OS.

**Imager's customisation dialog does not apply to HAOS** — no hostname, SSH key, or Wi-Fi is
injected. Headless Wi-Fi requires a separate USB stick holding a NetworkManager profile at
`CONFIG/network/my-network`. Use Ethernet for the install instead; it's far less trouble.

Onboarding is at `http://homeassistant.local:8123`.

## Cross-compiling for a Pi from an Apple Silicon Mac

The Mac and the Pi are both `aarch64`, so a Linux container builds Pi binaries at native
speed — no QEMU:

```bash
podman machine start
podman run --rm -v "$PWD":/w -w /w docker.io/library/rust:1-trixie \
  cargo build --release
```

Match the container's Debian release to the Pi's, so glibc lines up. Raspberry Pi OS currently
tracks **trixie**. Building on a 1 GB Pi directly also works, but wants a swapfile and a lot of
patience.
