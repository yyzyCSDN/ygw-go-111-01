package alarm

import "time"

type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelCritical
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	default:
		return "critical"
	}
}

type Alarm struct {
	DoorID string
	Kind   string
	Level  Level
	Seq    uint64
	At     time.Time
}
