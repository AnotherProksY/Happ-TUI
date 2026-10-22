package mainproxy

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	portpool "github.com/AnotherProksY/Happ-TUI/core/lib/PortPool"
	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
	"github.com/AnotherProksY/Happ-TUI/core/lib/proxy/builder"
	"github.com/AnotherProksY/Happ-TUI/core/lib/proxy/xray"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

type ProxyManager struct {
	status            structs.ProxyStatus
	mu                sync.Mutex
	appConfig         config.AppConfig
	xray_core         xray.XrayCore
	StatusChanged     chan structs.ProxyStatus
	testChannel       chan TestRequest
	TestResultChannel chan TestResult
	testGeneration    atomic.Uint64
	portPool          *portpool.PortPool
}

func (p *ProxyManager) Init(appConfig config.AppConfig) {
	p.status = structs.ProxyStatus{
		Connection:   "disconnected",
		IsTunEnabled: false,
	}
	p.appConfig = appConfig
	p.StatusChanged = make(chan structs.ProxyStatus)
	p.TestResultChannel = make(chan TestResult)
	test_channel := make(chan TestRequest)
	p.testChannel = test_channel
	go p.listenForTests(test_channel)
	p.xray_core = xray.XrayCore{
		Exited: make(chan error, 1),
	}
	test_port_range := appConfig.TestPortRange
	p.portPool = portpool.CreatePortPool(test_port_range.Start, test_port_range.End)
	if err := p.recoverPersistedTunMode(); err != nil {
		log.Println("failed to recover persisted TUN state:", err)
	}
}

var (
	// ErrNotConnected means a TUN change was asked for without a connection.
	ErrNotConnected = errors.New("no profile is currently active")
	// ErrNotRoot means TUN mode was asked for while the core isn't root.
	ErrNotRoot = errors.New("TUN mode needs the core to run as root: start happ-tui with sudo or enable the happ-tui-core service")
	// ErrProfile means the profile could not be turned into an Xray config.
	ErrProfile = errors.New("invalid profile")
	// ErrXray means Xray failed to start or exited right away.
	ErrXray = errors.New("xray failed to start")
	// ErrTun means setting up the TUN rules failed.
	ErrTun = errors.New("failed to enable TUN mode")
)

// xrayStartupGrace is how long Xray must stay up to count as started; a bad
// config makes it exit well within this.
const xrayStartupGrace = 500 * time.Millisecond

func (p *ProxyManager) ChangeTunMode(tun_mode bool) error {
	status := p.GetStatus()
	if tun_mode == status.IsTunEnabled {
		return nil
	}
	if status.Connection != "connected" {
		return ErrNotConnected
	}
	return p.Connect(status.Profile, tun_mode)
}

