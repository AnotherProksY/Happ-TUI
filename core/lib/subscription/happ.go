package subscription

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"syscall"
)

// HappVersion is the Happ desktop release whose user-agent is imitated.
const HappVersion = "4.5.6"

// DeviceInfo holds the device headers Happ sends with every subscription request.
type DeviceInfo struct {
	HWID      string
	OS        string
	OSVersion string
	Model     string
	Locale    string
}

// GetDeviceInfo collects device information for this machine. A non-empty
// hwid overrides the one derived from the machine id.
func GetDeviceInfo(hwid string) DeviceInfo {
	if hwid == "" {
		hwid = machineHWID()
	}
	return DeviceInfo{
		HWID:      hwid,
		OS:        "Linux",
		OSVersion: osVersion(),
		Model:     deviceModel(),
		Locale:    locale(),
	}
}

// machineHWID derives a stable identifier from the systemd machine id so the
// same machine is always counted as one device by the provider.
func machineHWID() string {
	id := ""
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if data, err := os.ReadFile(path); err == nil {
			id = strings.TrimSpace(string(data))
			if id != "" {
				break
			}
		}
	}
	if id == "" {
		id, _ = os.Hostname()
	}
	// the "bushuray:" salt predates the rename and must stay: changing it
	// changes the HWID, and the provider would count a new device
	sum := sha256.Sum256([]byte("bushuray:" + id))
	return hex.EncodeToString(sum[:])[:32]
}

func osVersion() string {
	if file, err := os.Open("/etc/os-release"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if value, ok := strings.CutPrefix(scanner.Text(), "VERSION_ID="); ok {
				if value = strings.Trim(value, `"'`); value != "" {
					return value
				}
			}
		}
	}
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		release := make([]byte, 0, len(uname.Release))
		for _, c := range uname.Release {
			if c == 0 {
				break
			}
			release = append(release, byte(c))
		}
		return string(release)
	}
	return ""
}

func deviceModel() string {
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		if model := string(bytes.TrimSpace(data)); model != "" {
			return model
		}
	}
	hostname, _ := os.Hostname()
	return hostname
}

func locale() string {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := os.Getenv(env)
		if value == "" || value == "C" || value == "POSIX" {
			continue
		}
		lang, _, _ := strings.Cut(value, "_")
		lang, _, _ = strings.Cut(lang, ".")
		if lang != "" {
			return strings.ToLower(lang)
		}
	}
	return "en"
}
