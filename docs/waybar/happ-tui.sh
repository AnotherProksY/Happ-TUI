#!/bin/sh
# Waybar module for Happ-TUI. It reads `happ-tui status --json --watch`, which
# prints a line on every change and every 2 s while connected, and turns each
# line into waybar's JSON (text, class, tooltip). Needs jq.
#
# The stream never ends on its own: when the core stops it prints
# {"core":"stopped"} and waits for it to come back.

ICON_ON=${HAPP_TUI_ICON_ON:-󰖂}
ICON_OFF=${HAPP_TUI_ICON_OFF:-󰖂}

exec happ-tui status --json --watch --interval 2s | jq --unbuffered -c --arg on "$ICON_ON" --arg off "$ICON_OFF" '
  def bytes:
    if . == null then "–"
    elif . >= 1e9 then "\(. / 1e9 * 10 | round / 10) GB"
    elif . >= 1e6 then "\(. / 1e6 * 10 | round / 10) MB"
    elif . >= 1e3 then "\(. / 1e3 | round) KB"
    else "\(.) B" end;
  def duration:
    (now - .) as $s
    | if $s >= 86400 then "\($s / 86400 | floor)d \($s % 86400 / 3600 | floor)h"
      elif $s >= 3600 then "\($s / 3600 | floor)h \($s % 3600 / 60 | floor)m"
      else "\($s / 60 | floor)m" end;
  def row($key; $value): "\($key | . + "         " | .[0:9])\($value)";

  if .core == "stopped" then
    {text: $off, class: "off", tooltip: "Happ-TUI core is not running"}
  else
    (if .connection == "connected" and .tun and .last_error == null then "on"
     elif .connection == "error" or .last_error != null then "warn"
     elif .connection == "connected" or .connection == "connecting" then "warn"
     else "off" end) as $class
    | {
        text: (if $class == "off" then $off else $on end),
        class: $class,
        tooltip: ([
          row("vpn"; .connection + (if .tun then " · TUN" else "" end)),
          (.profile | select(. != null) | row("profile"; "\(.name) (\(.protocol)), \(.group)")),
          (.connected_since | select(. != null) | row("uptime"; duration)),
          (.traffic | select(. != null) | row("traffic"; "↓ \(.down | bytes)  ↑ \(.up | bytes)")),
          (.rate | select(. != null) | row("rate"; "↓ \(.down | bytes)/s  ↑ \(.up | bytes)/s")),
          (.latency_ms | select(. != null) | row("latency"; "\(.) ms")),
          (.subscription | select(. != null) | row("plan"; "\(.used | bytes) / \(if .total > 0 then (.total | bytes) else "∞" end)"
              + (if .expire > 0 then ", expires \(.expire | strftime("%-d %b"))" else "" end))),
          (.last_error | select(. != null) | row("error"; .)),
          "",
          "click: connect/disconnect · right click: open Happ-TUI"
        ] | join("\n"))
      }
  end'
