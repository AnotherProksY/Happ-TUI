package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
)

// XrayUser is the system user that runs xray and v2parser when the core is
// root. It owns nothing and has no home, so a hostile profile config can't
// write files or gain root through them.
const XrayUser = "happ-tui-xray"

const (
	CAP_NET_ADMIN        = 12
	CAP_NET_BIND_SERVICE = 10
)

var xrayUserOnce sync.Mutex

// XrayUserIDs returns the uid and gid of XrayUser, creating the user if it
// doesn't exist yet (install.sh normally does that).
func XrayUserIDs() (uint32, uint32, error) {
	xrayUserOnce.Lock()
	defer xrayUserOnce.Unlock()

	u, err := user.Lookup(XrayUser)
	if err != nil {
		if create_err := createXrayUser(); create_err != nil {
			return 0, 0, fmt.Errorf("user %s is missing and could not be created: %w", XrayUser, create_err)
		}
		if u, err = user.Lookup(XrayUser); err != nil {
			return 0, 0, err
		}
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	if uid == 0 || gid == 0 {
		return 0, 0, fmt.Errorf("user %s must not be root", XrayUser)
	}
	return uint32(uid), uint32(gid), nil
}

func createXrayUser() error {
	nologin := "/usr/sbin/nologin"
	for _, candidate := range []string{"/usr/sbin/nologin", "/sbin/nologin", "/usr/bin/nologin", "/bin/false"} {
		if _, err := os.Stat(candidate); err == nil {
			nologin = candidate
			break
		}
	}
	output, err := exec.Command("useradd", "--system", "--user-group", "--no-create-home",
		"--home-dir", "/", "--shell", nologin, XrayUser).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

// Unprivileged prepares cmd to run as XrayUser when the core is root, with
// only the given ambient capabilities, no supplementary groups, "/" as the
// working directory and a minimal environment. When the core isn't root the
// command runs as the core's own user.
func Unprivileged(cmd *exec.Cmd, caps ...uintptr) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.Dir = "/"
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	if dir := filepath.Dir(cmd.Path); filepath.IsAbs(dir) {
		// xray looks for geoip.dat and geosite.dat here
		cmd.Env = append(cmd.Env, "XRAY_LOCATION_ASSET="+dir)
	}
	if !IsRoot() {
		return nil
	}
	uid, gid, err := XrayUserIDs()
	if err != nil {
		return err
	}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, Groups: []uint32{}}
	cmd.SysProcAttr.AmbientCaps = caps
	return nil
}
