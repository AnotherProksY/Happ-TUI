package cmd

import (
	"github.com/AnotherProksY/Happ-TUI/core/db"
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

// BuildStatus turns a connection status into the report of `happ-tui status`.
// It takes the status as a value because it also runs while the proxy
// manager holds its lock to announce a change.
func BuildStatus(database *db.DB, proxy_manager *proxy.ProxyManager, status structs.ProxyStatus) structs.StatusReport {
	report := structs.StatusReport{
		Core:       "running",
		Root:       utils.IsRoot(),
		Service:    utils.IsServiceMode(),
		Connection: status.Connection,
		Tun:        status.IsTunEnabled,
	}
	if status.LastError != "" {
		last_error := status.LastError
		report.LastError = &last_error
	}

	profile := status.Profile
	if profile.Id == 0 {
		// show what a click would connect to
		if latest, err := database.GetLatestConnectedProfile(); err == nil {
			profile = latest
		}
	} else if fresh, err := database.GetProfile(profile.GroupId, profile.Id); err == nil {
		// the stored copy has the latest test result
		profile = fresh
	}

	if profile.Id != 0 {
		report.Profile = &structs.StatusProfile{
			GroupId:  profile.GroupId,
			Id:       profile.Id,
			Name:     profile.Name,
			Protocol: profile.Protocol,
			Address:  profile.Address,
		}
		if profile.TestResult > 0 {
			latency := profile.TestResult
			report.LatencyMs = &latency
		}
		if group, err := database.LoadGroupConfig(profile.GroupId); err == nil {
			report.Profile.Group = group.Name
			info := group.SubscriptionInfo
			if info.Upload != 0 || info.Download != 0 || info.Total != 0 || info.Expire != 0 {
				report.Subscription = &structs.SubscriptionUsage{
					Used:   info.Upload + info.Download,
					Total:  info.Total,
					Expire: info.Expire,
				}
			}
		}
	}

	if status.Connection == "connected" {
		since := status.ConnectedSince
		report.ConnectedSince = &since
		report.Traffic, report.Rate = proxy_manager.Traffic()
	}
	return report
}
