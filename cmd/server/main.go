package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"platformscreendoor/internal/alarm"
	"platformscreendoor/internal/anti"
	"platformscreendoor/internal/console"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/interlock"
	"platformscreendoor/internal/record"
	"platformscreendoor/internal/train"
)

func main() {
	cfg := parseConfig()
	if err := os.MkdirAll(cfg.dataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	bus := event.New()
	records := record.NewStore(5000)
	recorder := record.NewRecorder(record.NewFileFactory(cfg.dataDir), cfg.dataDir, records)
	store := door.NewStore()
	doorCfg := door.Config{
		OpenTimeout:     cfg.openTimeout,
		CloseTimeout:    cfg.closeTimeout,
		TrapInterval:    cfg.trapInterval,
		HeartbeatTTL:    cfg.heartbeatTTL,
		HeartbeatJitter: cfg.heartbeatJitter,
	}
	doors := door.NewDoors(cfg.doors, doorCfg)
	store.AddAll(doors...)
	doorIDs := make([]string, 0, len(doors))
	for _, d := range doors {
		doorIDs = append(doorIDs, d.ID)
	}
	planner := train.NewPlanner()
	planner.RegisterLine("L1", doorIDs)
	dispatcher := alarm.NewDispatcher(bus)
	alarms := alarm.NewManager(recorder, dispatcher, alarm.DefaultPolicy())
	alarmHandler := alarm.NewHandler(alarms)
	monitor := anti.NewMonitor(cfg.trapInterval, alarms)
	antiHandler := anti.NewHandler(monitor)
	inter := interlock.New()
	trains := train.NewController(bus)
	actuator := console.NewSimActuator(80 * time.Millisecond)
	observer := door.NewFanout(alarms)
	svc := console.NewService(console.Deps{
		Store:        store,
		Interlock:    inter,
		Alarms:       alarms,
		AlarmHandler: alarmHandler,
		Monitor:      monitor,
		AntiHandler:  antiHandler,
		Recorder:     recorder,
		Records:      records,
		Bus:          bus,
		Actuator:     actuator,
		Sampler:      monitor,
		Trains:       trains,
		Observer:     observer,
	})
	probe := newProbe(svc, recorder, bus, store)
	go probe.Run(context.Background(), 2*time.Second, svc, alarms)
	sub := bus.Subscribe()
	go func() {
		for {
			events := bus.Drain(sub, 16)
			if len(events) == 0 {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			for _, e := range events {
				log.Printf("event seq=%d door=%s type=%s detail=%s", e.Seq, e.DoorID, e.Type, e.Detail)
			}
		}
	}()
	handler := newRouter(svc, cfg, probe, planner)
	log.Printf("platform screen door service listening on %s", cfg.addr)
	if err := http.ListenAndServe(cfg.addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}
