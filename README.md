# LibWake

LibWake lets you **start Unraid VMs using Wake-on-LAN (WOL)**.  
Send a WOL magic packet to a VM’s MAC address and LibWake will start that VM via libvirt.

## Features

- Listens for WOL magic packets (UDP ports **7/9** by default)
- Supports “Ethernet WOL” frames (EtherType **0x0842**)
- Starts VMs by matching the **target MAC** to a VM NIC MAC
- Per‑VM allow list in the Unraid WebGUI

## Requirements

- Unraid **7.x** with **VM Manager** enabled
- Your WOL sender must be on the same LAN (or routed correctly) and able to reach the Unraid host

## Install on Unraid

1. Unraid WebGUI → **Plugins** → **Install Plugin**
2. Paste this URL and click **Install**:

```text
https://raw.githubusercontent.com/MiranoVerhoef/LibWake/main/plugin/libwake.plg
```

Releases (binaries) are published here:

```text
https://github.com/MiranoVerhoef/LibWake/releases
```

## Configure

Unraid WebGUI → **Settings** → **VM Manager** → **LibWake**

- **Enable daemon**
- Set **Listen interface** (usually `br0`)
- (Optional) Set **Allow subnets** (comma-separated CIDRs)
- Select which **VMs** may be started by WOL packets

## Usage

Send a Wake-on-LAN magic packet to the MAC address of the VM you enabled in LibWake.

## Logs & troubleshooting

- Log file:
  - `/var/log/libwake/libwake.log`

- Service status:
  - ` /etc/rc.d/rc.libwake status `
