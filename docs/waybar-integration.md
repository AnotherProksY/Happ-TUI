# Waybar integration: requirements

Goal: a small VPN widget for waybar (Sway). It needs:

- **two states**: on (active colour) / off (dim);
- **click** connects to the last used profile with TUN mode on, or disconnects;
- **tooltip** with status and metrics, plus a hint for a shortcut that opens the full Happ-TUI in a floating terminal.

The widget is a shell script polled every 2–5 s and run on click. It has no TTY and can't enter a sudo password.

## What is there today (v0.3.0)

| Area | Today | Why it blocks the widget |
| --- | --- | --- |
| CLI | `happ-tui version`, `happ-tui die`. Anything else starts the TUI and needs `/dev/tty` | No way to query state or connect/disconnect from a script |
| Control channel | TCP `127.0.0.1:4897`, JSON `{"msg","data"}` with a 4-byte big-endian length prefix (`lib/Connection/connection.go`). Replies are asynchronous notifications, not request/response | Scriptable only with a custom client that waits for the right notification |
| Last profile | `~/.local/share/happ-tui/db/config.json` → `latest_connected_profile`, used only by `auto-connect-on-start` (`core/db/profiles.go`, `GetLatestConnectedProfile`) | No command connects to it |
| TUN | Works only when the core itself runs as root. The core is spawned by the TUI (`utils/spawn_core.go`), so it is root only if the TUI was started with `sudo`. The user's home is found via `SUDO_UID` (`core/utils/paths.go`) | A click can't start a root core |
| Security | Port 4897 has no authentication | With a root core, any local process can drive it |
| Metrics | Xray runs without StatsService/API. Only the profile `test-result` and `subscription_info` exist | No session traffic, rate or uptime |

## R1. Non-interactive CLI

No TUI and no TTY. Exit code 0 on success, non-zero on error. Errors go to stderr, and into an `error` field with `--json`.

| Command | Behaviour |
| --- | --- |
| `happ-tui status [--json]` | Print the state (see R4). If no core is running: `{"core":"stopped"}`, exit 0 |
| `happ-tui up [--tun] [--profile GROUP:ID] [--timeout 15s]` | Connect. The default profile is `latest_connected_profile`. `--tun` enables TUN once connected. Waits until connected or failed, up to the timeout. If no core is running, use the system service (R2) or spawn one |
| `happ-tui down` | Disable TUN and disconnect. The core keeps running |
| `happ-tui toggle [--tun]` | `up` or `down` depending on the current state |

Errors to report with distinct messages:
- no last profile;
- `--tun` while the core is not root;
- timeout;
- xray failed to start;
- subscription/profile missing.

## R2. Root core without sudo at click time

- A systemd **system** unit `happ-tui-core.service` runs the core as root for one user. `install.sh` installs it, disabled by default.
- The user is given explicitly (`--user NAME` or `HAPP_TUI_USER=`), not through `SUDO_UID`. Files in that user's home stay owned by the user, as now.
- `happ-tui` (TUI and CLI) started as the **normal user** attaches to that root core, and TUN mode is available without `sudo`.
- Without the service, behaviour stays as today: the core is spawned as the current user.

## R3. Protected control channel

Required together with R2.

- A unix socket, e.g. `/run/happ-tui/core.sock`, mode `0660`, owned by `root:happ-tui`. `install.sh --service` creates the group and adds the user to it.
- Or check `SO_PEERCRED` and accept only root and the configured user.
- When the core runs as root, TCP 4897 is off, or on only behind an explicit flag.

## R4. Metrics in `status --json`

Enable Xray's StatsService and API on a local port or socket:
- `api`;
- `stats`;
- `policy.levels.0.statsUserUplink/Downlink`;
- `policy.system.statsOutboundUplink/Downlink`.

Expose the results through `status --json`:

```json
{
  "core": "running",
  "root": true,
  "connection": "connected",
  "tun": true,
  "profile": {
    "group_id": 1, "id": 12,
    "name": "🇩🇪 Germany", "group": "Provider",
    "protocol": "vless", "address": "de1.example.com"
  },
  "connected_since": 1792630000,
  "traffic": {"up": 12345678, "down": 987654321},
  "rate": {"up": 1200, "down": 54000},
  "latency_ms": 84,
  "subscription": {"used": 123456789, "total": 107374182400, "expire": 1795000000},
  "last_error": null
}
```

