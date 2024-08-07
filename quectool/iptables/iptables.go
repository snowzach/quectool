package iptables

import (
	"fmt"
	"regexp"

	"github.com/coreos/go-iptables/iptables"
	"github.com/spf13/cast"
)

var (
	ipv4TTLRule        = regexp.MustCompile(`-A\s+POSTROUTING\s+-o\s+rmnet\+\s+-j\s+TTL\s+--ttl-set\s+(\d+)`)
	ipv6TTLRule        = regexp.MustCompile(`-A\s+POSTROUTING\s+-o\s+rmnet\+\s+-j\s+HL\s+--hl-set\s+(\d+)`)
	ipv4PortAcceptRule = regexp.MustCompile(`-A\s+INPUT\s+-i\s+([a-z0-9+]+)\s+-p\s+tcp\s+-m\s+multiport\s+--dports\s+([\d,:]+)\s+-j\s+ACCEPT`)
	ipv4PortDropRule   = regexp.MustCompile(`-A\s+INPUT\s+-p\s+tcp\s+-m\s+multiport\s+--dports\s+([\d,:]+)\s+-j\s+DROP`)
)

type IPTables struct {
	ipv4t *iptables.IPTables
	ipv6t *iptables.IPTables

	ttlValue int
}

func NewIPTables() (*IPTables, error) {
	ipv4t, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize iptables interface: %w", err)
	}
	ipv6t, err := iptables.NewWithProtocol(iptables.ProtocolIPv6)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize ip6tables interface: %w", err)
	}

	return &IPTables{
		ipv4t: ipv4t,
		ipv6t: ipv6t,
	}, err
}

// SetTTLValue will set the mangle TTL value for the rmnet+ interface so all packets leaving will have that TTL.
func (i *IPTables) SetTTLValue(value int) error {

	list, err := i.ipv4t.List("mangle", "POSTROUTING")
	if err != nil {
		return fmt.Errorf("unable to list ipv4 mangle rules: %w", err)
	}

	var ipv4exists bool
	for _, rule := range list {
		if matches := ipv4TTLRule.FindStringSubmatch(rule); matches != nil {
			if value > 0 && !ipv4exists && cast.ToInt(matches[1]) == value {
				ipv4exists = true
			} else {
				if err := i.ipv4t.Delete("mangle", "POSTROUTING", "-o", "rmnet+", "-j", "TTL", "--ttl-set", matches[1]); err != nil {
					return fmt.Errorf("unable to delete ipv4 mangle rule: %w", err)
				}
			}
		}
	}
	if value > 0 && !ipv4exists {
		if err := i.ipv4t.Append("mangle", "POSTROUTING", "-o", "rmnet+", "-j", "TTL", "--ttl-set", cast.ToString(value)); err != nil {
			return fmt.Errorf("unable to append ipv4 mangle rule: %w", err)
		}
	}

	list, err = i.ipv6t.List("mangle", "POSTROUTING")
	if err != nil {
		return fmt.Errorf("unable to ipv6 mangle rules: %w", err)
	}

	var ipv6exists bool
	for _, rule := range list {
		if matches := ipv6TTLRule.FindStringSubmatch(rule); matches != nil {
			if value > 0 && !ipv6exists && cast.ToInt(matches[1]) == value {
				ipv6exists = true
			} else {
				if err := i.ipv6t.Delete("mangle", "POSTROUTING", "-o", "rmnet+", "-j", "HL", "--hl-set", matches[1]); err != nil {
					return fmt.Errorf("unable to delete ipv6 mangle rule: %w", err)
				}
			}
		}
	}
	if value > 0 && !ipv6exists {
		if err := i.ipv6t.Append("mangle", "POSTROUTING", "-o", "rmnet+", "-j", "HL", "--hl-set", cast.ToString(value)); err != nil {
			return fmt.Errorf("unable to append ipv6 mangle rule: %w", err)
		}
	}

	return nil
}

// AllowTCPPorts will allow the specified TCP ports on the specified interfaces and drop all other ports.
// If ports is an empty string, all rules will be removed.
// Port should be a comma separated list of ports or a range of ports. ex "22,80,443,10000:10100"
func (i *IPTables) AllowTCPPorts(interfaces []string, portsInts []int) error {

	ports := ""
	for _, port := range portsInts {
		if ports != "" {
			ports += ","
		}
		ports += cast.ToString(port)
	}

	list, err := i.ipv4t.List("filter", "INPUT")
	if err != nil {
		return fmt.Errorf("unable to list ipv4 filter rules: %w", err)
	}

	interfaceAcceptExists := make(map[string]bool)
	for _, iface := range interfaces {
		interfaceAcceptExists[iface] = false
	}

	// Delete all accept rules that don't match
	for _, rule := range list {
		if matches := ipv4PortAcceptRule.FindStringSubmatch(rule); matches != nil {
			if _, hasInterface := interfaceAcceptExists[matches[1]]; ports != "" && ports == matches[2] && hasInterface {
				interfaceAcceptExists[matches[1]] = true
			} else {
				if err := i.ipv4t.Delete("filter", "INPUT", "-i", matches[1], "-p", "tcp", "-m", "multiport", "--dports", matches[2], "-j", "ACCEPT"); err != nil {
					return fmt.Errorf("unable to delete ipv4 filter accept rule: %w", err)
				}
			}
		}
	}

	// Delete all drop rules
	for _, rule := range list {
		if matches := ipv4PortDropRule.FindStringSubmatch(rule); matches != nil {
			if err := i.ipv4t.Delete("filter", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", matches[1], "-j", "DROP"); err != nil {
				return fmt.Errorf("unable to delete ipv4 filter drop rule: %w", err)
			}
		}
	}

	if ports != "" {
		// Add back any missing accept rules
		for _, iface := range interfaces {
			if !interfaceAcceptExists[iface] {
				if err := i.ipv4t.Append("filter", "INPUT", "-i", iface, "-p", "tcp", "-m", "multiport", "--dports", ports, "-j", "ACCEPT"); err != nil {
					return fmt.Errorf("unable to append ipv4 filter accept rule: %w", err)
				}
			}
		}

		// Add back drop rules
		if err := i.ipv4t.Append("filter", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", ports, "-j", "DROP"); err != nil {
			return fmt.Errorf("unable to append ipv4 filter drop rule: %w", err)
		}
	}

	return nil

}
