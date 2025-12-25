package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
	"github.com/doridoridoriand/deadman-go/internal/state"
)

func TestSchedulerMaxConcurrency(t *testing.T) {
	pinger := &blockingPinger{started: make(chan struct{})}
	store := state.NewStore(nil)
	targets := []config.TargetConfig{
		{Name: "a", Address: "192.0.2.1"},
		{Name: "b", Address: "192.0.2.2"},
	}

	s := NewScheduler(config.GlobalOptions{
		Interval:       1 * time.Millisecond,
		Timeout:        5 * time.Millisecond,
		MaxConcurrency: 1,
	}, targets, pinger, store)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	go func() { _ = s.Run(ctx) }()

	select {
	case <-pinger.started:
	case <-ctx.Done():
		t.Fatalf("scheduler did not start ping loop")
	}

	<-ctx.Done()

	if max := atomic.LoadInt32(&pinger.max); max > 1 {
		t.Fatalf("expected max concurrency 1, got %d", max)
	}
}

func TestSchedulerUpdateConfigStartsNewTarget(t *testing.T) {
	recorder := &recordingPinger{seen: make(map[string]int)}
	store := state.NewStore(nil)

	initial := []config.TargetConfig{{Name: "a", Address: "192.0.2.1"}}
	s := NewScheduler(config.GlobalOptions{
		Interval:       1 * time.Millisecond,
		Timeout:        2 * time.Millisecond,
		MaxConcurrency: 2,
	}, initial, recorder, store)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	go func() { _ = s.Run(ctx) }()

	recorder.waitFor(t, "192.0.2.1", 1, ctx)

	updated := []config.TargetConfig{
		{Name: "a", Address: "192.0.2.1"},
		{Name: "b", Address: "192.0.2.2"},
	}
	s.UpdateConfig(s.cfg, updated)

	recorder.waitFor(t, "192.0.2.2", 1, ctx)
}

type blockingPinger struct {
	inFlight int32
	max      int32
	started  chan struct{}
}

func (p *blockingPinger) Ping(ctx context.Context, addr string, timeout time.Duration) ping.Result {
	current := atomic.AddInt32(&p.inFlight, 1)
	defer atomic.AddInt32(&p.inFlight, -1)

	for {
		max := atomic.LoadInt32(&p.max)
		if current <= max {
			break
		}
		if atomic.CompareAndSwapInt32(&p.max, max, current) {
			break
		}
	}

	select {
	case <-p.started:
	default:
		close(p.started)
	}

	<-ctx.Done()
	return ping.Result{Success: false, Error: ctx.Err()}
}

type recordingPinger struct {
	mu   sync.Mutex
	seen map[string]int
}

func (p *recordingPinger) Ping(ctx context.Context, addr string, timeout time.Duration) ping.Result {
	p.mu.Lock()
	p.seen[addr]++
	p.mu.Unlock()
	return ping.Result{Success: true, RTT: 1 * time.Millisecond}
}

func (p *recordingPinger) waitFor(t *testing.T, addr string, count int, ctx context.Context) {
	t.Helper()
	for {
		p.mu.Lock()
		seen := p.seen[addr]
		p.mu.Unlock()
		if seen >= count {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for %s", addr)
		case <-time.After(1 * time.Millisecond):
		}
	}
}
