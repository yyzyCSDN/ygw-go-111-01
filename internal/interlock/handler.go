package interlock

import (
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/train"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Evaluate(il *Interlock, d door.View, t train.Train) Result {
	return il.Check(d, t)
}

func (h *Handler) Reason(res Result) string {
	return res.Reason
}
