package main

import (
	"context"
	"time"

	"platformscreendoor/internal/alarm"
	"platformscreendoor/internal/console"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/record"
)

type probe struct {
	svc   *console.Service
	rec   *record.Recorder
	bus   *event.Bus
	store *door.Store
}

func newProbe(svc *console.Service, rec *record.Recorder, bus *event.Bus, store *door.Store) *probe {
	return &probe{svc: svc, rec: rec, bus: bus, store: store}
}

func (p *probe) SelfCheck(ctx context.Context) []string {
	issues := make([]string, 0)
	if p.store.Count() == 0 {
		issues = append(issues, "no platform doors registered")
	}
	if p.rec.OpenCount() > 128 {
		issues = append(issues, "recorder handle count too high")
	}
	if p.bus.Subscribers() == 0 {
		issues = append(issues, "event bus has no subscribers")
	}
	if len(p.svc.DoorViews()) == 0 {
		issues = append(issues, "console has no door views")
	}
	select {
	case <-ctx.Done():
		issues = append(issues, "health check cancelled")
	default:
	}
	return issues
}

func (p *probe) Run(ctx context.Context, interval time.Duration, svc *console.Service, alarms *alarm.Manager) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for _, view := range svc.DoorViews() {
				offline := !view.Online || alarms.Offline(view.ID, now, door.DefaultHeartbeatTTL, alarms.DefaultJitter())
				if offline {
					_ = alarms.Raise(view.ID, "offline")
				}
			}
		}
	}
}
