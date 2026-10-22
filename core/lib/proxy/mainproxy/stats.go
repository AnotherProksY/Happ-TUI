package mainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

const (
	statsInterval = 2 * time.Second
	// rateWindow is how far back the rate is averaged
	rateWindow = 5 * time.Second
)

type trafficSample struct {
	at       time.Time
	up, down int64
}

// statsPoller reads Xray's outbound counters while connected, so that
// `happ-tui status` only reads a cache.
type statsPoller struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	traffic *structs.TrafficCounters
	rate    *structs.TrafficCounters
	samples []trafficSample
}

func (s *statsPoller) start(apiPort int, proxyTags []string) {
	s.stop()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.traffic = &structs.TrafficCounters{}
	s.rate = nil
	s.samples = []trafficSample{{at: time.Now()}}
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(statsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			up, down, err := queryOutboundTraffic(ctx, apiPort, proxyTags)
			if err != nil {
				if ctx.Err() == nil {
					log.Println("failed to read traffic counters:", err)
				}
				continue
			}
			s.add(trafficSample{at: time.Now(), up: up, down: down})
		}
	}()
}

func (s *statsPoller) add(sample trafficSample) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel == nil {
		return
	}
	s.samples = append(s.samples, sample)
	for len(s.samples) > 2 && sample.at.Sub(s.samples[1].at) >= rateWindow {
		s.samples = s.samples[1:]
	}
	s.traffic = &structs.TrafficCounters{Up: sample.up, Down: sample.down}
	oldest := s.samples[0]
	if seconds := sample.at.Sub(oldest.at).Seconds(); seconds > 0 {
		s.rate = &structs.TrafficCounters{
			Up:   int64(float64(max(sample.up-oldest.up, 0)) / seconds),
			Down: int64(float64(max(sample.down-oldest.down, 0)) / seconds),
		}
	}
}

func (s *statsPoller) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.cancel = nil
	s.traffic = nil
	s.rate = nil
	s.samples = nil
}

func (s *statsPoller) get() (*structs.TrafficCounters, *structs.TrafficCounters) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var traffic, rate *structs.TrafficCounters
	if s.traffic != nil {
		copied := *s.traffic
		traffic = &copied
	}
	if s.rate != nil {
		copied := *s.rate
		rate = &copied
	}
	return traffic, rate
}

// queryOutboundTraffic sums the counters of the outbounds that carry VPN
// traffic, using the xray binary as the API client.
func queryOutboundTraffic(ctx context.Context, apiPort int, proxyTags []string) (int64, int64, error) {
	xraybin, err := utils.GetXrayBin()
	if err != nil {
		return 0, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, xraybin, "api", "statsquery",
		"--server=127.0.0.1:"+strconv.Itoa(apiPort), "-t", "2", "-pattern", "outbound>>>")
	if err := utils.Unprivileged(cmd); err != nil {
		return 0, 0, err
	}
	output, err := cmd.Output()
	if err != nil {
		var exit_err *exec.ExitError
		if errors.As(err, &exit_err) && len(exit_err.Stderr) > 0 {
			return 0, 0, fmt.Errorf("xray api statsquery: %w: %s", err, strings.TrimSpace(string(exit_err.Stderr)))
		}
		return 0, 0, fmt.Errorf("xray api statsquery: %w", err)
	}
	return sumOutboundTraffic(output, proxyTags)
}

func sumOutboundTraffic(output []byte, proxyTags []string) (int64, int64, error) {
	var response struct {
		Stat []struct {
			Name  string          `json:"name"`
			Value json.RawMessage `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return 0, 0, fmt.Errorf("parse stats: %w", err)
	}
	var up, down int64
	for _, stat := range response.Stat {
		// outbound>>>TAG>>>traffic>>>uplink
		parts := strings.Split(stat.Name, ">>>")
		if len(parts) != 4 || parts[0] != "outbound" || !slices.Contains(proxyTags, parts[1]) {
			continue
		}
		value, err := strconv.ParseInt(strings.Trim(string(stat.Value), `"`), 10, 64)
		if err != nil {
			continue
		}
		switch parts[3] {
		case "uplink":
			up += value
		case "downlink":
			down += value
		}
	}
	return up, down, nil
}
