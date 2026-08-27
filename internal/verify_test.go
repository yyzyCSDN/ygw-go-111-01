package internal_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/interlock"
	"platformscreendoor/internal/train"
)

func TestBug06_Cascadetimeout(t *testing.T) {
	il := interlock.New()
	act := &fakeActuator{confirmFor: map[string]bool{"PSD06B": true}}
	dA := door.New("PSD06A", 1, 0, door.Config{TrapInterval: 5 * time.Millisecond})
	dB := door.New("PSD06B", 1, 1, door.Config{TrapInterval: 5 * time.Millisecond})
	gA := il.GrantDoor("PSD06A", "T6", 1)
	gB := il.GrantDoor("PSD06B", "T6", 2)
	if err := dA.ApplyGrant(gA); err != nil {
		t.Fatal(err)
	}
	if err := dA.ConfirmGrant(gA); err != nil {
		t.Fatal(err)
	}
	if err := dB.ApplyGrant(gB); err != nil {
		t.Fatal(err)
	}
	if err := dB.ConfirmGrant(gB); err != nil {
		t.Fatal(err)
	}
	sampler := &fakeSampler{trapped: false}
	il.BeginChain()
	errs := door.CloseBatch(context.Background(), []*door.Door{dA, dB}, act, sampler, il, nil, nil, 30*time.Millisecond)
	if len(errs) != 1 {
		t.Fatalf("expected one timeout error, got %d", len(errs))
	}
	if dB.State() != door.StateClosed {
		t.Fatalf("PSD06B state = %v, want closed", dB.State())
	}
	il.ReleaseChain()
	tr := train.New("T6b", "L1")
	tr.Docked = true
	tr.Aligned = true
	tr.DoorMap = map[int]string{1: "PSD06B"}
	if res := il.Check(dB.View(), tr.Snapshot()); !res.Allowed {
		t.Fatalf("check after chain release denied: %s", res.Reason)
	}
}

type fakeActuator struct {
	mu        sync.Mutex
	confirm   bool
	confirmFor map[string]bool
	driveErr  error
	moves     []string
}

func (a *fakeActuator) Drive(id string, open bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.moves = append(a.moves, id)
	return a.driveErr
}

func (a *fakeActuator) Confirm(id string, open bool) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.confirmFor != nil {
		if v, ok := a.confirmFor[id]; ok {
			return v, nil
		}
	}
	return a.confirm, nil
}

type fakeSampler struct {
	trapped bool
}

func (s *fakeSampler) Sample(ctx context.Context, doorID string) bool {
	return s.trapped
}
