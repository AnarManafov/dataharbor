package common

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlotManager_AcquireRelease(t *testing.T) {
	sm := NewSlotManager("test", 2)

	rel1, err := sm.Acquire("alice")
	assert.NoError(t, err)
	assert.NotNil(t, rel1)
	assert.Equal(t, 1, sm.InUse("alice"))
	assert.Equal(t, 1, sm.TotalInUse())

	rel2, err := sm.Acquire("alice")
	assert.NoError(t, err)
	assert.Equal(t, 2, sm.InUse("alice"))

	// Third acquire should fail
	rel3, err := sm.Acquire("alice")
	assert.Error(t, err)
	assert.Nil(t, rel3)

	// Different user not blocked
	relBob, err := sm.Acquire("bob")
	assert.NoError(t, err)
	assert.Equal(t, 1, sm.InUse("bob"))
	assert.Equal(t, 3, sm.TotalInUse())

	rel1()
	assert.Equal(t, 1, sm.InUse("alice"))

	// Now alice can acquire again
	rel4, err := sm.Acquire("alice")
	assert.NoError(t, err)

	rel2()
	rel4()
	relBob()
	assert.Equal(t, 0, sm.InUse("alice"))
	assert.Equal(t, 0, sm.InUse("bob"))
	assert.Equal(t, 0, sm.TotalInUse())
}

func TestSlotManager_ReleaseIsIdempotent(t *testing.T) {
	sm := NewSlotManager("test", 1)
	rel, err := sm.Acquire("alice")
	assert.NoError(t, err)

	rel()
	rel() // second call must not panic nor corrupt counters
	rel()

	assert.Equal(t, 0, sm.InUse("alice"))
	assert.Equal(t, 0, sm.TotalInUse())

	// Alice should be able to acquire again
	rel2, err := sm.Acquire("alice")
	assert.NoError(t, err)
	rel2()
}

func TestSlotManager_Disabled(t *testing.T) {
	sm := NewSlotManager("test", 0)
	// Any number of acquires should succeed
	releases := make([]SlotRelease, 100)
	for i := range releases {
		r, err := sm.Acquire("alice")
		assert.NoError(t, err)
		releases[i] = r
	}
	// Counters stay at zero when disabled
	assert.Equal(t, 0, sm.InUse("alice"))
	assert.Equal(t, 0, sm.TotalInUse())
	for _, r := range releases {
		r()
	}
}

func TestSlotManager_EmptyUserKey(t *testing.T) {
	sm := NewSlotManager("test", 2)
	rel, err := sm.Acquire("")
	assert.Error(t, err)
	assert.Nil(t, rel)
}

func TestSlotManager_ConcurrentAcquireRelease(t *testing.T) {
	sm := NewSlotManager("test", 5)

	const goroutines = 50
	const iterations = 200
	var successCount atomic.Int64
	var maxObserved atomic.Int64

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rel, err := sm.Acquire("shared")
				if err != nil {
					continue
				}
				successCount.Add(1)

				// Record peak usage
				inUse := int64(sm.InUse("shared"))
				for {
					cur := maxObserved.Load()
					if inUse <= cur || maxObserved.CompareAndSwap(cur, inUse) {
						break
					}
				}

				rel()
			}
		}()
	}
	wg.Wait()

	// Cap must never be exceeded
	assert.LessOrEqual(t, maxObserved.Load(), int64(5), "per-user cap exceeded under contention")
	// All slots released
	assert.Equal(t, 0, sm.InUse("shared"))
	assert.Equal(t, 0, sm.TotalInUse())
	// Many successes expected (sanity)
	assert.Greater(t, successCount.Load(), int64(100))
}

func TestSlotManager_PerUserCap(t *testing.T) {
	sm := NewSlotManager("test", 7)
	assert.Equal(t, 7, sm.PerUserCap())
}