Fields:
- `connection`: `connected` | `connecting` | `disconnected` | `error`.
- `connected_since`: unix time.
- `traffic`: bytes through the proxy outbound(s) since connecting.
- `rate`: bytes/s, averaged over 2–5 s.
- `latency_ms`: the last `test-result` of the profile, `null` if never tested.
- `subscription`: from `subscription_info` (`upload + download`, `total`, `expire`).
- `traffic` and `rate` are `null` when disconnected. `profile` is then the last used profile, so the widget can show what a click will connect to.
- All numbers are raw (bytes, unix seconds). The widget formats them.
- The call must be cheap: it is polled every 2–5 s.

## R5. Small things

- `happ-tui` (TUI) attaching to an already running root core prints nothing to stdout. Today it prints `core was not found…` and `connection established`.
- `install.sh --service` installs the unit, creates the `happ-tui` group and adds `$SUDO_USER` to it (R2/R3). `uninstall.sh` removes them.

## How the widget will use it

| Action | Command |
| --- | --- |
| Poll | `happ-tui status --json`; class `on` / `off` / `warn` (`connection: error` or `last_error`) |
| Click | `happ-tui toggle --tun`, then a signal to waybar to refresh |
| Shortcut from the tooltip | `foot --app-id happ-tui happ-tui` in a floating window, like btop for the system monitor |

Tooltip:
```
vpn      connected · TUN
profile  🇩🇪 Germany (vless), Provider
uptime   1h 12m
traffic  ↓ 1.2 GB  ↑ 34 MB
rate     ↓ 54 KB/s  ↑ 1.2 KB/s
latency  84 ms
plan     1.1 GB / 100 GB, expires 12 Dec
```

## Implementation (v0.4)

Done as described above, except for these changes. They were agreed after review because the original wording was unsafe or wouldn't work.

| Spec | Implemented | Why |
| --- | --- | --- |
| R2: the root service keeps files in the user's home | `/etc/happ-tui` (0600) and `/var/lib/happ-tui` (0700); `ProtectHome=yes`; `install.sh --service` copies the home data once; TUI/CLI use only the socket | A root process writing and deleting in a directory the user controls can be pointed at system files through symlinks |
| — | Xray and v2parser run as `happ-tui-xray`; in TUN mode only with `CAP_NET_ADMIN` + `CAP_NET_BIND_SERVICE`; profile configs are filtered (no `api`/`stats`/`reverse`, no log file paths) | Xray used to run as an alias of uid 0, so a JSON profile could write files as root |
| R3: group `happ-tui`, socket `0660` | `SO_PEERCRED`: only root and the configured user; no group | Group membership needs a re-login; the uid check doesn't |
| R3: TCP 4897 behind a flag | TCP removed in both modes; `$XDG_RUNTIME_DIR/happ-tui/core.sock` for a user core | Any local user could drive a root core over TCP |
| R1: "use the system service or spawn one" | The service is enabled and always running; without it the CLI spawns a user core, and `--tun` fails with `not-root` | A CLI can't start a system service without polkit or sudo |
| R4: `statsUserUplink/Downlink` | Only `statsOutbound*`; the API has StatsService only | User counters need users with `email` on inbounds, which a client doesn't have; HandlerService could add an open proxy |
| Widget polls `status --json` | `status --json --watch` | No polling and refresh signals; it updates at once and survives core restarts |
| `toggle --tun` | Disconnected → up with TUN; connected without TUN → enable TUN; connected with TUN → down. A TUN failure keeps the proxy connection and sets `last_error` | Agreed rules |

Additions:
- `happ-tui logs [-n] [-f]`, from the core's memory (journald needs sudo or the `systemd-journal` group);
- `ExecStopPost=happ-tui-core --cleanup` removes TUN rules after a stop or crash;
- `uninstall.sh --purge`.

The widget itself: [`docs/waybar`](waybar).
