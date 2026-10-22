package cmd

import (
	"errors"
	"log"

	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
)

// errorKey maps an error to the stable key scripts get in an error reply.
func errorKey(err error) string {
	switch {
	case errors.Is(err, errNoLastProfile):
		return "no-last-profile"
	case errors.Is(err, errProfileNotFound):
		return "profile-not-found"
	case errors.Is(err, proxy.ErrNotRoot):
		return "not-root"
	case errors.Is(err, proxy.ErrTun):
		return "tun-failed"
	case errors.Is(err, proxy.ErrXray):
		return "xray-failed"
	case errors.Is(err, proxy.ErrProfile):
		return "profile-invalid"
	case errors.Is(err, proxy.ErrNotConnected):
		return "not-connected"
	}
	return "error"
}

var (
	errNoLastProfile   = errors.New("no profile was used yet; connect once from happ-tui or pass --profile GROUP:ID")
	errProfileNotFound = errors.New("profile not found")
)

func (cmd *Cmd) resolveProfile(id *structs.ProfileID) (structs.Profile, error) {
	if id == nil {
		latest, err := cmd.DB.GetLatestConnectedProfileID()
		if err != nil {
			return structs.Profile{}, err
		}
		if latest.Id == 0 {
			return structs.Profile{}, errNoLastProfile
		}
		id = &latest
	}
	profile, err := cmd.DB.GetProfile(id.GroupId, id.Id)
	if err != nil {
		return profile, errProfileNotFound
	}
	return profile, nil
}

// up connects to profile, with TUN when tun is set. If TUN can't be enabled
// the proxy connection is kept and the TUN error is returned.
func (cmd *Cmd) up(profile structs.Profile, tun bool, proxy_manager *proxy.ProxyManager) error {
	status := proxy_manager.GetStatus()
	same_profile := status.Profile.GroupId == profile.GroupId && status.Profile.Id == profile.Id
	if status.Connection == "connected" && same_profile && status.IsTunEnabled == tun {
		return nil
	}

	err := proxy_manager.Connect(profile, tun)
	if err != nil && tun && (errors.Is(err, proxy.ErrTun) || errors.Is(err, proxy.ErrNotRoot)) {
		already_up := status.Connection == "connected" && same_profile && !status.IsTunEnabled
		if already_up || proxy_manager.Connect(profile, false) == nil {
			proxy_manager.SetLastError(err.Error())
			cmd.rememberProfile(profile)
		}
		return err
	}
	if err == nil {
		cmd.rememberProfile(profile)
	}
	return err
}

func (cmd *Cmd) rememberProfile(profile structs.Profile) {
	if err := cmd.DB.UpdateLatestConnectedProfile(profile.GroupId, profile.Id); err != nil {
		log.Println(err.Error())
	}
}

func (cmd *Cmd) replyStatusOrError(err error, proxy_manager *proxy.ProxyManager) {
	if err != nil {
		cmd.ReplyError(errorKey(err), err.Error())
		return
	}
	cmd.reply("status", BuildStatus(cmd.DB, proxy_manager, proxy_manager.GetStatus()))
}

func (cmd *Cmd) Up(data structs.UpData, proxy_manager *proxy.ProxyManager) {
	ConnectionMutex.Lock()
	defer ConnectionMutex.Unlock()

	profile, err := cmd.resolveProfile(data.Profile)
	if err == nil {
		err = cmd.up(profile, data.Tun, proxy_manager)
	}
	cmd.replyStatusOrError(err, proxy_manager)
}

func (cmd *Cmd) Down(proxy_manager *proxy.ProxyManager) {
	ConnectionMutex.Lock()
	defer ConnectionMutex.Unlock()

	var err error
	if stopErr := proxy_manager.Stop(); stopErr != nil {
		err = errors.Join(proxy.ErrTun, stopErr)
	}
	cmd.replyStatusOrError(err, proxy_manager)
}

// Toggle switches the connection for a click on a widget:
//   - with tun: disconnected → connect with TUN; connected without TUN →
//     enable TUN; connected with TUN → disconnect;
//   - without tun: connected in any way → disconnect, otherwise connect.
func (cmd *Cmd) Toggle(data structs.ToggleData, proxy_manager *proxy.ProxyManager) {
	ConnectionMutex.Lock()
	defer ConnectionMutex.Unlock()

	status := proxy_manager.GetStatus()
	connected := status.Connection == "connected" || status.Connection == "connecting"

	var err error
	switch {
	case connected && (!data.Tun || status.IsTunEnabled):
		if stopErr := proxy_manager.Stop(); stopErr != nil {
			err = errors.Join(proxy.ErrTun, stopErr)
		}
	case connected:
		err = cmd.up(status.Profile, true, proxy_manager)
	default:
		var profile structs.Profile
		profile, err = cmd.resolveProfile(data.Profile)
		if err == nil {
			err = cmd.up(profile, data.Tun, proxy_manager)
		}
	}
	cmd.replyStatusOrError(err, proxy_manager)
}

func (cmd *Cmd) GetStatus(proxy_manager *proxy.ProxyManager) {
	cmd.reply("status", BuildStatus(cmd.DB, proxy_manager, proxy_manager.GetStatus()))
}
