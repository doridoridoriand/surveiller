package state

import (
	"sync"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
)

const (
	defaultHistorySize      = 100
	defaultDownThreshold    = 3
	thresholdDataPointCount = 10 // 閾値判定に使うデータポイント数
)

// StoreImpl is a thread-safe in-memory state store.
type StoreImpl struct {
	mu            sync.RWMutex
	targets       map[string]*TargetStatus
	historySize   int
	downThreshold int
	timeout       time.Duration
}

// NewStore creates a store initialized with the provided targets.
func NewStore(targets []config.TargetConfig, timeout time.Duration) *StoreImpl {
	store := &StoreImpl{
		targets:       make(map[string]*TargetStatus),
		historySize:   defaultHistorySize,
		downThreshold: defaultDownThreshold,
		timeout:       timeout,
	}
	store.UpdateTargets(targets)
	return store
}

// UpdateResult updates the target status based on a ping result.
func (s *StoreImpl) UpdateResult(name string, result ping.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	target, ok := s.targets[name]
	if !ok {
		target = &TargetStatus{Name: name, Status: StatusUnknown}
		s.targets[name] = target
	}

	now := time.Now()
	if result.Success {
		target.LastRTT = result.RTT
		target.LastSuccessAt = now
		target.ConsecutiveOK++
		target.ConsecutiveNG = 0
		target.TotalSuccess++

		// Historyに追加（判定前に追加して、直近のデータポイントを含める）
		s.appendHistory(target, result.RTT, now)

		// 直近N個のデータポイントの平均RTTで閾値判定
		avgRTT := calculateRecentAvgRTT(target.History, thresholdDataPointCount)
		if avgRTT <= 0 {
			// データポイントが不足している場合は最新のRTTを使用
			avgRTT = result.RTT
		}

		// RTTに基づいてOK/WARNを判定
		// OK: timeoutの25%以内
		// WARN: timeoutの25%超、50%以内
		// timeoutの50%超もWARNとして扱う
		okThreshold := s.timeout / 4   // 25%
		warnThreshold := s.timeout / 2 // 50%
		if avgRTT <= okThreshold {
			target.Status = StatusOK
		} else if avgRTT <= warnThreshold {
			target.Status = StatusWarn
		} else {
			// timeoutの50%超もWARNとして扱う
			target.Status = StatusWarn
		}
		return
	}

	target.LastFailureAt = now
	target.ConsecutiveNG++
	target.ConsecutiveOK = 0
	target.TotalFailure++
	if target.ConsecutiveNG >= s.downThreshold {
		target.Status = StatusDown
	} else {
		target.Status = StatusWarn
	}
}

// GetSnapshot returns a snapshot copy of all target states.
func (s *StoreImpl) GetSnapshot() []TargetStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]TargetStatus, 0, len(s.targets))
	for _, target := range s.targets {
		result = append(result, copyTargetStatus(target))
	}
	return result
}

// UpdateTargets updates the target list, keeping history for existing targets.
func (s *StoreImpl) UpdateTargets(targets []config.TargetConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := make(map[string]*TargetStatus, len(targets))
	for _, tgt := range targets {
		if existing, ok := s.targets[tgt.Name]; ok {
			existing.Address = tgt.Address
			existing.Group = tgt.Group
			updated[tgt.Name] = existing
			continue
		}
		updated[tgt.Name] = &TargetStatus{
			Name:    tgt.Name,
			Address: tgt.Address,
			Group:   tgt.Group,
			Status:  StatusUnknown,
		}
	}

	s.targets = updated
}

// GetTargetStatus returns a copy of a single target status.
func (s *StoreImpl) GetTargetStatus(name string) (TargetStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target, ok := s.targets[name]
	if !ok {
		return TargetStatus{}, false
	}
	return copyTargetStatus(target), true
}

func (s *StoreImpl) appendHistory(target *TargetStatus, rtt time.Duration, at time.Time) {
	point := RTTPoint{Time: at, RTT: rtt}
	if s.historySize <= 0 {
		return
	}
	if len(target.History) < s.historySize {
		target.History = append(target.History, point)
		return
	}
	copy(target.History, target.History[1:])
	target.History[len(target.History)-1] = point
}

func copyTargetStatus(source *TargetStatus) TargetStatus {
	clone := *source
	if len(source.History) > 0 {
		clone.History = append([]RTTPoint(nil), source.History...)
	}
	return clone
}

// calculateRecentAvgRTT calculates the average RTT from the most recent N data points.
// Returns 0 if there are no data points.
func calculateRecentAvgRTT(history []RTTPoint, count int) time.Duration {
	if len(history) == 0 {
		return 0
	}

	// 直近N個のデータポイントを使用（Historyは時系列順に並んでいる）
	start := len(history) - count
	if start < 0 {
		start = 0
	}

	var sum time.Duration
	for i := start; i < len(history); i++ {
		sum += history[i].RTT
	}

	usedCount := len(history) - start
	if usedCount == 0 {
		return 0
	}

	return sum / time.Duration(usedCount)
}
