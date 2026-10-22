package utils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

const maxMigratedFileSize = 16 << 20

// MigrateUserData copies name's config and groups into the service
// directories, once: each target is only written while it doesn't exist.
// The source files stay, for when the service is turned off again.
//
// It runs as root on files in a home directory the user controls, so it
// only copies regular files that belong to that user (so nothing the user
// couldn't read already ends up served back to them), opens them without
// following symlinks and checks they are still the files it looked at.
func MigrateUserData(name string) error {
	u, err := user.Lookup(name)
	if err != nil {
		return fmt.Errorf("unknown user %q: %w", name, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}

	var errs []error
	config_target := filepath.Join(ServiceConfigDir, "config.json")
	if _, err := os.Lstat(config_target); errors.Is(err, fs.ErrNotExist) {
		for _, app := range []string{AppName, legacyAppName} {
			source := filepath.Join(u.HomeDir, ".config", app, "config.json")
			if _, err := os.Lstat(source); err != nil {
				continue
			}
			if err := os.MkdirAll(ServiceConfigDir, 0o700); err != nil {
				return err
			}
			if err := copyUserFile(source, config_target, uid); err != nil {
				errs = append(errs, err)
			} else {
				fmt.Println("copied", source, "to", config_target)
			}
			break
		}
	}

	data_target := filepath.Join(ServiceDataDir, "db")
	if _, err := os.Lstat(data_target); errors.Is(err, fs.ErrNotExist) {
		for _, app := range []string{AppName, legacyAppName} {
			source := filepath.Join(u.HomeDir, ".local", "share", app, "db")
			if info, err := os.Lstat(source); err != nil || !info.IsDir() {
				continue
			}
			if err := os.MkdirAll(ServiceDataDir, 0o700); err != nil {
				return err
			}
			if err := copyUserTree(source, data_target, uid); err != nil {
				errs = append(errs, err)
			} else {
				fmt.Println("copied", source, "to", data_target)
			}
			break
		}
	}
	return errors.Join(errs...)
}

func copyUserTree(from string, to string, uid int) error {
	return filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink, refusing to copy it", path)
		case entry.IsDir():
			return os.Mkdir(target, 0o700)
		default:
			return copyUserFile(path, target, uid)
		}
	})
}

func copyUserFile(from string, to string, uid int) error {
	src, err := os.OpenFile(from, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", from, err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || int(stat.Uid) != uid {
		return fmt.Errorf("%s is not a regular file owned by uid %d, refusing to copy it", from, uid)
	}
	if info.Size() > maxMigratedFileSize {
		return fmt.Errorf("%s is too large", from)
	}
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, io.LimitReader(src, maxMigratedFileSize)); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
