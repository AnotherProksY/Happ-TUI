package cmds

import (
	servercmds "github.com/AnotherProksY/Happ-TUI/lib/ServerCommands"
	t "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func EnterTunView() tea.Msg {
	servercmds.IsRoot()
	return nil
}

func KillCore() tea.Msg {
	return t.KillCore{}
}

func ExitTunView() tea.Msg {
	return t.TunViewExit{}
}
