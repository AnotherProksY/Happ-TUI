package builder

import (
	"encoding/json"
	"testing"

	"github.com/AnotherProksY/Happ-TUI/core/structs"
)

const hostileConfig = `{"remarks":"x",
 "log":{"loglevel":"debug","access":"/etc/cron.d/x","error":"/root/.bashrc","dnsLog":true},
 "api":{"tag":"api","services":["HandlerService"]},
 "stats":{},
 "reverse":{"bridges":[{"tag":"b","domain":"x"}]},
 "metrics":{"tag":"m"},
 "inbounds":[{"tag":"socks","port":10808,"protocol":"socks"}],
 "outbounds":[{"tag":"proxy","protocol":"vless"},{"tag":"direct","protocol":"freedom"}],
 "routing":{"rules":[]},
 "dns":{"servers":["1.1.1.1"]},
 "burstObservatory":{"subjectSelector":["proxy"]}}`

func TestSanitize(t *testing.T) {
	out, err := Sanitize([]byte(hostileConfig))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	json.Unmarshal(out, &config)
	for _, key := range []string{"api", "stats", "reverse", "metrics", "remarks"} {
		if _, ok := config[key]; ok {
			t.Errorf("%s must be removed", key)
		}
	}
	for _, key := range []string{"outbounds", "routing", "dns", "burstObservatory", "inbounds"} {
		if _, ok := config[key]; !ok {
			t.Errorf("%s must be kept", key)
		}
	}
	log := config["log"].(map[string]any)
	if _, ok := log["access"]; ok {
		t.Errorf("access log path must be removed")
	}
	if _, ok := log["error"]; ok {
		t.Errorf("error log path must be removed")
	}
	if log["loglevel"] != "debug" || log["dnsLog"] != true {
		t.Errorf("harmless log settings must be kept: %v", log)
	}
}

func TestProfileConfigSanitizesJSONProfiles(t *testing.T) {
	out, err := ProfileConfig(structs.Profile{Uri: hostileConfig}, 3090, -1, 13345)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	json.Unmarshal(out, &config)
	if _, ok := config["reverse"]; ok {
		t.Errorf("reverse must not reach xray")
	}
	if api, ok := config["api"].(map[string]any); ok {
		for _, service := range api["services"].([]any) {
			if service == "HandlerService" {
				t.Errorf("the profile's HandlerService must not reach xray")
			}
		}
	}
	if log, ok := config["log"].(map[string]any); ok && log["access"] != nil {
		t.Errorf("access log path must not reach xray")
	}
}
