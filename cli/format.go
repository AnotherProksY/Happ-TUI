package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"
)

// printHuman prints a status report the way a person reads it.
func printHuman(w io.Writer, report sharedtypes.StatusReport) {
	row := func(key string, value string) {
		fmt.Fprintf(w, "%-11s %s\n", key, value)
	}

	core := "running"
	if report.Service {
		core += ", system service"
	}
	if report.Root {
		core += ", root"
	}
	row("core", core)

	connection := report.Connection
	if report.Tun {
		connection += " · TUN"
	}
	row("connection", connection)

	if p := report.Profile; p != nil {
		profile := fmt.Sprintf("%s (%s)", p.Name, p.Protocol)
		if p.Group != "" {
			profile += ", " + p.Group
		}
		row("profile", profile)
	}
	if report.ConnectedSince != nil {
		row("uptime", formatDuration(time.Since(time.Unix(*report.ConnectedSince, 0))))
	}
	if t := report.Traffic; t != nil {
		row("traffic", fmt.Sprintf("↓ %s  ↑ %s", formatBytes(t.Down), formatBytes(t.Up)))
	}
	if r := report.Rate; r != nil {
		row("rate", fmt.Sprintf("↓ %s/s  ↑ %s/s", formatBytes(r.Down), formatBytes(r.Up)))
	}
	if report.LatencyMs != nil {
		row("latency", fmt.Sprintf("%d ms", *report.LatencyMs))
	}
	if s := report.Subscription; s != nil {
		plan := formatBytes(s.Used) + " / "
		if s.Total > 0 {
			plan += formatBytes(s.Total)
		} else {
			plan += "∞"
		}
		if s.Expire > 0 {
			plan += ", expires " + time.Unix(s.Expire, 0).Format("2 Jan 2006")
		}
		row("plan", plan)
	}
	if report.LastError != nil {
		row("last error", *report.LastError)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours >= 24 {
		return fmt.Sprintf("%dd %dh", hours/24, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(bytes)/float64(div)), ".0") + " " + string("KMGTPE"[exp]) + "B"
}
