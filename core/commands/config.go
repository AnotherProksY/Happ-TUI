package cmd

import (
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

// GetConfig gives clients the settings they display, so they never read the
// config file themselves.
func (cmd *Cmd) GetConfig() {
	cmd.reply("config", structs.ClientConfig{
		SocksPort:    cmd.AppConfig.SocksPort,
		HttpPort:     cmd.AppConfig.HttpPort,
		NoBackground: cmd.AppConfig.NoBackground,
		Root:         utils.IsRoot(),
		Service:      utils.IsServiceMode(),
	})
}
