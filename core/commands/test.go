package cmd

import (
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"log"
)

func (cmd *Cmd) TestProfile(data structs.TestProfileData, proxy_manager *proxy.ProxyManager) {
	profile, err := cmd.DB.GetProfile(data.Profile.GroupId, data.Profile.Id)
	if err != nil {
		log.Println(err.Error())
		cmd.warn("test-failed", err.Error())
		return
	}
	// use the check url the provider asked for, like Happ does
	test_url := ""
	if group, err := cmd.DB.LoadGroupConfig(data.Profile.GroupId); err == nil {
		test_url = group.SubscriptionInfo.CheckUrl
	}
	proxy_manager.TestProfile(profile, test_url)
}

func (cmd *Cmd) StopTests(proxy_manager *proxy.ProxyManager) {
	proxy_manager.StopTests()
}
