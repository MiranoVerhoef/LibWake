# LibWake

LibWake listens for Wake-on-LAN (WOL) packets and starts the matching Unraid VM via libvirt.

## Install (manual)

In Unraid:
1. Go to **Plugins** → **Install Plugin**
2. Paste this URL:

```
https://raw.githubusercontent.com/MiranoVerhoef/LibWake/main/plugin/libwake.plg
```

## Where to find it

After installing, open:
**Settings → User Utilities → LibWake**

## Notes

- Enable LibWake on the settings page, then click **Apply**.
- Default UDP port is **9** (you can set **7,9** if you want both).
