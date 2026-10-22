package cmd

import (
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/lib/subscription"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"log"
	"sync"
)

var subscription_mutex = sync.Mutex{}

func (cmd *Cmd) UpdateSubscription(data structs.UpdateSubscriptionData, proxy_manager *proxy.ProxyManager) {
	if subscription_mutex.TryLock() == false {
		cmd.warn("command-ignored", "update-subscription")
		return
	}
	defer subscription_mutex.Unlock()

	group, err := cmd.DB.LoadGroupConfig(data.GroupId)
	if err != nil {
		cmd.warn("update-subscription-failed", "Failed to load group config for updating subscription")
		return
	}

	result, err := subscription.Fetch(group.SubscriptionUrl, cmd.AppConfig, group.SubscriptionInfo.ChangeUserAgent)
	if err != nil {
		log.Println("failed to get subscription content:", err)
		cmd.warn("update-subscription-failed", "Failed to get subscription content: "+err.Error())
		return
	}

	db_profiles := lib.GetDBAddProfileDatasFromStr(result.Body, data.GroupId)
	if len(db_profiles) == 0 {
		cmd.warn("update-subscription-failed", "Subscription has no supported profiles")
		return
	}

	status := proxy_manager.GetStatus()
	keep_profile_id := 0
	if status.Connection == "connected" && status.Profile.GroupId == data.GroupId {
		keep_profile_id = status.Profile.Id
	}

	profiles, group, err := cmd.DB.UpdateGroupAndProfiles(data.GroupId, db_profiles, keep_profile_id, result.Info)
	if err != nil {
		log.Println(err)
		cmd.warn("update-subscription-failed", "Failed to add new subscription content to database")
		return
	}
	cmd.send("subscription-updated", structs.SubscriptionUpdated{
		GroupId:          data.GroupId,
		Name:             group.Name,
		Profiles:         profiles,
		SubscriptionInfo: group.SubscriptionInfo,
	})
}
