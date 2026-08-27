package interlock

import (
	"sync"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/train"
)

type Interlock struct {
	mu     sync.Mutex
	grants map[string]door.Grant
	chain  bool
}

func New() *Interlock {
	return &Interlock{
		grants: make(map[string]door.Grant),
	}
}

func (il *Interlock) OpenBudget() time.Duration {
	return door.DefaultOpenTimeout
}

func (il *Interlock) CloseBudget() time.Duration {
	return door.DefaultCloseTimeout
}

func (il *Interlock) Check(d door.View, t train.Train) Result {
	il.mu.Lock()
	defer il.mu.Unlock()
	budget := il.OpenBudget()
	if _, ok := il.grants[d.ID]; ok {
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

// GrantDoor records an open grant for the given door only. Each door carries
// its own grant so that confirming or releasing one door never touches another
// door's state.
func (il *Interlock) GrantDoor(doorID string, trainID string, seq uint64) door.Grant {
	il.mu.Lock()
	defer il.mu.Unlock()
	g := door.Grant{DoorID: doorID, TrainID: trainID, Seq: seq}
	il.grants[doorID] = g
	return g
}

// GrantFor returns the grant for the requested door, and only that door.
func (il *Interlock) GrantFor(doorID string) (door.Grant, bool) {
	il.mu.Lock()
	defer il.mu.Unlock()
	g, ok := il.grants[doorID]
	return g, ok
}

func (il *Interlock) ReleaseDoor(doorID string) {
	il.mu.Lock()
	defer il.mu.Unlock()
	delete(il.grants, doorID)
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
	out := make([]string, 0, len(il.grants))
	for id := range il.grants {
		out = append(out, id)
	}
	return out
}
