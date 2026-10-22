package cmds

import (
	t "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func ExitPasteProfileView() tea.Msg {
	return t.PasteProfileViewExit{}
}

func EnterPasteProfileView() tea.Msg {
	return t.PasteProfileViewEnter{}
}
