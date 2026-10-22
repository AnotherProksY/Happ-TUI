package mainmodel

import (
	"strings"
	"time"

	list "github.com/AnotherProksY/Happ-TUI/components/List"
	notif_publisher "github.com/AnotherProksY/Happ-TUI/lib/NotifPublisher"
	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"

	tea "github.com/charmbracelet/bubbletea"
)

func applySubscriptionUpdated(msg sharedtypes.SubscriptionUpdated, m Model) (tea.Model, tea.Cmd) {
	tid := findGroupTab(msg.GroupId, m)

	if tid == -1 {
		return m, nil
	}

	items := []list.ListItem{}

	for _, profile := range msg.Profiles {
		child := list.ListItem{
			Name:       profile.Name,
			ProfileId:  profile.Id,
			Protocol:   convertProtocolForDisplay(profile.Protocol),
			TestResult: profile.TestResult,
			Uri:        profile.Uri,
		}
		items = append(items, child)
	}

	m.Tabs.Children[tid].Content.Items = items
	m.Tabs.Children[tid].SubscriptionInfo = msg.SubscriptionInfo
	if msg.Name != "" {
		m.Tabs.Children[tid].Title = msg.Name
	}
	if m.Tabs.Children[tid].Content.Primary != -1 {
		m.Tabs.Children[tid].Content.Primary = 0
	}
	m.Tabs.Children[tid].Content.ResetCursor()
	m.Tabs.Warning = "Subscription updated"
	if announce, _, _ := strings.Cut(msg.SubscriptionInfo.Announce, "\n"); announce != "" {
		m.Tabs.Warning += " · " + announce
	}
	m.Tabs.WarningMode = "success"
	m.Tabs.LastWarningTime = time.Now()
	go func() {
		time.Sleep(time.Second * 4)
		notif_publisher.ClearWarningsNotif(sharedtypes.ClearWarnings{})
	}()
	return m, nil
}
