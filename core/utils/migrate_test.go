package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyUserTreeRefusesSymlinks(t *testing.T) {
	uid := os.Getuid()
	from := t.TempDir()
	os.MkdirAll(filepath.Join(from, "groups", "1"), 0o755)
	os.WriteFile(filepath.Join(from, "config.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(from, "groups", "1", "1.json"), []byte(`{}`), 0o644)

	to := filepath.Join(t.TempDir(), "db")
	if err := copyUserTree(from, to, uid); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(to, "groups", "1", "1.json")); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("copied file missing or not 0600: %v %v", info, err)
	}

	secret := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(secret, []byte("x"), 0o600)
	os.Symlink(secret, filepath.Join(from, "groups", "1", "2.json"))
	err := copyUserTree(from, filepath.Join(t.TempDir(), "db"), uid)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("a symlink must be refused, got %v", err)
	}
}

func TestCopyUserFileChecksOwner(t *testing.T) {
	from := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(from, []byte(`{}`), 0o644)
	if err := copyUserFile(from, filepath.Join(t.TempDir(), "c"), os.Getuid()+1); err == nil {
		t.Errorf("a file of another user must be refused")
	}
}
