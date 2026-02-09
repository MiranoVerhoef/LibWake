# LibWake (Unraid)

A modern Wake-on-LAN → VM start daemon + Unraid plugin.

This project aims to combine the **clean VM-integrated UX** of the older unRAID-libvirtwol plugin
with the **clean, portable daemon approach** of virtwold, updated for modern Unraid.

## What it does

- Listens for WOL magic packets (UDP ports 7/9 by default)
- Optionally listens for the EtherType `0x0842` “Ethernet” style WOL frame
- Matches the target MAC to a VM's NIC MAC address
- Starts the VM via `virsh start <vm>` (if it isn't already running)
- Lets you enable/disable WOL on a per-VM basis in the Unraid WebGUI (initially via a settings page)

## Install (developer preview)

1. Publish this repo to GitHub.
2. Create a release that includes a Linux amd64 binary named:
   - `libwake-linux-amd64`
3. In Unraid, go to **Plugins → Install Plugin** and paste the raw URL to:
   - `plugin/libwake.plg`

> The `.plg` file contains placeholders (repo path, version). Adjust entities at the top of
> `plugin/libwake.plg` to match your GitHub repo.

## Configure

After install:

- Settings → VM Manager → LibWake
- Enable the daemon
- Choose the interface (usually `br0`)
- Select which VMs may be started

## Files

- Go daemon: `cmd/libwake`
- Unraid plugin artifacts: `plugin/`

## Roadmap

- Inject per-VM toggle directly into the VM edit page (“Advanced view”) like the classic plugin
- Auto-reload mappings without restart
- GitHub Actions: build binary + publish release assets

## License

MIT
