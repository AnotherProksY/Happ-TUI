package tabs

import (
	"fmt"
	"time"

	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderStatusBar() string {
	bg_width := m.Width
	connection_status := ""
	tun_status := ""

	if m.IsConnected {
		bg_width -= 11
		connection_status = lipgloss.NewStyle().Background(lipgloss.Color("#8839ef")).Foreground(lipgloss.Color("#ffffff")).Width(11).Align(lipgloss.Center).Render("CONNECTED")
	} else {
		bg_width -= 15
		connection_status = lipgloss.NewStyle().Background(lipgloss.Color("#1e1e2e")).Foreground(lipgloss.Color("#ffffff")).Width(15).Align(lipgloss.Center).Render("NOT CONNECTED")
	}

	if m.TunStatus == "connected" {
		bg_width -= 8
		tun_status = lipgloss.NewStyle().Background(lipgloss.Color("#8839ef")).Foreground(lipgloss.Color("#ffffff")).Width(8).Align(lipgloss.Center).Render("TUN ON")
	} else if m.TunStatus == "waiting" {
		bg_width -= 9
		tun_status = lipgloss.NewStyle().Background(lipgloss.Color("#1e1e2e")).Foreground(lipgloss.Color("#ffffff")).Width(9).Align(lipgloss.Center).Render("WAITING")
	} else {
		bg_width -= 9
		tun_status = lipgloss.NewStyle().Background(lipgloss.Color("#1e1e2e")).Foreground(lipgloss.Color("#ffffff")).Width(9).Align(lipgloss.Center).Render("TUN OFF")
	}

	status_text := fmt.Sprintf(" SOCKS5 *:%d | HTTP *:%d", m.SocksPort, m.HttpPort)
	if len(m.Children) > 0 {
		status_text += renderSubscriptionInfo(m.Children[m.ActiveTap].SubscriptionInfo)
	}
	bg := lipgloss.NewStyle().Width(bg_width).MaxWidth(bg_width).Inline(true).Background(lipgloss.Color("#11111b")).Foreground(lipgloss.Color("#585b70")).Render(status_text)

	return lipgloss.NewStyle().MarginTop(0).Render(lipgloss.JoinHorizontal(lipgloss.Top, connection_status, bg, tun_status))
}

// renderSubscriptionInfo shows traffic and expiry the provider reported for the group.
func renderSubscriptionInfo(info sharedtypes.SubscriptionInfo) string {
	used := info.Upload + info.Download
	if used == 0 && info.Total == 0 && info.Expire == 0 {
		return ""
	}
	total := "∞"
	if info.Total > 0 {
		total = formatBytes(info.Total)
	}
	text := fmt.Sprintf(" | %s / %s", formatBytes(used), total)
	if info.Expire > 0 {
		text += " | until " + time.Unix(info.Expire, 0).Format("2006-01-02")
	}
	return text
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
