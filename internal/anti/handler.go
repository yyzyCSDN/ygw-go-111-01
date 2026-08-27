package anti

type Handler struct {
	monitor *Monitor
}

func NewHandler(monitor *Monitor) *Handler {
	return &Handler{monitor: monitor}
}

func (h *Handler) ReleaseAll(doorIDs []string) {
	for _, doorID := range doorIDs {
		h.monitor.Release(doorID)
	}
}

func (h *Handler) ClearTrap(doorID string) {
	h.monitor.Release(doorID)
}

func (h *Handler) SetTrap(doorID string, trapped bool) {
	h.monitor.SetTrap(doorID, trapped)
}
