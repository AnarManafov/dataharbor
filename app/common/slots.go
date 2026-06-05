package common

import (
	"fmt"
	"sync"
)

// SlotManager enforces a per-user concurrency limit on long-running operations
// (e.g. uploads, downloads). It is safe for concurrent use.
//
// Unlike the original download-slot logic in controller/xrd.go, this
// implementation:
//   - supports a configurable per-user capacity (not a hard-coded 1)
//   - returns an opaque release handle so callers cannot double-free or
//     accidentally release a slot they never acquired (the original bug)
//   - tracks total active slots across all users for observability
//
// Zero-value SlotManager is NOT valid; use NewSlotManager.
type SlotManager struct {
	name       string
	perUserCap int

	mu         sync.Mutex
	perUser    map[string]int
	totalInUse int
}

// SlotRelease is returned by Acquire and releases exactly one slot when called.
// It is idempotent: calling it more than once is a no-op. This prevents the
// double-release bug that disabled the original download slot logic.
type SlotRelease func()

// NewSlotManager creates a manager that allows up to perUserCap concurrent
// operations per user key. If perUserCap <= 0, the manager is effectively
// disabled (Acquire always succeeds and returns a no-op release).
func NewSlotManager(name string, perUserCap int) *SlotManager {
	return &SlotManager{
		name:       name,
		perUserCap: perUserCap,
		perUser:    make(map[string]int),
	}
}

// Acquire attempts to reserve a slot for the given user key. On success it
// returns a release function and nil. On failure it returns nil and an error
// describing why the slot could not be acquired.
//
// The returned release must be called exactly once when the operation
// completes (success or failure). Calling it more than once is safe.
func (s *SlotManager) Acquire(userKey string) (SlotRelease, error) {
	if s.perUserCap <= 0 {
		// Disabled: no limits
		return func() {}, nil
	}
	if userKey == "" {
		return nil, fmt.Errorf("%s: empty user key", s.name)
	}

	s.mu.Lock()
	current := s.perUser[userKey]
	if current >= s.perUserCap {
		total := s.totalInUse
		s.mu.Unlock()
		GetLogger().Infow("slot acquire denied",
			"manager", s.name,
			"userKey", userKey,
			"userInUse", current,
			"perUserCap", s.perUserCap,
			"totalInUse", total,
		)
		return nil, fmt.Errorf("%s: concurrent operation limit reached (%d per user)", s.name, s.perUserCap)
	}
	s.perUser[userKey] = current + 1
	s.totalInUse++
	totalInUse := s.totalInUse
	s.mu.Unlock()

	GetLogger().Debugw("slot acquired",
		"manager", s.name,
		"userKey", userKey,
		"userInUse", current+1,
		"totalInUse", totalInUse,
	)

	var once sync.Once
	return func() {
		once.Do(func() {
			s.release(userKey)
		})
	}, nil
}

// release decrements the counters. Internal only; callers use the returned
// SlotRelease from Acquire instead.
func (s *SlotManager) release(userKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.perUser[userKey]
	if current <= 0 {
		// Defensive: should never happen because SlotRelease is single-fire.
		GetLogger().Warnw("slot release called with no active slots",
			"manager", s.name,
			"userKey", userKey,
		)
		return
	}
	if current == 1 {
		delete(s.perUser, userKey)
	} else {
		s.perUser[userKey] = current - 1
	}
	s.totalInUse--

	GetLogger().Debugw("slot released",
		"manager", s.name,
		"userKey", userKey,
		"userInUse", current-1,
		"totalInUse", s.totalInUse,
	)
}

// InUse returns how many slots the given user currently holds.
func (s *SlotManager) InUse(userKey string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.perUser[userKey]
}

// TotalInUse returns how many slots are currently held across all users.
func (s *SlotManager) TotalInUse() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totalInUse
}

// PerUserCap returns the configured per-user capacity. A value of 0 or less
// means the manager is disabled (all Acquire calls succeed).
func (s *SlotManager) PerUserCap() int {
	return s.perUserCap
}
