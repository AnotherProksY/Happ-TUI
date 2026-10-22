package global

import (
	appconfig "github.com/AnotherProksY/Happ-TUI/lib/AppConfig"

	"github.com/charmbracelet/lipgloss"
)

var NoColor = lipgloss.NoColor{}
var BgColor = lipgloss.Color("#020308")

func GetBgColor() lipgloss.TerminalColor {
	if appconfig.GetConfig().NoBackground {
		return NoColor
	} else {
		return BgColor
	}
}

// Version is set at build time with -ldflags "-X github.com/AnotherProksY/Happ-TUI/global.Version=..."
var Version = "v0.3.0-dev"
