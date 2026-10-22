package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// SystemSocketPath is where a root core (system service or started with
// sudo) listens. /run is root-owned, so other users can't plant a socket there.
const SystemSocketPath = "/run/happ-tui/core.sock"

// serviceUser is the user a system service runs for, set by SetServiceUser.
var serviceUser *user.User

// SetServiceUser switches to service mode: the core runs as root for name,
// keeps its data outside of that user's home and lets them control it.
func SetServiceUser(name string) error {
	u, err := user.Lookup(name)
	if err != nil {
		return fmt.Errorf("unknown user %q: %w", name, err)
	}
	serviceUser = u
	return nil
}

func IsServiceMode() bool {
	return serviceUser != nil
}

// UserSocketPath is where a core running as uid listens.
func UserSocketPath(uid int) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" || os.Getuid() != uid {
		dir = filepath.Join("/run/user", strconv.Itoa(uid))
	}
	return filepath.Join(dir, AppName, "core.sock")
}

// SocketPath is where this core listens.
func SocketPath() string {
	if IsRoot() {
		return SystemSocketPath
	}
	return UserSocketPath(os.Getuid())
}

// AllowedUID is the user besides root that may control this core.
func AllowedUID() int {
	if serviceUser != nil {
		uid, _ := strconv.Atoi(serviceUser.Uid)
		return uid
	}
	if IsRoot() {
		if uid, err := strconv.Atoi(os.Getenv("SUDO_UID")); err == nil {
			return uid
		}
	}
	return os.Getuid()
}

// ListenSocket creates the control socket. Access is checked per connection
// with SO_PEERCRED, so the socket itself is world-connectable; its directory
// must belong to us and must not be a symlink.
func ListenSocket(path string) (*net.UnixListener, error) {
	dir := filepath.Dir(path)
	mode := os.FileMode(0o700)
	if IsRoot() {
		mode = 0o755
	}
	if err := os.MkdirAll(dir, mode); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !ok || int(stat.Uid) != os.Getuid() || info.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("%s must be a directory owned by uid %d and not writable by others", dir, os.Getuid())
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&fs.ModeSocket == 0 {
			return nil, fmt.Errorf("%s exists and is not a socket", path)
		}
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			conn.Close()
			return nil, fmt.Errorf("another core is already listening on %s", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o666); err != nil {
		listener.Close()
		return nil, err
	}
	return listener, nil
}

// PeerUID returns the uid of the process on the other end of conn.
func PeerUID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return -1, err
	}
	var cred *syscall.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return -1, err
	}
	if credErr != nil {
		return -1, credErr
	}
	return int(cred.Uid), nil
}
