// Package scanner expands CIDR ranges and checks which hosts answer ICMP echo.
package scanner

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// MaxPrefixLen is the smallest accepted prefix length: /24 or narrower.
const MaxPrefixLen = 24

// ErrInvalidSubnet marks a subnet the scanner refuses; its message names the limit.
var ErrInvalidSubnet = errors.New("invalid subnet")

// ParseSubnet parses s as an IPv4 CIDR no wider than /24 and masks host bits.
func ParseSubnet(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(s))
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%w: %q is not CIDR notation (IPv4 only, /%d or narrower, e.g. 192.168.1.0/24)", ErrInvalidSubnet, s, MaxPrefixLen)
	}
	addr := p.Addr().Unmap()
	if !addr.Is4() || p.Addr().Is4In6() {
		return netip.Prefix{}, fmt.Errorf("%w: only IPv4 is supported (/%d or narrower)", ErrInvalidSubnet, MaxPrefixLen)
	}
	if p.Bits() < MaxPrefixLen {
		return netip.Prefix{}, fmt.Errorf("%w: /%d is too wide, use /%d or narrower (IPv4 only)", ErrInvalidSubnet, p.Bits(), MaxPrefixLen)
	}
	return netip.PrefixFrom(addr, p.Bits()).Masked(), nil
}

// Hosts lists the addresses to probe: network and broadcast are dropped when the
// prefix has more than two addresses (/24 gives 254, /31 gives 2, /32 gives 1).
func Hosts(p netip.Prefix) []netip.Addr {
	p = p.Masked()
	var all []netip.Addr
	for a := p.Addr(); p.Contains(a); a = a.Next() {
		all = append(all, a)
	}
	if len(all) > 2 {
		all = all[1 : len(all)-1]
	}
	return all
}
