# LibWake

LibWake listens for Wake-on-LAN (WOL) packets and starts matching Unraid VMs.

- Per‑VM enable/disable toggle integrated into **Settings → VM Manager** (same style as libvirtwol).
- Optional daemon listening on UDP (7/9) and raw ethernet WOL frames (ethertype 0x0842).

## Install (manual)

In Unraid:
1. Go to **Plugins** → **Install Plugin**
2. Paste:

```
https://raw.githubusercontent.com/MiranoVerhoef/LibWake/main/plugin/libwake.plg
```

## Where to find it

- **Settings → VM Manager**: LibWake section (per‑VM toggles)
- **Settings → User Utilities → LibWake**: LibWake settings (daemon config)

## License

GPLv2. See `COPYING`.
