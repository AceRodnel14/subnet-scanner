package scanner

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakePinger func(ctx context.Context, ip netip.Addr) (bool, error)

func (f fakePinger) Ping(ctx context.Context, ip netip.Addr) (bool, error) { return f(ctx, ip) }

func TestParseSubnet(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		hosts   int
		wantErr bool
	}{
		{"192.168.1.0/24", "192.168.1.0/24", 254, false},
		{"192.168.1.77/24", "192.168.1.0/24", 254, false},
		{"10.0.0.4/30", "10.0.0.4/30", 2, false},
		{"10.0.0.0/31", "10.0.0.0/31", 2, false},
		{"10.0.0.5/32", "10.0.0.5/32", 1, false},
		{"192.168.0.0/23", "", 0, true},
		{"10.0.0.0/8", "", 0, true},
		{"fe80::/64", "", 0, true},
		{"::ffff:10.0.0.0/120", "", 0, true},
		{"garbage", "", 0, true},
		{"", "", 0, true},
		{"10.0.0.0", "", 0, true},
		{"10.0.0.0/33", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			p, err := ParseSubnet(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidSubnet) {
					t.Errorf("not ErrInvalidSubnet: %v", err)
				}
				return
			}
			if p.String() != tt.want {
				t.Errorf("prefix = %s, want %s", p, tt.want)
			}
			if n := len(Hosts(p)); n != tt.hosts {
				t.Errorf("hosts = %d, want %d", n, tt.hosts)
			}
		})
	}
}

func TestHostsExcludeNetworkAndBroadcast(t *testing.T) {
	h := Hosts(netip.MustParsePrefix("10.0.0.0/24"))
	if h[0].String() != "10.0.0.1" || h[len(h)-1].String() != "10.0.0.254" {
		t.Errorf("range %s..%s", h[0], h[len(h)-1])
	}
}

func TestScanSorted(t *testing.T) {
	// Later hosts finish first, so the order would be wrong without sorting.
	p := fakePinger(func(_ context.Context, ip netip.Addr) (bool, error) {
		time.Sleep(time.Duration(254-ip.As4()[3]) * 50 * time.Microsecond)
		return ip.As4()[3]%2 == 0, nil
	})
	got, err := Scan(context.Background(), netip.MustParsePrefix("10.0.0.0/24"), p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 254 {
		t.Fatalf("len = %d", len(got))
	}
	for i, r := range got {
		want := netip.AddrFrom4([4]byte{10, 0, 0, byte(i + 1)}).String()
		if r.IP != want || r.Alive != ((i+1)%2 == 0) {
			t.Fatalf("result %d = %+v", i, r)
		}
	}
}

func TestScanConcurrencyCap(t *testing.T) {
	var cur, peak atomic.Int32
	p := fakePinger(func(_ context.Context, _ netip.Addr) (bool, error) {
		n := cur.Add(1)
		for {
			m := peak.Load()
			if n <= m || peak.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		cur.Add(-1)
		return true, nil
	})
	if _, err := Scan(context.Background(), netip.MustParsePrefix("10.0.0.0/24"), p); err != nil {
		t.Fatal(err)
	}
	if peak.Load() > Concurrency || peak.Load() < 2 {
		t.Errorf("peak = %d, cap %d", peak.Load(), Concurrency)
	}
}

func TestScanErrorCancelsRest(t *testing.T) {
	boom := errors.New("boom")
	var calls atomic.Int32
	p := fakePinger(func(ctx context.Context, ip netip.Addr) (bool, error) {
		calls.Add(1)
		if ip.As4()[3] == 1 {
			return false, boom
		}
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
		return false, ctx.Err()
	})
	start := time.Now()
	res, err := Scan(context.Background(), netip.MustParsePrefix("10.0.0.0/24"), p)
	if !errors.Is(err, boom) || res != nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("error did not cancel the remaining pings")
	}
	if calls.Load() >= 254 {
		t.Errorf("calls = %d, expected fewer than all hosts", calls.Load())
	}
}

func TestScanContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var once sync.Once
	p := fakePinger(func(ctx context.Context, _ netip.Addr) (bool, error) {
		once.Do(cancel)
		<-ctx.Done()
		return false, ctx.Err()
	})
	_, err := Scan(ctx, netip.MustParsePrefix("10.0.0.0/24"), p)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
