package internal_test

import (
	"testing"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/interlock"
)

func TestBug01_Missingdefault(t *testing.T) {
	door.DefaultOpenTimeout = 60 * time.Millisecond
	defer func() {
		door.DefaultOpenTimeout = 6 * time.Second
	}()
	d := door.New("PSD01", 1, 0, door.Config{})
	if got := door.OpenTimeoutFor(d.Config); got != 60*time.Millisecond {
		t.Fatalf("missing config open timeout = %v, want shared default", got)
	}
	il := interlock.New()
	if got := il.OpenBudget(); got != 60*time.Millisecond {
		t.Fatalf("interlock open budget = %v, want shared default", got)
	}
}
