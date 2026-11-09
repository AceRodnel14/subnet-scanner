package scanner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	probes       = 3
	probeTimeout = time.Second
)

// ErrNotPermitted means the kernel refused an unprivileged ICMP socket.
var ErrNotPermitted = errors.New("ICMP ping sockets not permitted; set net.ipv4.ping_group_range to include GID 65532")

// ICMPPinger pings with unprivileged ICMP datagram sockets (udp4).
type ICMPPinger struct{}

// Ping opens one socket per call (the kernel assigns the echo ID per socket) and
// sends three probes, one second apart at most.
func (ICMPPinger) Ping(ctx context.Context, ip netip.Addr) (bool, error) {
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return false, listenErr(err)
	}
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	dst := &net.UDPAddr{IP: ip.AsSlice()}
	buf := make([]byte, 1500)
	for seq := 1; seq <= probes; seq++ {
		msg := icmp.Message{Type: ipv4.ICMPTypeEcho, Body: &icmp.Echo{ID: 0, Seq: seq, Data: []byte("subnet-scanner")}}
		b, err := msg.Marshal(nil)
		if err != nil {
			return false, err
		}
		if _, err := conn.WriteTo(b, dst); err != nil {
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			continue // unreachable or no route: treat as no answer
		}
		deadline := time.Now().Add(probeTimeout)
		_ = conn.SetReadDeadline(deadline)
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil {
					return false, ctx.Err()
				}
				break // deadline exceeded: next probe
			}
			m, err := icmp.ParseMessage(1, buf[:n])
			if err == nil && m.Type == ipv4.ICMPTypeEchoReply {
				return true, nil
			}
		}
	}
	return false, nil
}

// listenErr marks permission errors as ErrNotPermitted; others (EMFILE, ENOBUFS, ...) pass through.
func listenErr(err error) error {
	if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
		return fmt.Errorf("%w: %w", ErrNotPermitted, err)
	}
	return fmt.Errorf("icmp listen: %w", err)
}
