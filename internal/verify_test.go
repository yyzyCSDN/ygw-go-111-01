package internal_test

import (
	"testing"

	"platformscreendoor/internal/console"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/interlock"
	"platformscreendoor/internal/record"
	"platformscreendoor/internal/train"
)

func TestBug04_Muxcrosstalk(t *testing.T) {
	store := door.NewStore()
	d4 := door.New("PSD04", 1, 3, door.Config{})
	d6 := door.New("PSD06", 1, 5, door.Config{})
	store.AddAll(d4, d6)
	il := interlock.New()
	bus := event.New()
	records := record.NewStore(100)
	svc := console.NewService(console.Deps{
		Store:     store,
		Interlock: il,
		Records:   records,
		Bus:       bus,
	})
	tr := train.New("T4", "L1")
	tr.Docked = true
	tr.Aligned = true
	tr.DoorMap = map[int]string{3: "PSD04", 5: "PSD06"}
	svc.TrainDocked(tr, true)
	if err := svc.OpenDoor("PSD04", "T4"); err != nil {
		t.Fatal(err)
	}
	if err := svc.OpenDoor("PSD06", "T4"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmOpen("PSD04"); err != nil {
		t.Fatal(err)
	}
	if d4.State() != door.StateOpen {
		t.Fatalf("PSD04 state = %v, want open", d4.State())
	}
	if d6.State() != door.StateOpening {
		t.Fatalf("PSD06 state = %v, want opening (crosstalk)", d6.State())
	}
}
