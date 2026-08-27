package internal_test

import (
	"testing"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
)

func TestBug07_Eventorder(t *testing.T) {
	d := door.New("PSD07", 1, 6, door.Config{})
	if err := d.ApplyEvent(event.Event{Seq: 10, DoorID: "PSD07", Type: event.TypeOpen}); err != nil {
		t.Fatal(err)
	}
	if d.State() != door.StateOpen {
		t.Fatalf("door state = %v, want open", d.State())
	}
	if err := d.ApplyEvent(event.Event{Seq: 9, DoorID: "PSD07", Type: event.TypeClose}); err != nil {
		t.Fatal(err)
	}
	if d.State() != door.StateOpen {
		t.Fatalf("stale close event overrode newer state: %v", d.State())
	}
	bus := event.New()
	items := []event.Event{
		{Seq: 12, DoorID: "PSD07", Type: event.TypeOpen},
		{Seq: 11, DoorID: "PSD07", Type: event.TypeClose},
	}
	sorted := bus.Ordered(items)
	if len(sorted) != 2 || sorted[0].Seq != 11 || sorted[1].Seq != 12 {
		t.Fatal("bus did not order events by sequence")
	}
}
