package lib

import (
	"encoding/base64"
	"strings"
	"testing"
)

const xrayJSONSubscription = `[
 {"remarks": "🇩🇪 Germany", "inbounds": [{"tag": "socks", "port": 10808, "protocol": "socks"}],
  "outbounds": [
   {"tag": "proxy", "protocol": "vless", "settings": {"vnext": [{"address": "de.example.com", "port": 443}]},
    "streamSettings": {"security": "reality", "realitySettings": {"serverName": "sni.example.com"}}},
   {"tag": "direct", "protocol": "freedom"}]},
 {"remarks": "Hysteria", "outbounds": [
   {"tag": "direct", "protocol": "freedom"},
   {"tag": "proxy", "protocol": "hysteria", "settings": {"address": "hy.example.com", "port": 443}}]}
]`

func TestXrayJSONSubscription(t *testing.T) {
	for _, body := range []string{xrayJSONSubscription, base64.StdEncoding.EncodeToString([]byte(xrayJSONSubscription))} {
		profiles := GetDBAddProfileDatasFromStr(body, 3)
		if len(profiles) != 2 {
			t.Fatalf("got %d profiles", len(profiles))
		}
		de := profiles[0]
		if de.Name != "🇩🇪 Germany" || de.Protocol != "vless" || de.Address != "de.example.com" || de.Host != "sni.example.com" || de.GroupId != 3 {
			t.Errorf("unexpected profile %+v", de)
		}
		if !strings.HasPrefix(de.Uri, `{"remarks"`) || strings.Contains(de.Uri, "\n") {
			t.Errorf("uri should be compact JSON: %q", de.Uri[:40])
		}
		hy := profiles[1]
		if hy.Protocol != "hysteria" || hy.Address != "hy.example.com" {
			t.Errorf("unexpected profile %+v", hy)
		}
	}
}
