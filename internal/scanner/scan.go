package scanner

import (
	"context"
	"net/netip"
	"sync"
)

// Concurrency caps simultaneous pings.
const Concurrency = 50

// Pinger reports whether ip answers an ICMP echo.
type Pinger interface {
	Ping(ctx context.Context, ip netip.Addr) (bool, error)
}

// Result is one scanned host.
type Result struct {
	IP    string `json:"ip"`
	Alive bool   `json:"alive"`
}

// Scan pings every host in prefix and returns the results in address order.
// The first error cancels the remaining work and is returned.
func Scan(ctx context.Context, prefix netip.Prefix, p Pinger) ([]Result, error) {
	hosts := Hosts(prefix)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	alive := make([]bool, len(hosts))
	sem := make(chan struct{}, Concurrency)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
loop:
	for i, ip := range hosts {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ok, err := p.Ping(ctx, ip)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				mu.Unlock()
				return
			}
			alive[i] = ok
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]Result, len(hosts))
	for i, ip := range hosts {
		out[i] = Result{IP: ip.String(), Alive: alive[i]}
	}
	return out, nil
}
