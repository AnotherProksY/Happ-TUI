package mainproxy

import (
	"errors"
	"fmt"
	"log"

	tunscripts "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy/scripts"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

func (p *ProxyManager) prepareTunMode() error {
	if err := tunscripts.RemoveLegacyUser(); err != nil {
		log.Println("failed to remove the legacy bxray_tproxy user:", err)
	}
	_, _, err := utils.XrayUserIDs()
	return err
}

// enableTun redirects all traffic of the machine to Xray. With hijackDNS,
// DNS queries leaving the machine are answered by Xray too.
func (p *ProxyManager) enableTun(hijackDNS bool) error {
	if err := p.beginTunMode(); err != nil {
		return err
	}
	xrayUID, _, err := utils.XrayUserIDs()
	if err != nil {
		return err
	}

	type tunStep struct {
		name string
		run  func() error
	}
	var steps []tunStep
	if tunscripts.HasNft() {
		steps = []tunStep{
			{name: "generate routing table", run: tunscripts.GenerateRoutingTable},
			{name: "apply nftables rules", run: func() error { return tunscripts.ApplyNftRules(hijackDNS, xrayUID) }},
			{name: "block IPv6", run: func() error { return tunscripts.BlockIPv6(xrayUID) }},
			{name: "loosen rp_filter", run: tunscripts.LoosenRpFilters},
		}
	} else {
		steps = []tunStep{
			{name: "generate routing table", run: tunscripts.GenerateRoutingTable},
			{name: "create TProxy chain", run: tunscripts.BypassLocalAndLan},
			{name: "configure TProxy targets", run: tunscripts.MarkAndForwardPackets},
			{name: "attach PREROUTING chain", run: tunscripts.ApplyMainRules},
			{name: "configure OUTPUT routing", run: func() error { return tunscripts.ProxyGateway(xrayUID) }},
			{name: "loosen rp_filter", run: tunscripts.LoosenRpFilters},
		}
		if hijackDNS {
			steps = append(steps, tunStep{name: "hijack DNS", run: tunscripts.HijackDns})
		} else {
			steps = append(steps, tunStep{name: "bypass system DNS", run: tunscripts.BypassDns})
		}
	}

	for _, step := range steps {
		log.Println("TUN setup:", step.name)
		if err := step.run(); err != nil {
			rollbackErr := p.rollbackTunMode()
			return errors.Join(
				fmt.Errorf("%s: %w", step.name, err),
				wrapRollbackError(rollbackErr),
			)
		}
	}

	log.Println("TUN ready")
	return nil
}

func (p *ProxyManager) disableTun() error {
	return p.rollbackTunMode()
}

func wrapRollbackError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("roll back TUN mode: %w", err)
}
