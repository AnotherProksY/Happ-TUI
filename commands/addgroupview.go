package cmds

import (
	t "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func ExitAddGroupView() tea.Msg {
	return t.AddGroupExit{}
}

func EnterAddGroupView() tea.Msg {
	return t.AddGroupEnter{}
}
