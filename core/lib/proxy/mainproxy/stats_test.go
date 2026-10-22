package mainproxy

import (
	"testing"
	"time"
)

func TestSumOutboundTraffic(t *testing.T) {
	output := []byte(`{"stat":[
		{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"100"},
		{"name":"outbound>>>proxy-2>>>traffic>>>downlink","value":2000},
		{"name":"outbound>>>proxy>>>traffic>>>downlink"},
		{"name":"outbound>>>direct>>>traffic>>>downlink","value":"999"},
		{"name":"inbound>>>socks>>>traffic>>>uplink","value":"5"}]}`)
	up, down, err := sumOutboundTraffic(output, []string{"proxy", "proxy-2"})
	if err != nil {
		t.Fatal(err)
	}
	if up != 100 || down != 2000 {
		t.Errorf("up=%d down=%d", up, down)
	}
	if _, _, err := sumOutboundTraffic([]byte(`{}`), nil); err != nil {
		t.Errorf("an empty answer is no error: %v", err)
	}
}

func TestStatsRate(t *testing.T) {
	s := &statsPoller{cancel: func() {}}
	start := time.Now()
	s.samples = []trafficSample{{at: start}}
	s.add(trafficSample{at: start.Add(2 * time.Second), up: 200, down: 2000})
	s.add(trafficSample{at: start.Add(4 * time.Second), up: 400, down: 4000})
	traffic, rate := s.get()
	if traffic.Up != 400 || traffic.Down != 4000 {
		t.Errorf("traffic = %+v", traffic)
	}
	if rate.Up != 100 || rate.Down != 1000 {
		t.Errorf("rate = %+v", rate)
	}
	// samples older than the window are dropped, so the rate follows changes
	for i := 3; i <= 8; i++ {
		s.add(trafficSample{at: start.Add(time.Duration(2*i) * time.Second), up: 400, down: 4000})
	}
	if _, rate := s.get(); rate.Up != 0 || rate.Down != 0 {
		t.Errorf("idle rate = %+v", rate)
	}
}
