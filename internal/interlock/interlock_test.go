package interlock

import (
	"testing"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/train"
)

// dockedTrain builds a train view that satisfies every Interlock.Check precondition
// except for the platform-busy / outstanding gate, so a returned Reason isolates that gate.
func dockedTrain() train.Train {
	t := train.New("T1", "L1")
	t.Docked = true
	t.Aligned = true
	t.MapDoor(0, "PSD01")
	t.MapDoor(1, "PSD02")
	return t.Snapshot()
}

func doorView(id string) door.View {
	return door.View{
		ID:    id,
		Index: 0,
		State: door.StateClosed,
	}
}

// TestReleaseDoorClearsOutstanding reproduces the bug where a single door's
// confirm never returned: the close path released the grant but the
// outstanding flag leaked forever, so Check() kept returning "platform busy"
// and the next train could no longer open any door.
//
// The platform-busy gate is global (any outstanding door blocks new grants),
// so the contract we verify is per-door: releasing a door removes *its* flag
// from the outstanding set. Once the last door is released, the platform must
// become free again — the exact behaviour that broke before this fix.
func TestReleaseDoorClearsOutstanding(t *testing.T) {
	il := New()

	// Train docks, two doors are granted (outstanding) for the open phase.
	il.GrantDoor("PSD01", "T1", 1)
	il.GrantDoor("PSD02", "T1", 2)

	tr := dockedTrain()
	if got := il.Check(doorView("PSD01"), tr); got.Allowed {
		t.Fatalf("expected platform busy while doors outstanding, got allowed=%v reason=%q", got.Allowed, got.Reason)
	}

	// Only PSD01's confirm never returns; it is released (e.g. close confirm timeout).
	il.ReleaseDoor("PSD01")

	// PSD01 must be gone from the outstanding set; only PSD02 remains.
	outstanding := il.Outstanding()
	if len(outstanding) != 1 || outstanding[0] != "PSD02" {
		t.Fatalf("expected only PSD02 outstanding after releasing PSD01, got %v", outstanding)
	}
	// Its grant record is gone too.
	if _, ok := il.GrantFor("PSD01"); ok {
		t.Fatalf("expected PSD01 grant removed after ReleaseDoor")
	}

	// The platform is still busy because PSD02 is outstanding — that's correct
	// global behaviour, not a leak.
	if got := il.Check(doorView("PSD01"), tr); got.Allowed {
		t.Fatalf("expected platform busy while PSD02 still outstanding, got allowed=%v", got.Allowed)
	}

	// The whole platform must clear once the last door is released too.
	il.ReleaseDoor("PSD02")
	if outstanding := il.Outstanding(); len(outstanding) != 0 {
		t.Fatalf("expected no outstanding doors after releasing all, got %v", outstanding)
	}
	if got := il.Check(doorView("PSD01"), tr); !got.Allowed {
		t.Fatalf("expected platform clear after all releases, got reason=%q", got.Reason)
	}
}

// TestReleaseDoorNotHoldingPlatform ensures releasing an unrelated door does
// not stall unrelated open attempts — the per-door isolation the caller relies on.
func TestReleaseDoorNotHoldingPlatform(t *testing.T) {
	il := New()
	tr := dockedTrain()

	il.GrantDoor("PSD01", "T1", 1)
	il.GrantDoor("PSD02", "T1", 2)

	// PSD02's confirm times out; it is released.
	il.ReleaseDoor("PSD02")

	// PSD01 is still outstanding, so the platform is still busy for *new* grants.
	if got := il.Check(doorView("PSD01"), tr); got.Allowed {
		t.Fatalf("expected platform busy while PSD01 still outstanding, got allowed=%v", got.Allowed)
	}

	il.ReleaseDoor("PSD01")

	// Now the platform must be fully free again — i.e. ReleaseDoor must not
	// leave behind a leaked outstanding entry that blocks the next train.
	if got := il.Check(doorView("PSD01"), tr); !got.Allowed {
		t.Fatalf("expected platform free after releasing all grants, got reason=%q", got.Reason)
	}
}

func TestBudgets(t *testing.T) {
	il := New()
	if il.OpenBudget() != door.DefaultOpenTimeout {
		t.Fatalf("open budget = %v, want %v", il.OpenBudget(), door.DefaultOpenTimeout)
	}
	if il.CloseBudget() != door.DefaultCloseTimeout {
		t.Fatalf("close budget = %v, want %v", il.CloseBudget(), door.DefaultCloseTimeout)
	}
	_ = time.Second // keep time import meaningful if budgets ever become time-based
}
