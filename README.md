## Happ-TUI
Happ-TUI is a keyboard-driven Xray client for your terminal that talks to subscriptions the way the [Happ](https://happ.su) app does. It is useful for providers that only serve Happ, for example [Remnawave](https://docs.rw) panels with the HWID device limit enabled.

Happ-TUI is a fork of [Bushuray-tui](https://github.com/Keivan-sf/Bushuray-tui) and [Bushuray-core](https://github.com/Keivan-sf/Bushuray-core) by [@Keivan-sf](https://github.com/Keivan-sf), written in Go with [bubbletea](https://github.com/charmbracelet/bubbletea).

> Happ-TUI is an unofficial project. It is not affiliated with, endorsed by or supported by Happ.

### Install
Download `happ-tui-<version>-linux-x86_64.tar.gz` from the releases (or [build it](#building)), then:
```
tar -xzf happ-tui-<version>-linux-x86_64.tar.gz
cd happ-tui-<version>
sudo ./install.sh
```
This puts `happ-tui` into `/usr/local/bin` and its core, `xray`, `v2parser` and the geo files into `/usr/local/lib/happ-tui`. Set `PREFIX` to install elsewhere, e.g. `sudo PREFIX=/opt/happ-tui ./install.sh`. On Fedora Atomic and other image based systems `/usr/local` is writable, so no layering or reboot is needed.

To remove it: `sudo /usr/local/lib/happ-tui/uninstall.sh`. Your config and groups are kept.

### How to use
```
happ-tui                        # start (the core is started in the background)
sudo /usr/local/bin/happ-tui    # start with TUN mode available
happ-tui die                    # stop the core and close the connection
happ-tui version
```
Press `a` to add a group with your subscription link (the name may stay empty, it comes from the subscription), `enter` to connect, `v` for TUN mode and `?` for all keys.

On some distributions `sudo` doesn't search `/usr/local/bin`, hence the full path above.

### Files
| What | Where |
| --- | --- |
| Config | `~/.config/happ-tui/config.json` |
| Groups and profiles | `~/.local/share/happ-tui/` |
| Logs | `~/.local/state/happ-tui/happ-tui.log` and `core.log` |

On the first start the config and groups of Bushuray (`~/.config/bushuray`, `~/.local/share/bushuray`) are copied over. When started with `sudo`, files in your home directory are still owned by you.

### Configuration
The config file is created with the following template:
```json
{
   "socks-port": 3090,
   "http-port": 3091,
   "core-tcp-port": 4897,
   "test-port-range": {
    "start": 3095,
    "end": 30120
   },
   "test-url": "https://cp.cloudflare.com",
   "no-background": false,
   "auto-connect-on-start": false,
   "remove-failed-profiles": false,
   "subscription-user-agent": "",
   "subscription-client": "happ",
   "hwid": "",
   "subscription-headers": {},
   "dns": {
    "servers": ["1.1.1.1"],
    "query-strategy": "UseIP",
    "mode": "proxy"
   }
}
```
- **socks-port**: Exposed local socks5 port
- **http-port**: Exposed local http port
- **test-port-range**: Port range used for profile testing
- **test-url**: The end-point which the connectivity of profiles will be tested on
- **no-background**: Whether or not tui should have a background. Use this if you want your own terminal background or you have a transparent terminal
- **auto-connect-on-start**: If true, the core connects to the last used profile when it starts
- **remove-failed-profiles**: If true, profiles that fail a new test are deleted (old test results are not affected)
- **subscription-user-agent**: Overrides the user-agent of subscription requests, e.g. `Throne` for a provider that only serves Throne. Leave it empty in `happ` mode
- **subscription-client**: `happ` (default) | `plain`. `happ` makes subscription requests the way Happ does, see [Happ-compatible subscriptions](#happ-compatible-subscriptions). `plain` only sends **subscription-user-agent**
- **hwid**: Device id sent as `X-Hwid` in `happ` mode. Leave empty to derive a stable one from `/etc/machine-id`
- **subscription-headers**: Extra request headers for subscriptions, e.g. `{"X-Device-Model": "ThinkPad"}`. They override the built-in ones, and an empty value removes a header
- **dns**:
     - **servers**: list of dns servers to use. `localhost` will use the default system dns
     - **query-strategy**: `UseIP` | `UseIPv4` | `UseIPv6` | `UseSystem`. This will directly be passed to [xray dns object](https://xtls.github.io/en/config/dns.html#dnsobject). You should mostly use `UseIP` if you're setting a dns server other than localhost
     - **mode**: `proxy` | `system`. If set to `system`, will use default system dns. if set to `proxy`, will use the servers provided in **servers** and tunnel them through the profile

### Happ-compatible subscriptions
Some providers only serve clients that look like Happ and answer everybody else with `404`. In `happ` mode Happ-TUI sends the same request headers as Happ:

| Header | Value |
| --- | --- |
| `User-Agent` | `Happ/4.5.6`, or the one the provider asked for with `Change-User-Agent`, or **subscription-user-agent** if set |
| `X-Hwid` | **hwid**, or `sha256("bushuray:" + /etc/machine-id)` cut to 32 chars |
| `X-Device-Os` | `Linux` |
| `X-Ver-Os` | `VERSION_ID` from `/etc/os-release` |
| `X-Device-Model` | `/sys/class/dmi/id/product_name`, or hostname |
| `X-Device-Locale` | language from `$LANG`, or `en` |

The HWID is the same on every run, so the machine counts as **one device** in the provider's device limit. It is also the same as in Bushuray, so switching doesn't use another device slot. Pin it with **hwid** if you move between machines or containers with different machine ids.

What else is supported, like in Happ:
- Subscriptions with share links (plain or base64, standard or URL-safe) and subscriptions with full Xray JSON configs (one object or an array). A JSON profile keeps the provider's outbounds, routing, balancers and dns; Happ-TUI only replaces the inbounds with its own socks/http (and tproxy for TUN mode) ports. Your **dns** setting is not applied to JSON profiles
- Response headers: `profile-title` (used as the group name if you leave it empty), `subscription-userinfo` (traffic and expiry are shown in the status bar), `profile-update-interval` (the group is updated automatically), `announce`, `support-url`, `profile-web-page-url`, `change-user-agent` and `check-url-via-proxy` (used instead of **test-url** to test that group's profiles)
- The subscription is downloaded right after a group is added. An error response (e.g. `404` or a device limit) doesn't wipe the group's profiles

Encrypted `happ://crypt...` links are not supported.

### TUN mode
TUN mode sends the traffic of all applications to Xray, and the provider's routing decides what goes `direct` and what goes through the VPN. Start Happ-TUI with `sudo`, connect to a profile, then press `v`.

It uses TPROXY rather than a tun interface. The rules live in nftables tables of their own (`ip happ_tui` and `ip6 happ_tui`), so they don't clash with tailscale, firewalld or docker; without the `nft` command Happ-TUI falls back to iptables. What is not sent to Xray: private, CGNAT (tailscale) and local subnets, and tailscale's own packets (fwmark `0x80000`).
- **DNS**: with JSON subscription profiles (and with `dns.mode` other than `system`) DNS queries leaving the machine are answered by Xray, using the provider's dns servers; queries to the local stub (`127.0.0.53`) and to tailscale's MagicDNS are left alone, so systemd-resolved and tailnet names keep working. Queries other than A/AAAA are refused.
- **IPv6**: Xray only gets IPv4 here, so while TUN mode is on all IPv6 traffic leaving the machine is rejected (except loopback, link-local, multicast and `fc00::/7`, which includes tailscale) to avoid leaks. `sudo nft list table ip6 happ_tui` shows how much was blocked.
- ICMP (`ping`) is not sent to Xray.

Running as root raises security concerns, see [this issue](https://github.com/Keivan-sf/Bushuray-core/issues/10). Thanks to [@navidmafi](https://github.com/navidmafi) and [@jpegMushrum](https://github.com/jpegMushrum) for their help with the original TUN mode.

### Building
The back-end lives in [`core/`](core), vendored from Bushuray-core (see `core/UPSTREAM`). For development:
```
go build -o happ-tui .
(cd core && go build -o ../happ-tui-core .)
```
and put `xray`, `geoip.dat`, `geosite.dat` and [`v2parser`](https://github.com/Keivan-sf/v2-uri-parser) into `./bin`.

To build the release archive (needs go, cargo, git, curl, unzip):
```
VERSION=0.3.0 packaging/build-release.sh
```
It downloads a pinned Xray release, checks its SHA-256, builds v2parser at a pinned commit and writes `dist/happ-tui-<version>-linux-x86_64.tar.gz`.

### Debugging
If something is not working as expected, look at the logs:
```
tail -f ~/.local/state/happ-tui/core.log
```
