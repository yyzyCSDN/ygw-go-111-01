package internal_test

import (
	"fmt"
	"sync"
	"testing"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/interlock"
)

func TestBug11_ConcurrentDoorGrant(t *testing.T) {
	il := interlock.New()
	d := door.New("PSD11", 1, 10, door.Config{})
	const workers = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_ = il.GrantDoor("PSD11", fmt.Sprintf("T%d", i), uint64(i+1))
		}(i)
	}
	close(start)
	wg.Wait()
	g, ok := il.GrantFor("PSD11")
	if !ok {
		t.Fatal("grant missing after concurrent grants")
	}
	if g.DoorID != "PSD11" {
		t.Fatalf("grant door = %s, want PSD11", g.DoorID)
	}
	if err := d.ApplyGrant(g); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmGrant(g); err != nil {
		t.Fatal(err)
	}
	if d.State() != door.StateOpen {
		t.Fatalf("door state = %v, want open", d.State())
	}
	for i := 0; i < workers; i++ {
		if _, ok := il.GrantFor("PSD11"); !ok {
			t.Fatal("grant lost during concurrent read")
		}
	}
}
