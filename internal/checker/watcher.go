package checker

import (
	"context"
	"math"
	"sync"
	"time"
)

// WatchConfig holds configuration for continuous monitoring
type WatchConfig struct {
	Interval   time.Duration // Check interval
	WarnDays  int
	CriticalDays int
}

// Watcher continuously monitors hosts at a fixed interval
func WatchHosts(ctx context.Context, hosts []Host, cfg WatchConfig, onResults func([]Result)) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Run immediately on start
	runCheck(ctx, hosts, cfg.WarnDays, cfg.CriticalDays, onResults)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCheck(ctx, hosts, cfg.WarnDays, cfg.CriticalDays, onResults)
		}
	}
}

// runCheck runs checks with a timeout
func runCheck(ctx context.Context, hosts []Host, warnDays, criticalDays int, onResults func([]Result)) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	done := make(chan struct{})
	var wg sync.WaitGroup

	results := make([]Result, len(hosts))
	var mu sync.Mutex

	for i, h := range hosts {
		wg.Add(1)
		go func(idx int, host string, port int) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				r := checkHost(host, port, warnDays, criticalDays)
				mu.Lock()
				results[idx] = r
				mu.Unlock()
			}
		}(i, h.Host, h.Port)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return
	case <-done:
		onResults(results)
	}
}
