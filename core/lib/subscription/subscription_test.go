package subscription

import (
	"net/http"
	"testing"

	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
)

func TestNewRequestHapp(t *testing.T) {
	cfg := config.AppConfig{
		SubscriptionClient:  config.SubscriptionClientHapp,
		HWID:                "test-hwid-123456",
		SubscriptionHeaders: map[string]string{"X-Device-Model": "", "X-Extra": "1"},
	}
	req, err := NewRequest("https://example.com/sub", cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("User-Agent"); got != "Happ/"+HappVersion {
		t.Errorf("User-Agent = %q", got)
	}
	if got := req.Header.Get("X-Hwid"); got != "test-hwid-123456" {
		t.Errorf("X-Hwid = %q", got)
	}
	if got := req.Header.Get("X-Device-Os"); got != "Linux" {
		t.Errorf("X-Device-Os = %q", got)
	}
	if _, ok := req.Header["X-Device-Model"]; ok {
		t.Errorf("empty override should remove X-Device-Model")
	}
	if got := req.Header.Get("X-Extra"); got != "1" {
		t.Errorf("X-Extra = %q", got)
	}

	req, _ = NewRequest("https://example.com/sub", cfg, "Happ/4.12.0/ios/1")
	if got := req.Header.Get("User-Agent"); got != "Happ/4.12.0/ios/1" {
		t.Errorf("provider user-agent not used: %q", got)
	}
	cfg.SubscriptionUserAgent = "Custom"
	req, _ = NewRequest("https://example.com/sub", cfg, "Happ/4.12.0/ios/1")
	if got := req.Header.Get("User-Agent"); got != "Custom" {
		t.Errorf("configured user-agent should win: %q", got)
	}
}

func TestNewRequestPlain(t *testing.T) {
	cfg := config.AppConfig{SubscriptionClient: config.SubscriptionClientPlain}
	req, _ := NewRequest("https://example.com/sub", cfg, "")
	if req.Header.Get("X-Hwid") != "" {
		t.Errorf("plain mode must not send X-Hwid")
	}
	if v, ok := req.Header["User-Agent"]; !ok || v != nil {
		t.Errorf("plain mode without user-agent should suppress it, got %v", v)
	}
}

func TestMachineHWIDFormat(t *testing.T) {
	hwid := machineHWID()
	if len(hwid) != 32 {
		t.Fatalf("hwid length = %d", len(hwid))
	}
	if hwid != machineHWID() {
		t.Fatalf("hwid is not stable")
	}
}

func TestParseInfo(t *testing.T) {
	header := http.Header{}
	header.Set("Profile-Title", "base64:T2ZmaWNpYWwgVlBO")
	header.Set("Subscription-Userinfo", "upload=10; download=81119134298; total=0; expire=1700000000")
	header.Set("Profile-Update-Interval", "1")
	header.Set("Support-Url", "http://t.me/help")
	header.Set("Announce", "plain text")
	header.Set("Change-User-Agent", "Happ/4.12.0/ios/1")
	header.Set("Check-Url-Via-Proxy", "http://example.com/generate_204")

	info := ParseInfo(header)
	if info.Title != "Official VPN" {
		t.Errorf("Title = %q", info.Title)
	}
	if info.Upload != 10 || info.Download != 81119134298 || info.Total != 0 || info.Expire != 1700000000 {
		t.Errorf("userinfo = %+v", info)
	}
	if info.UpdateInterval != 3600 {
		t.Errorf("UpdateInterval = %d", info.UpdateInterval)
	}
	if info.SupportUrl != "http://t.me/help" || info.Announce != "plain text" {
		t.Errorf("texts = %+v", info)
	}
	if info.CheckUrl != "http://example.com/generate_204" {
		t.Errorf("CheckUrl = %q", info.CheckUrl)
	}
	if info.ChangeUserAgent != "Happ/4.12.0/ios/1" {
		t.Errorf("ChangeUserAgent = %q", info.ChangeUserAgent)
	}
}

func TestDecodeBase64Variants(t *testing.T) {
	for _, in := range []string{"aGk/Pz4+", "aGk_Pz4-", "aGk/Pz4", "aGk/\nPz4+"} {
		got, ok := DecodeBase64(in)
		if !ok || got != "hi??>>" && got != "hi??>" {
			t.Errorf("DecodeBase64(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := DecodeBase64("vless://not-base64"); ok {
		t.Errorf("share link must not decode")
	}
}
