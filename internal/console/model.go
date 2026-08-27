package console

import (
	"time"

	"platformscreendoor/internal/event"
)

type DoorView struct {
	ID       string
	Side     int
	Index    int
	State    string
	Locked   bool
	Local    bool
	Online   bool
	Obstacle bool
}

type EventView struct {
	Seq    uint64
	DoorID string
	Type   event.Type
	Detail string
	At     time.Time
}

type AlarmView struct {
	DoorID string
	Kind   string
	Level  string
	Seq    uint64
	At     time.Time
}

type Snapshot struct {
	Doors  []DoorView
	Alarms []AlarmView
	Events []EventView
	Latest uint64
}

type Command struct {
	Seq    uint64
	Kind   string
	DoorID string
	Detail string
	At     time.Time
}
