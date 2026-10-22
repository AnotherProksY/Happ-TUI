package builder

import (
	"encoding/json"
	"testing"
)

const providerConfig = `{"remarks":"x","meta":{"a":1},
 "inbounds":[{"tag":"socks-provider","port":10808,"listen":"127.0.0.1","protocol":"socks"},
             {"tag":"http-provider","port":10809,"protocol":"http"}],
 "outbounds":[{"tag":"proxy","protocol":"vless"},{"tag":"direct","protocol":"freedom"}],
 "routing":{"rules":[{"type":"field","inboundTag":["socks-provider"],"outboundTag":"proxy"}]},
 "dns":{"servers":["1.1.1.1"]}}`

func TestFromXrayJSON(t *testing.T) {
	out, err := FromXrayJSON([]byte(providerConfig), 3090, 3091, 13345)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(out, &config); err != nil {
		t.Fatal(err)
	}
	if _, ok := config["remarks"]; ok {
		t.Errorf("remarks should be removed")
	}
	if _, ok := config["dns"]; !ok {
		t.Errorf("provider dns should be kept")
	}
	inbounds := config["inbounds"].([]any)
	if len(inbounds) != 3 {
		t.Fatalf("got %d inbounds", len(inbounds))
	}
	want := []struct {
		tag      string
		protocol string
		port     float64
	}{{"socks-provider", "socks", 3090}, {"http-provider", "http", 3091}, {"tproxy-in", "tunnel", 13345}}
	for i, w := range want {
		inbound := inbounds[i].(map[string]any)
		if inbound["tag"] != w.tag || inbound["protocol"] != w.protocol || inbound["port"] != w.port {
			t.Errorf("inbound %d = %v", i, inbound)
		}
	}

	out, _ = FromXrayJSON([]byte(providerConfig), 4000, -1, -1)
	json.Unmarshal(out, &config)
	if n := len(config["inbounds"].([]any)); n != 1 {
		t.Errorf("test config should only have socks inbound, got %d", n)
	}
}

func TestIsXrayJSON(t *testing.T) {
	if !IsXrayJSON(" {\"outbounds\":[]}") || IsXrayJSON("vless://x") {
		t.Fatal("IsXrayJSON misdetects")
	}
}

func TestFromXrayJSONTunDNSHijack(t *testing.T) {
	out, err := FromXrayJSON([]byte(providerConfig), 3090, 3091, 13345)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	json.Unmarshal(out, &config)

	outbounds := config["outbounds"].([]any)
	dnsOut := outbounds[len(outbounds)-1].(map[string]any)
	if dnsOut["protocol"] != "dns" || dnsOut["tag"] != xrayDNSOutboundTag {
		t.Fatalf("dns outbound missing: %v", dnsOut)
	}
	if outbounds[0].(map[string]any)["tag"] != "proxy" {
		t.Errorf("first outbound must stay the provider's default")
	}
	rules := config["routing"].(map[string]any)["rules"].([]any)
	first := rules[0].(map[string]any)
	if first["outboundTag"] != xrayDNSOutboundTag || first["port"] != "53" || first["inboundTag"].([]any)[0] != tproxyInboundTag {
		t.Errorf("dns hijack rule must come first: %v", first)
	}
	if len(rules) != 2 {
		t.Errorf("provider rules must be kept, got %d rules", len(rules))
	}

	out, _ = FromXrayJSON([]byte(providerConfig), 4000, -1, -1)
	config = nil
	json.Unmarshal(out, &config)
	if n := len(config["outbounds"].([]any)); n != 2 {
		t.Errorf("no dns outbound without TUN, got %d outbounds", n)
	}
}
