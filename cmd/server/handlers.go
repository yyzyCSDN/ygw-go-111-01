package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"platformscreendoor/internal/console"
	"platformscreendoor/internal/train"
)

type doorOpenRequest struct {
	DoorID  string `json:"doorID"`
	TrainID string `json:"trainID"`
}

type doorIDRequest struct {
	DoorID string `json:"doorID"`
}

type localRequest struct {
	DoorID string `json:"doorID"`
	Local  bool   `json:"local"`
}

type trainDockRequest struct {
	TrainID string         `json:"trainID"`
	LineID  string         `json:"lineID"`
	Aligned bool           `json:"aligned"`
	DoorMap map[int]string `json:"doorMap"`
}

type heartbeatRequest struct {
	DoorID string `json:"doorID"`
}

type batchCloseRequest struct {
	DoorIDs []string `json:"doorIDs"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func newRouter(svc *console.Service, cfg config, probe *probe, planner *train.Planner) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/console.html")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		issues := probe.SelfCheck(r.Context())
		if len(issues) > 0 {
			writeError(w, http.StatusServiceUnavailable, issues[0])
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/doors", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.DoorViews())
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		after := uint64(0)
		limit := 200
		query := r.URL.Query()
		if value := query.Get("after"); value != "" {
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err == nil {
				after = parsed
			}
		}
		if value := query.Get("limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if query.Get("resume") == "true" {
			writeJSON(w, http.StatusOK, svc.Resume(limit))
			return
		}
		if value := query.Get("door"); value != "" {
			writeJSON(w, http.StatusOK, svc.DoorEvents(value, after, limit))
			return
		}
		writeJSON(w, http.StatusOK, svc.Events(after, limit))
	})
	mux.HandleFunc("/api/alarms", func(w http.ResponseWriter, r *http.Request) {
		if doorID := r.URL.Query().Get("door"); doorID != "" {
			writeJSON(w, http.StatusOK, svc.AlarmHistory(doorID))
			return
		}
		writeJSON(w, http.StatusOK, svc.AlarmViews())
	})
	mux.HandleFunc("/api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.Snapshot())
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.Stats())
	})
	mux.HandleFunc("/api/lines", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"lines": planner.Lines(),
			"count": planner.LineCount(),
		})
	})
	mux.HandleFunc("/api/commands", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.RecentCommands(50))
	})
	mux.HandleFunc("/api/batch/close", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req batchCloseRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		errs := svc.CloseBatch(r.Context(), req.DoorIDs)
		writeJSON(w, http.StatusOK, map[string]interface{}{"errors": errs})
	})
	mux.HandleFunc("/api/batch/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req batchCloseRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		errs := svc.ResetAll(req.DoorIDs)
		writeJSON(w, http.StatusOK, map[string]interface{}{"errors": errs})
	})
	mux.HandleFunc("/api/door/status", func(w http.ResponseWriter, r *http.Request) {
		doorID := r.URL.Query().Get("id")
		if doorID == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}
		view, ok := svc.DoorView(doorID)
		if !ok {
			writeError(w, http.StatusNotFound, "door not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("/api/door/open", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req doorOpenRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.OpenDoor(req.DoorID, req.TrainID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"state": "opening"})
	})
	mux.HandleFunc("/api/door/confirm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req doorIDRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.ConfirmOpen(req.DoorID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"state": "open"})
	})
	mux.HandleFunc("/api/door/close", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req doorIDRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.CloseDoor(r.Context(), req.DoorID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"state": "closed"})
	})
	mux.HandleFunc("/api/door/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req doorIDRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.ResetDoor(req.DoorID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"state": "closed"})
	})
	mux.HandleFunc("/api/console/local", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req localRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.SetLocal(req.DoorID, req.Local); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req heartbeatRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		svc.Heartbeat(req.DoorID, time.Now())
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/train/dock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req trainDockRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !planner.HasLine(req.LineID) {
			writeError(w, http.StatusBadRequest, "unknown line")
			return
		}
		t := train.New(req.TrainID, req.LineID)
		t.Docked = true
		if len(req.DoorMap) == 0 {
			t.DoorMap = planner.Plan(req.LineID)
		} else {
			for index, doorID := range req.DoorMap {
				t.MapDoor(index, doorID)
			}
		}
		svc.TrainDocked(t, req.Aligned)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/train/leave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req doorIDRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		svc.TrainLeave(req.DoorID)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	return mux
}