// Connect (re)starts Xray with profile. On failure the status becomes
// "error" with the reason in last_error.
func (p *ProxyManager) Connect(profile structs.Profile, tun_mode bool) error {
	if tun_mode && !utils.IsRoot() {
		return ErrNotRoot
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.connect(profile, tun_mode)
	if err != nil {
		log.Println("connect failed:", err)
		p.status = structs.ProxyStatus{
			Connection: "error",
			Profile:    profile,
			LastError:  err.Error(),
		}
		p.StatusChanged <- p.status
	}
	return err
}

func (p *ProxyManager) connect(profile structs.Profile, tun_mode bool) error {
	if p.status.IsTunEnabled || tunStateExists() {
		if err := p.disableTun(); err != nil {
			return fmt.Errorf("%w: disable previous TUN mode: %v", ErrTun, err)
		}
	}
	if p.xray_core.IsRunning() {
		p.xray_core.Stop()
	}

	p.xray_core = xray.XrayCore{
		Exited: make(chan error, 1),
	}

	p.status = structs.ProxyStatus{
		Connection: "connecting",
		Profile:    profile,
	}
	p.StatusChanged <- p.status

	tproxy_port := -1
	if tun_mode {
		tproxy_port = 13345
	}

	xrayCoreConfig, err := builder.ProfileConfig(
		profile,
		p.appConfig.SocksPort,
		p.appConfig.HttpPort,
		tproxy_port,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProfile, err)
	}

	// Full Xray configs from a JSON subscription bring the provider's own dns
	// and routing, which are kept as they are, like Happ does.
	isXrayJSON := builder.IsXrayJSON(profile.Uri)
	builder := builder.NewBuilder(xrayCoreConfig)
	if !isXrayJSON {
		if err := builder.ApplyDNS(p.appConfig.Dns); err != nil {
			return fmt.Errorf("%w: %v", ErrProfile, err)
		}
	}
	xrayConfig := builder.Build()

	if tun_mode {
		log.Println("enabling TUN with Xray TProxy on port 13345; no TUN network interface will be created")
		if err := p.prepareTunMode(); err != nil {
			return fmt.Errorf("%w: %v", ErrTun, err)
		}
		// NET_ADMIN for IP_TRANSPARENT and socket marks; NET_BIND_SERVICE
		// because answers to intercepted UDP (DNS on 53, NTP on 123) are
		// sent from a socket bound to the original, often privileged, port
		if err := p.xray_core.Start(xrayConfig, utils.CAP_NET_ADMIN, utils.CAP_NET_BIND_SERVICE); err != nil {
			return fmt.Errorf("%w: %v", ErrXray, err)
		}
	} else {
		var caps []uintptr
		if p.appConfig.SocksPort < 1024 || p.appConfig.HttpPort < 1024 {
			caps = append(caps, utils.CAP_NET_BIND_SERVICE)
		}
		if err := p.xray_core.Start(xrayConfig, caps...); err != nil {
			return fmt.Errorf("%w: %v", ErrXray, err)
		}
	}

	// a bad config makes Xray exit right away; catch that before the TUN
	// rules send all traffic to it
	select {
	case exitErr, ok := <-p.xray_core.Exited:
		p.xray_core.Stop()
		if ok && exitErr != nil {
			return fmt.Errorf("%w: %v", ErrXray, exitErr)
		}
		return fmt.Errorf("%w: it exited right away, see the logs", ErrXray)
	case <-time.After(xrayStartupGrace):
	}

	if tun_mode {
		log.Println("TUN Xray process started; configuring policy routing")
		// JSON profiles bring the provider's dns, which handles DNS in TUN
		// mode like Happ does; share links follow the dns setting
		hijackDNS := isXrayJSON || p.appConfig.Dns.Mode != config.DNSModeSystem
		if err := p.enableTun(hijackDNS); err != nil {
			p.xray_core.Stop()
			return fmt.Errorf("%w: %v", ErrTun, err)
		}
		log.Println("successfully enabled tun")
	}

	p.status = structs.ProxyStatus{
		Connection:     "connected",
		IsTunEnabled:   tun_mode,
		Profile:        profile,
		ConnectedSince: time.Now().Unix(),
	}
	p.StatusChanged <- p.status
	log.Println("changing connection status to", p.status.Connection)

	exited := p.xray_core.Exited
	go func(exited <-chan error) {
		for {
			exitErr, ok := <-exited
			if !ok {
				return
			}
			p.mu.Lock()
			if p.xray_core.Exited != exited {
				p.mu.Unlock()
				return
			}
			if p.status.IsTunEnabled || tunStateExists() {
				if err := p.disableTun(); err != nil {
					log.Println("failed to clean TUN mode after Xray exit:", err)
				}
			}
			last_error := "xray exited"
			if exitErr != nil {
				last_error = "xray exited: " + exitErr.Error()
			}
			p.status = structs.ProxyStatus{
				Connection: "error",
				Profile:    p.status.Profile,
				LastError:  last_error,
			}
			status := p.status
			p.mu.Unlock()
			p.StatusChanged <- status
			log.Println(last_error)
		}
	}(exited)

	return nil
}

// SetLastError records a problem that didn't end the connection, e.g. TUN
// failing while the proxy stays up.
func (p *ProxyManager) SetLastError(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.LastError = message
	p.StatusChanged <- p.status
}

func (p *ProxyManager) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.xray_core.Stop()

	var cleanupErr error
	if p.status.IsTunEnabled || tunStateExists() {
		cleanupErr = p.disableTun()
	}
	p.status = structs.ProxyStatus{
		IsTunEnabled: false,
		Connection:   "disconnected",
	}
	p.StatusChanged <- p.status
	return wrapRollbackError(cleanupErr)
}

func (p *ProxyManager) GetStatus() structs.ProxyStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// Traffic is the traffic through the proxy outbounds since connecting and
// its current rate, or nil while unknown.
func (p *ProxyManager) Traffic() (*structs.TrafficCounters, *structs.TrafficCounters) {
	return nil, nil
}
