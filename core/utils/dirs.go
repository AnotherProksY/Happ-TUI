package utils

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const AppName = "happ-tui"

// legacyAppName is the name the config and data directories had before the
// project was renamed from bushuray.
const legacyAppName = "bushuray"

// Service mode keeps everything outside of the user's home: a root process
// writing there could be tricked through symlinks into touching system files.
const (
	ServiceConfigDir = "/etc/happ-tui"
	ServiceDataDir   = "/var/lib/happ-tui"
)

func ConfigDir() (string, error) {
	if IsServiceMode() {
		return ServiceConfigDir, nil
	}
	return homeSubdir(".config", AppName)
}

func DataDir() (string, error) {
	if IsServiceMode() {
		return ServiceDataDir, nil
	}
	return homeSubdir(".local", "share", AppName)
}

func StateDir() (string, error) {
	return homeSubdir(".local", "state", AppName)
}

func homeSubdir(parts ...string) (string, error) {
	home, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, parts...)...), nil
}

// MigrateLegacyDirs copies the bushuray config and data directories to the
// happ-tui ones, once: a directory is only copied while its new location
// does not exist yet. The old directories are left untouched.
func MigrateLegacyDirs() error {
	home, err := GetHomeDir()
	if err != nil {
		return err
	}
	var errs []error
	for _, parts := range [][]string{{".config"}, {".local", "share"}} {
		base := filepath.Join(append([]string{home}, parts...)...)
		from := filepath.Join(base, legacyAppName)
		to := filepath.Join(base, AppName)
		if _, err := os.Stat(to); err == nil || !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if info, err := os.Stat(from); err != nil || !info.IsDir() {
			continue
		}
		if err := copyTree(from, to); err != nil {
			errs = append(errs, err)
			continue
		}
		log.Println("migrated", from, "to", to)
	}
	return errors.Join(errs...)
}

func copyTree(from string, to string) error {
	return filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entry.IsDir():
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if err := copyFile(path, target, info.Mode().Perm()); err != nil {
				return err
			}
		default:
			return nil
		}
		ChownToInvokingUser(target)
		return nil
	})
}

func copyFile(from string, to string, perm fs.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

// EnsureDir creates dir and its missing parents, owned by the invoking user
// when running under sudo.
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		if err := EnsureDir(parent); err != nil {
			return err
		}
	}
	mode := os.FileMode(0o755)
	if IsServiceMode() {
		// subscription links are secrets
		mode = 0o700
	}
	if err := os.Mkdir(dir, mode); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	ChownToInvokingUser(dir)
	return nil
}

// LogFile returns the path of a log file in the state directory, creating it
// so that a run under sudo doesn't leave a root-owned file behind.
func LogFile(name string) string {
	dir, err := StateDir()
	if err != nil || EnsureDir(dir) != nil {
		return name
	}
	path := filepath.Join(dir, name)
	if file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		file.Close()
		ChownToInvokingUser(path)
	}
	return path
}

// ChownToInvokingUser hands a path created under sudo back to the user who
// ran sudo. It does nothing otherwise.
func ChownToInvokingUser(path string) {
	if os.Getuid() != 0 {
		return
	}
	uid, err := strconv.Atoi(os.Getenv("SUDO_UID"))
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err != nil {
		gid = uid
	}
	os.Lchown(path, uid, gid)
}
