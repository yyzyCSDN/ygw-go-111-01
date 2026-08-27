package interlock

import (
	"sync"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/train"
)

type Interlock struct {
	mu          sync.Mutex
	grants      map[string]door.Grant
	outstanding map[string]bool
	chain       bool
}

func New() *Interlock {
	return &Interlock{
		grants:      make(map[string]door.Grant),
		outstanding: make(map[string]bool),
	}
}

func (il *Interlock) OpenBudget() time.Duration {
	return door.DefaultOpenTimeout
}

func (il *Interlock) CloseBudget() time.Duration {
	return door.DefaultCloseTimeout
}

// checkLocked runs the interlock preconditions. The caller must hold il.mu.
// Factored out so the validation and the grant write can run under a single
// lock in AcquireGrant, closing the check-then-grant race.
func (il *Interlock) checkLocked(d door.View, t train.Train) Result {
	budget := il.OpenBudget()
	if il.outstanding[d.ID] {
		return Result{Allowed: false, Reason: "door busy", Budget: budget}
	}
	if !t.Docked {
		return Result{Allowed: false, Reason: "train not docked", Budget: budget}
	}
	if !t.Aligned {
		return Result{Allowed: false, Reason: "train not aligned", Budget: budget}
	}
	target, ok := t.DoorFor(d.Index)
	if !ok || target != d.ID {
		return Result{Allowed: false, Reason: "door mapping mismatch", Budget: budget}
	}
	if d.State != door.StateClosed {
		return Result{Allowed: false, Reason: "door not closed", Budget: budget}
	}
	if d.Obstacle {
		return Result{Allowed: false, Reason: "obstacle present", Budget: budget}
	}
	if d.Local {
		return Result{Allowed: false, Reason: "local control active", Budget: budget}
	}
	return Result{Allowed: true, Reason: "ok", Budget: budget}
}

func (il *Interlock) Check(d door.View, t train.Train) Result {
	il.mu.Lock()
	defer il.mu.Unlock()
	return il.checkLocked(d, t)
}

// AcquireGrant atomically validates the interlock preconditions and grants the
// door to the requesting train. Validation and the grant write happen under a
// single lock, so two near-simultaneous requests for the same door cannot both
// succeed: the second is rejected with "door busy" and the first train's grant
// is never overwritten. A door that has been released (e.g. after a prior train
// closed and departed) may be acquired again.
//
// This is the path used by door-open requests. Without the atomic
// check+grant, a second train arriving while the first train's grant was in
// flight could clobber the recorded grant and leave the door state and the
// grant map pointing at different trains, which surfaced on the console as the
// door flickering between open and closed.
func (il *Interlock) AcquireGrant(d door.View, t train.Train, seq uint64) (door.Grant, Result) {
	il.mu.Lock()
	defer il.mu.Unlock()
	res := il.checkLocked(d, t)
	if !res.Allowed {
		return door.Grant{}, res
	}
	g := door.Grant{DoorID: d.ID, TrainID: t.ID, Seq: seq}
	il.grants[d.ID] = g
	il.outstanding[d.ID] = true
	return g, res
}

func (il *Interlock) GrantDoor(doorID string, trainID string, seq uint64) door.Grant {
	il.mu.Lock()
	defer il.mu.Unlock()
	// Never overwrite an outstanding grant. If the door already holds an
	// unreleased grant (e.g. for a prior train that has not closed yet), return
	// the existing grant instead of clobbering it with a new train's identity.
	if il.outstanding[doorID] {
		if g, ok := il.grants[doorID]; ok {
			return g
		}
		return door.Grant{DoorID: doorID}
	}
	g := door.Grant{DoorID: doorID, TrainID: trainID, Seq: seq}
	il.grants[doorID] = g
	il.outstanding[doorID] = true
	return g
}

func (il *Interlock) GrantFor(doorID string) (door.Grant, bool) {
	g, ok := il.grants[doorID]
	return g, ok
}

func (il *Interlock) ReleaseDoor(doorID string) {
	il.mu.Lock()
	defer il.mu.Unlock()
	delete(il.grants, doorID)
	delete(il.outstanding, doorID)
}

func (il *Interlock) BeginChain() {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.chain = true
}

func (il *Interlock) ReleaseChain() {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.chain = false
}

func (il *Interlock) Snapshot() map[string]door.Grant {
	il.mu.Lock()
	defer il.mu.Unlock()
	out := make(map[string]door.Grant, len(il.grants))
	for id, g := range il.grants {
		out[id] = g
	}
	return out
}

func (il *Interlock) Outstanding() []string {
	il.mu.Lock()
	defer il.mu.Unlock()
	out := make([]string, 0, len(il.outstanding))
	for id := range il.outstanding {
		out = append(out, id)
	}
	return out
}
