package alarm

import "platformscreendoor/internal/door"

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) Evaluate(d door.View, online bool) []Alarm {
	out := make([]Alarm, 0)
	if !online {
		_ = h.manager.Raise(d.ID, "offline")
		out = append(out, Alarm{DoorID: d.ID, Kind: "offline", Level: LevelWarn})
	}
	if d.Obstacle {
		_ = h.manager.Raise(d.ID, "trap")
		out = append(out, Alarm{DoorID: d.ID, Kind: "trap", Level: LevelCritical})
	}
	return out
}

func (h *Handler) Manager() *Manager {
	return h.manager
}
