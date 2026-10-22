package cmds

import (
	t "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func EnterHelpView() tea.Msg {
	return t.HelpViewEnter{}
}

func ExitHelpView() tea.Msg {
	return t.HelpViewExit{}
}
