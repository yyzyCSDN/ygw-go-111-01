package record

import (
	"time"

	"platformscreendoor/internal/event"
)

type Record struct {
	Fingerprint uint64
	Seq         uint64
	DoorID      string
	Type        event.Type
	Detail      string
	At          time.Time
}

func FromEvent(e event.Event) Record {
	return Record{
		Fingerprint: Fingerprint(e.DoorID, string(e.Type), e.Detail),
		Seq:         e.Seq,
		DoorID:      e.DoorID,
		Type:        e.Type,
		Detail:      e.Detail,
		At:          e.At,
	}
}
