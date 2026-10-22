package tunscripts

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// NftTable is the nftables table that holds all TUN rules. Using a table of
// our own keeps happ-tui from clashing with chains other tools (tailscale,
// firewalld, docker) create in the shared iptables-compatible tables.
const NftTable = "happ_tui"

// legacyNftTable is the table name used before the rename from bushuray.
const legacyNftTable = "bushuray"

// tailscaleMark is the fwmark tailscaled puts on its own packets
// (value 0x80000 under mask 0xff0000), so they can bypass Xray.
const tailscaleMark = "meta mark & 0x00ff0000 == 0x00080000"

// noDNSHijackSubnets keep their DNS: the local stub resolver
// (systemd-resolved) and tailscale's MagicDNS (100.100.100.100). The stub's
// own upstream queries are still caught.
var noDNSHijackSubnets = []string{"127.0.0.0/8", "100.64.0.0/10"}

// ipv6Allowed is the IPv6 traffic that may still flow while TUN mode is on:
// loopback, link-local (neighbor discovery), multicast and unique local
// addresses (LAN and tailscale's fd7a:115c:a1e0::/48). Everything else would
// bypass Xray, which only handles IPv4 here.
var ipv6Allowed = []string{"::1/128", "fe80::/10", "ff00::/8", "fc00::/7"}

// reservedSubnets are never sent to Xray: private, link-local, CGNAT (which
// covers tailscale peers), loopback and multicast/reserved ranges.
var reservedSubnets = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"224.0.0.0/3",
}

func HasNft() bool {
	_, err := exec.LookPath("nft")
	return err == nil
}

// ApplyNftRules sends all TCP/UDP traffic of this machine (and of clients that
// use it as a gateway) to the Xray tproxy inbound, except Xray's own traffic,
// tailscale's traffic and local destinations. When hijackDNS is false, DNS
// bypasses Xray like with the iptables rules.
func ApplyNftRules(hijackDNS bool) error {
	local_subnets, err := runScriptWithSh(`
		ip address | grep -w inet | awk '{print $2}'
	`)
	if err != nil {
		return err
	}
	bypass, err := normalizeSubnets(append(reservedSubnets, extractSubnets(local_subnets)...))
	if err != nil {
		return err
	}

	prerouting_dns := "meta l4proto { tcp, udp } th dport 53 return"
	output_dns := "meta l4proto { tcp, udp } th dport 53 return"
	if hijackDNS {
		prerouting_dns = "ip daddr @no_dns_hijack return\n\t\tmeta l4proto { tcp, udp } th dport 53 tproxy to :13345 meta mark set 127 accept"
		output_dns = "ip daddr @no_dns_hijack return\n\t\tmeta l4proto { tcp, udp } th dport 53 meta mark set 127 return"
	}

	// "add table" + "delete table" makes the script idempotent
	script := fmt.Sprintf(`
add table ip %[1]s
delete table ip %[1]s
table ip %[1]s {
	set bypass {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { %[2]s }
	}

	set no_dns_hijack {
		type ipv4_addr
		flags interval
		elements = { %[5]s }
	}

	chain prerouting {
		type filter hook prerouting priority mangle; policy accept;
		%[3]s
		ip daddr @bypass return
		meta l4proto { tcp, udp } tproxy to :13345 meta mark set 127 accept
	}

	chain output {
		type route hook output priority mangle; policy accept;
		meta skgid 24333 return
		%[6]s return
		%[4]s
		ip daddr @bypass return
		meta l4proto { tcp, udp } meta mark set 127
	}
}
`, NftTable, strings.Join(bypass, ", "), prerouting_dns, output_dns,
		strings.Join(noDNSHijackSubnets, ", "), tailscaleMark)

	return runNft(script)
}

// BlockIPv6 rejects IPv6 traffic that would leave the machine past Xray.
// Rejecting rather than dropping makes applications fall back to IPv4 at once.
func BlockIPv6() error {
	script := fmt.Sprintf(`
add table ip6 %[1]s
delete table ip6 %[1]s
table ip6 %[1]s {
	set allowed {
		type ipv6_addr
		flags interval
		elements = { %[2]s }
	}

	chain output {
		type filter hook output priority filter; policy accept;
		oifname "lo" accept
		meta skgid 24333 accept
		%[3]s accept
		ip6 daddr @allowed accept
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpv6 type admin-prohibited
	}

	chain forward {
		type filter hook forward priority filter; policy accept;
		ip6 daddr @allowed accept
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpv6 type admin-prohibited
	}
}
`, NftTable, strings.Join(ipv6Allowed, ", "), tailscaleMark)
	return runNft(script)
}

func RemoveNftRules() error {
	if !HasNft() {
		return nil
	}
	var script strings.Builder
	for _, table := range []string{NftTable, legacyNftTable} {
		fmt.Fprintf(&script, "add table ip %[1]s\ndelete table ip %[1]s\nadd table ip6 %[1]s\ndelete table ip6 %[1]s\n", table)
	}
	return runNft(script.String())
}

func runNft(script string) error {
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nft: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// normalizeSubnets turns interface addresses like 192.168.1.5/24 into
// network prefixes nft accepts.
func normalizeSubnets(subnets []string) ([]string, error) {
	result := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		_, network, err := net.ParseCIDR(subnet)
		if err != nil {
			return nil, fmt.Errorf("invalid subnet %q: %w", subnet, err)
		}
		if network.IP.To4() == nil {
			continue
		}
		result = append(result, network.String())
	}
	return result, nil
}
