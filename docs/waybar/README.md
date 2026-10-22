# Waybar module

[`happ-tui.sh`](happ-tui.sh) turns `happ-tui status --json --watch` into waybar's JSON. It runs for as long as waybar does: the stream prints a line on every change and every 2 s while connected, and keeps going while the core restarts, so no polling or refresh signals are needed. It needs `jq`.

For TUN mode on click without a password, install with `sudo ./install.sh --service` (see the main README).

## Setup
```
install -Dm755 happ-tui.sh ~/.config/waybar/scripts/happ-tui.sh
```

`~/.config/waybar/config.jsonc`:
```jsonc
"modules-right": ["custom/vpn", /* ... */],

"custom/vpn": {
  "exec": "~/.config/waybar/scripts/happ-tui.sh",
  "return-type": "json",
  "restart-interval": 5,
  "on-click": "happ-tui toggle --tun",
  "on-click-right": "foot --app-id happ-tui happ-tui",
  "tooltip": true
}
```
`restart-interval` restarts the script if `happ-tui` itself is missing for a moment, e.g. during an upgrade.

`~/.config/waybar/style.css`:
```css
#custom-vpn.on   { color: #a6e3a1; }
#custom-vpn.off  { color: #585b70; }
#custom-vpn.warn { color: #f9e2af; }
```

Classes:
- `on`: connected with TUN;
- `warn`: connecting, connected without TUN, or an error (the reason is in the tooltip);
- `off`: disconnected, or the core is not running.

## Sway
Open Happ-TUI in a floating window, from the right click above or a key:
```
for_window [app_id="happ-tui"] floating enable, resize set 1000 650, move position center
bindsym $mod+Shift+v exec foot --app-id happ-tui happ-tui
```

Set `HAPP_TUI_ICON_ON` / `HAPP_TUI_ICON_OFF` in the environment of waybar to change the icons.
