package internal_test

import (
	"testing"

	"platformscreendoor/internal/alarm"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/record"
)

func TestBug09_Alarmsuppressed(t *testing.T) {
	bus := event.New()
	records := record.NewStore(50)
	rec := record.NewRecorder(nil, t.TempDir(), records)
	dispatch := alarm.NewDispatcher(bus)
	alarms := alarm.NewManager(rec, dispatch, alarm.DefaultPolicy())
	d := door.New("PSD09", 1, 8, door.Config{})
	if err := alarms.Raise("PSD09", "trap"); err != nil {
		t.Fatal(err)
	}
	d.Stop()
	if err := d.Reset(alarms); err != nil {
		t.Fatal(err)
	}
	if err := alarms.Raise("PSD09", "trap"); err != nil {
		t.Fatal(err)
	}
	if got := alarms.Count("trap"); got != 2 {
		t.Fatalf("trap alarm count = %d, want 2 (suppression not cleared)", got)
	}
}
