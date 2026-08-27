package internal_test

import (
	"errors"
	"sync"
	"testing"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/record"
)

func TestBug08_Handleleak(t *testing.T) {
	records := record.NewStore(100)
	rec := record.NewRecorder(nil, t.TempDir(), records)
	d := door.New("PSD08", 1, 7, door.Config{})
	act := &fakeActuator{driveErr: errors.New("motor failure"), confirm: true}
	if err := d.Open(act, nil, rec); err == nil {
		t.Fatal("open should fail")
	}
	if got := rec.OpenCount(); got != 0 {
		t.Fatalf("recorder still holds %d handles after failed open", got)
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
