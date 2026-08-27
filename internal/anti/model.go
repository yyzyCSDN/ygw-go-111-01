package anti

import "time"

type Sample struct {
	DoorID  string
	Blocked bool
	At      time.Time
}

type State struct {
	Blocked map[string]bool
	Count   int
}

func NewState() *State {
	return &State{Blocked: make(map[string]bool)}
}
