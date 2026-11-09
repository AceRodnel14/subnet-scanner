package scanner

import (
	"context"
	"errors"
	"net/netip"
	"syscall"
	"testing"
	"time"
)

func TestICMPPingLoopback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ok, err := ICMPPinger{}.Ping(ctx, netip.MustParseAddr("127.0.0.1"))
	if errors.Is(err, ErrNotPermitted) {
		t.Skipf("ICMP sockets not permitted: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("127.0.0.1 not alive")
	}
}

func TestICMPPingCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// 192.0.2.1 (TEST-NET-1) never answers; a canceled ctx must return fast.
	start := time.Now()
	ok, err := ICMPPinger{}.Ping(ctx, netip.MustParseAddr("192.0.2.1"))
	if errors.Is(err, ErrNotPermitted) {
		t.Skipf("ICMP sockets not permitted: %v", err)
	}
	if ok || time.Since(start) > 2*time.Second {
		t.Errorf("ok=%v after %v err=%v", ok, time.Since(start), err)
	}
}

func TestListenErr(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want bool
	}{
		{syscall.EACCES, true},
		{syscall.EPERM, true},
		{syscall.EMFILE, false},
		{syscall.ENOBUFS, false},
	} {
		got := listenErr(tt.err)
		if errors.Is(got, ErrNotPermitted) != tt.want || !errors.Is(got, tt.err) {
			t.Errorf("%v: got %v, want notPermitted=%v", tt.err, got, tt.want)
		}
	}
}
