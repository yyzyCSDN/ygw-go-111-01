package door

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// stubActuator lets callers pin per-door Drive/Confirm behaviour. If Confirm is
// nil it just returns true after the configured delay, mimicking SimActuator.
type stubActuator struct {
	mu        sync.Mutex
	delay     time.Duration
	confirmFn func(id string, open bool) (bool, error)
}

func (a *stubActuator) Drive(id string, open bool) error { return nil }
func (a *stubActuator) Confirm(id string, open bool) (bool, error) {
	a.mu.Lock()
	fn := a.confirmFn
	a.mu.Unlock()
	if fn != nil {
		return fn(id, open)
	}
	time.Sleep(a.delay)
	return true, nil
}

type fakeReleaser struct {
	mu     sync.Mutex
	calls  []string
}

func (f *fakeReleaser) ReleaseDoor(doorID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, doorID)
}

func (f *fakeReleaser) released(doorID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range f.calls {
		if id == doorID {
			return true
		}
	}
	return false
}

// newTestDoor returns a door primed in the open state so Close() can run.
// The trap interval is shrunk so Confirm() is actually polled within the
// (short) close timeout used by the tests.
func newTestDoor(id string) *Door {
	d := New(id, 1, 0, Config{TrapInterval: 5 * time.Millisecond})
	d.state = StateOpen
	return d
}

// TestCloseBatchContinuesAfterError reproduces the scenario the operator saw:
// during a departure-linked close, one door's confirmation never comes back.
// Before the fix, CloseBatch returned on the first error and never attempted
// the remaining doors — so the whole close chain stalled and later interlocked
// actions ground to a halt.
func TestCloseBatchContinuesAfterError(t *testing.T) {
	// PSD01's close confirm never locks (returns false forever → confirm timeout);
	// PSD02 confirms normally. The trap interval (set per-door) must be shorter
	// than the timeout so Confirm is actually polled within the window.
	act := &stubActuator{
		delay: time.Millisecond,
		confirmFn: func(id string, open bool) (bool, error) {
			if id == "PSD01" {
				return false, nil // never reaches locked/closed
			}
			return true, nil
		},
	}
	rel := &fakeReleaser{}

	doors := []*Door{newTestDoor("PSD01"), newTestDoor("PSD02")}
	ctx := context.Background()
	// timeout well above the 5ms trap interval so Confirm is sampled in time.
	timeout := 150 * time.Millisecond

	errs := CloseBatch(ctx, doors, act, nil, rel, nil, nil, timeout)

	// PSD01 must report its confirm-timeout error.
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error from PSD01, got %v", errs)
	}
	if !errors.Is(errs[0], ErrConfirm) {
		t.Fatalf("expected ErrConfirm from PSD01, got %v", errs[0])
	}

	// The chain must NOT have stopped: PSD02 must have been attempted and closed.
	if got := doors[1].State(); got != StateClosed {
		t.Fatalf("expected PSD02 to reach closed despite PSD01 failing, got state=%s", got)
	}

	// And the failing door's grant must still have been released (its own
	// timeout path releases it), so it does not leak into the interlock.
	if !rel.released("PSD01") {
		t.Fatalf("expected PSD01 grant released on its confirm timeout")
	}
}

// TestCloseBatchCollectsMultipleErrors confirms the batch reports every failing
// door, not just the first, so the operator can see the full picture.
func TestCloseBatchCollectsMultipleErrors(t *testing.T) {
	act := &stubActuator{
		delay: time.Millisecond,
		confirmFn: func(id string, open bool) (bool, error) {
			return false, nil // every door times out on confirm
		},
	}
	rel := &fakeReleaser{}

	doors := []*Door{newTestDoor("PSD01"), newTestDoor("PSD02"), newTestDoor("PSD03")}
	ctx := context.Background()
	// timeout above the 5ms trap interval so each door actually polls Confirm.
	timeout := 120 * time.Millisecond

	errs := CloseBatch(ctx, doors, act, nil, rel, nil, nil, timeout)

	if len(errs) != 3 {
		t.Fatalf("expected 3 errors (one per door), got %d: %v", len(errs), errs)
	}
	for i, d := range doors {
		if !rel.released(d.ID) {
			t.Fatalf("door %s grant not released on timeout", d.ID)
		}
		if got := doors[i].State(); got != StateStopped {
			t.Fatalf("expected %s stopped after timeout, got %s", d.ID, got)
		}
	}
}
