package mainmodel

import (
	servercmds "github.com/AnotherProksY/Happ-TUI/lib/ServerCommands"
	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func applyIsRootAnswer(msg sharedtypes.IsRootAnswer, m Model) (tea.Model, tea.Cmd) {
	if !msg.IsRoot {
		m.ActiveSection = "tunview"
		return m, nil
	}
	m.Tabs.TunStatus = "waiting"
	servercmds.EnableTun()
	return m, nil
}
