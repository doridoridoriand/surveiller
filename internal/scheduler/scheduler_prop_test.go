package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
	"github.com/doridoridoriand/deadman-go/internal/state"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

type timestampPinger struct {
	mu    sync.Mutex
	times map[string][]time.Time
}

func (p *timestampPinger) Ping(ctx context.Context, addr string, timeout time.Duration) ping.Result {
	p.mu.Lock()
	p.times[addr] = append(p.times[addr], time.Now())
	p.mu.Unlock()
	return ping.Result{Success: true, RTT: 1 * time.Millisecond}
}

func (p *timestampPinger) count(addr string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.times[addr])
}

func (p *timestampPinger) snapshots(addr string) []time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]time.Time, len(p.times[addr]))
	copy(out, p.times[addr])
	return out
}

func TestPropertySchedulerConcurrencyLimit(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	props := gopter.NewProperties(params)

	props.Property("max concurrency is respected", prop.ForAll(
		func(maxConc, targetCount int) bool {
			if maxConc < 1 || targetCount < 1 {
				return true
			}
			pinger := &blockingPinger{started: make(chan struct{})}
			store := state.NewStore(nil, 5*time.Millisecond)
			targets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				targets[i] = config.TargetConfig{
					Name:    fmt.Sprintf("target-%d", i+1),
					Address: fmt.Sprintf("192.0.2.%d", i+1),
				}
			}
			s := NewScheduler(config.GlobalOptions{
				Interval:       1 * time.Millisecond,
				Timeout:        5 * time.Millisecond,
				MaxConcurrency: maxConc,
			}, targets, pinger, store)

			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
			defer cancel()

			go func() { _ = s.Run(ctx) }()

			select {
			case <-pinger.started:
			case <-ctx.Done():
				return false
			}

			<-ctx.Done()

			return int(atomic.LoadInt32(&pinger.max)) <= maxConc
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(4) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(4) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}

func TestPropertySchedulerTargetsStart(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	props := gopter.NewProperties(params)

	props.Property("each target receives pings", prop.ForAll(
		func(targetCount int) bool {
			if targetCount < 1 {
				return true
			}
			pinger := &timestampPinger{times: make(map[string][]time.Time)}
			store := state.NewStore(nil, 2*time.Millisecond)
			targets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				addr := fmt.Sprintf("192.0.2.%d", i+1)
				targets[i] = config.TargetConfig{Name: addr, Address: addr}
			}

			s := NewScheduler(config.GlobalOptions{
				Interval:       1 * time.Millisecond,
				Timeout:        2 * time.Millisecond,
				MaxConcurrency: targetCount,
			}, targets, pinger, store)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()

			go func() { _ = s.Run(ctx) }()
			<-ctx.Done()

			for _, tgt := range targets {
				if pinger.count(tgt.Address) == 0 {
					return false
				}
			}
			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(4) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}

func TestPropertySchedulerIntervalRespected(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 15
	props := gopter.NewProperties(params)

	props.Property("interval is not shorter than configured", prop.ForAll(
		func(intervalMs int) bool {
			if intervalMs < 5 {
				return true
			}
			interval := time.Duration(intervalMs) * time.Millisecond
			pinger := &timestampPinger{times: make(map[string][]time.Time)}
			store := state.NewStore(nil, 2*time.Millisecond)
			target := config.TargetConfig{Name: "a", Address: "192.0.2.1"}

			s := NewScheduler(config.GlobalOptions{
				Interval:       interval,
				Timeout:        2 * time.Millisecond,
				MaxConcurrency: 1,
			}, []config.TargetConfig{target}, pinger, store)

			ctx, cancel := context.WithTimeout(context.Background(), interval*4)
			defer cancel()

			go func() { _ = s.Run(ctx) }()
			<-ctx.Done()

			times := pinger.snapshots(target.Address)
			if len(times) < 2 {
				return true
			}
			minGap := interval - 1*time.Millisecond
			for i := 1; i < len(times); i++ {
				if times[i].Sub(times[i-1]) < minGap {
					return false
				}
			}
			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(15) + 5
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}

func TestPropertySchedulerStopsOnCancel(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 15
	props := gopter.NewProperties(params)

	props.Property("pings stop after cancellation", prop.ForAll(
		func(intervalMs int) bool {
			if intervalMs < 2 {
				return true
			}
			interval := time.Duration(intervalMs) * time.Millisecond
			pinger := &timestampPinger{times: make(map[string][]time.Time)}
			store := state.NewStore(nil, 2*time.Millisecond)
			target := config.TargetConfig{Name: "a", Address: "192.0.2.1"}

			s := NewScheduler(config.GlobalOptions{
				Interval:       interval,
				Timeout:        1 * time.Millisecond,
				MaxConcurrency: 1,
			}, []config.TargetConfig{target}, pinger, store)

			ctx, cancel := context.WithCancel(context.Background())
			go func() { _ = s.Run(ctx) }()

			time.Sleep(interval * 2)
			cancel()
			time.Sleep(interval * 2)
			countAfterCancel := pinger.count(target.Address)
			time.Sleep(interval * 2)

			return pinger.count(target.Address) == countAfterCancel
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 2
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}
