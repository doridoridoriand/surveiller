package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
	"github.com/doridoridoriand/deadman-go/internal/state"
)

// Scheduler drives periodic ping execution.
type Scheduler interface {
	Run(ctx context.Context) error
	UpdateConfig(global config.GlobalOptions, targets []config.TargetConfig)
	Stop()
}

// Impl provides a default scheduler implementation.
type Impl struct {
	mu         sync.RWMutex
	cfg        config.GlobalOptions
	targets    map[string]config.TargetConfig
	pinger     ping.Pinger
	state      state.Store
	semaphore  chan struct{}
	targetJobs map[string]context.CancelFunc
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	runCtx     context.Context
}

// NewScheduler constructs a scheduler instance.
func NewScheduler(global config.GlobalOptions, targets []config.TargetConfig, pinger ping.Pinger, store state.Store) *Impl {
	s := &Impl{
		cfg:        global,
		targets:    make(map[string]config.TargetConfig),
		pinger:     pinger,
		state:      store,
		semaphore:  make(chan struct{}, maxConcurrency(global.MaxConcurrency)),
		targetJobs: make(map[string]context.CancelFunc),
	}
	for _, tgt := range targets {
		s.targets[tgt.Name] = tgt
	}
	return s
}

// Run starts ping loops for all targets and blocks until context cancellation.
func (s *Impl) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.runCtx = runCtx
	targets := make([]config.TargetConfig, 0, len(s.targets))
	for _, tgt := range s.targets {
		targets = append(targets, tgt)
	}
	s.mu.Unlock()

	for _, tgt := range targets {
		s.startTarget(runCtx, tgt)
	}

	<-runCtx.Done()
	s.wg.Wait()
	s.mu.Lock()
	s.cancel = nil
	s.runCtx = nil
	s.mu.Unlock()
	return runCtx.Err()
}

// UpdateConfig applies new global options and updates target goroutines.
func (s *Impl) UpdateConfig(global config.GlobalOptions, targets []config.TargetConfig) {
	s.mu.Lock()
	s.cfg = global
	s.semaphore = make(chan struct{}, maxConcurrency(global.MaxConcurrency))

	updated := make(map[string]config.TargetConfig, len(targets))
	for _, tgt := range targets {
		updated[tgt.Name] = tgt
	}

	runCtx := s.runCtx
	toStart := make([]config.TargetConfig, 0)
	toRestart := make([]config.TargetConfig, 0)
	toStop := make([]context.CancelFunc, 0)

	for name, tgt := range updated {
		existing, ok := s.targets[name]
		if !ok {
			toStart = append(toStart, tgt)
			continue
		}
		if existing.Address != tgt.Address {
			if cancel, ok := s.targetJobs[name]; ok {
				toStop = append(toStop, cancel)
				delete(s.targetJobs, name)
			}
			toRestart = append(toRestart, tgt)
		}
	}

	for name, cancel := range s.targetJobs {
		if _, ok := updated[name]; !ok {
			toStop = append(toStop, cancel)
			delete(s.targetJobs, name)
		}
	}

	s.targets = updated
	s.mu.Unlock()

	for _, cancel := range toStop {
		cancel()
	}
	if runCtx == nil {
		return
	}
	for _, tgt := range toStart {
		s.startTarget(runCtx, tgt)
	}
	for _, tgt := range toRestart {
		s.startTarget(runCtx, tgt)
	}
}

// Stop cancels all running target loops.
func (s *Impl) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Impl) startTarget(ctx context.Context, target config.TargetConfig) {
	s.mu.Lock()
	if _, ok := s.targetJobs[target.Name]; ok {
		s.mu.Unlock()
		return
	}
	targetCtx, cancel := context.WithCancel(ctx)
	s.targetJobs[target.Name] = cancel
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		s.runTargetLoop(targetCtx, target)
	}()
}

func (s *Impl) runTargetLoop(ctx context.Context, target config.TargetConfig) {
	for {
		interval, timeout := s.currentTiming()
		if interval <= 0 {
			interval = time.Second
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		sem, err := s.acquire(ctx)
		if err != nil {
			return
		}
		result := s.pingOnce(ctx, target.Address, timeout)
		s.release(sem)
		s.state.UpdateResult(target.Name, result)
	}
}

func (s *Impl) pingOnce(ctx context.Context, addr string, timeout time.Duration) ping.Result {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.pinger.Ping(pingCtx, addr, timeout)
}

func (s *Impl) acquire(ctx context.Context) (chan struct{}, error) {
	sem := s.currentSemaphore()
	select {
	case sem <- struct{}{}:
		return sem, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Impl) release(sem chan struct{}) {
	select {
	case <-sem:
	default:
	}
}

func (s *Impl) currentTiming() (time.Duration, time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Interval, s.cfg.Timeout
}

func (s *Impl) currentSemaphore() chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.semaphore
}

func maxConcurrency(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}
