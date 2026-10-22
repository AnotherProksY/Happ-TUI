package TCPServer

import (
	cmd "github.com/AnotherProksY/Happ-TUI/core/commands"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"log"
	"time"
)

// autoUpdateSubscriptions refreshes groups whose provider sent a
// profile-update-interval header, like Happ does.
func (s *Server) autoUpdateSubscriptions() {
	command_handler := cmd.Cmd{DB: s.DB, BroadCast: s.BroadCast, Reply: func([]byte) error { return nil }, AppConfig: s.appConfig}
	// failed updates don't move LastUpdated, so remember attempts to avoid
	// retrying every minute
	last_attempt := map[int]int64{}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		groups, err := s.DB.GetAllGroupsAndProfiles()
		if err != nil {
			log.Println("auto update: failed to load groups:", err)
		}
		now := time.Now().Unix()
		for _, group_with_profiles := range groups {
			group := group_with_profiles.Group
			interval := int64(group.SubscriptionInfo.UpdateInterval)
			if group.SubscriptionUrl == "" || interval <= 0 || now-max(group.LastUpdated, last_attempt[group.Id]) < interval {
				continue
			}
			last_attempt[group.Id] = now
			log.Println("auto updating subscription of group", group.Id)
			command_handler.UpdateSubscription(structs.UpdateSubscriptionData{GroupId: group.Id}, s.proxy_manager)
		}
		<-ticker.C
	}
}
