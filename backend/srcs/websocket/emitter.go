package websocket

import (
	"backend/database"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// SendLogEvent takes a fully‐populated ModuleLog (with CreatedAt, Meta, etc.)
// and builds & dispatches a WebSocket Event.
// If the channel is full, the event is dropped and a warning is logged.
func SendLogEvent(rec database.ModuleLog) {
	ts := rec.CreatedAt.Format(time.RFC3339)

	payloadMap := map[string]any{
		"level":   rec.Level,
		"message": rec.Message,
		"meta":    rec.Meta,
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("%s [ERROR] failed to marshal WS payload: %v", ts, err)
		return
	}

	evt := Event{
		EventType: "log",
		ModuleID:  rec.ModuleID,
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}

	select {
	case Events <- evt:
		// Event dispatched, nothing to do
	default:
		fmt.Printf("%s [WARN] WS event channel full, dropped log for module %q\n",
			ts, rec.ModuleID)
	}
}

// SendModuleStatusChangedEvent builds and dispatches a WebSocket event when a module changed status.
// If the channel is full, the event is dropped and a warning is logged.
func SendModuleStatusChangedEvent(moduleID, moduleName, newStatus string) {
	ts := time.Now().Format(time.RFC3339)

	payloadMap := map[string]any{
		"module_id":   moduleID,
		"module_name": moduleName,
		"new_status":  newStatus,
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("%s [ERROR] failed to marshal module_status_changed payload: %v", ts, err)
		return
	}

	evt := Event{
		EventType: "module_status_changed",
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}

	select {
	case Events <- evt:
	default:
		log.Printf("%s [WARN] WS event channel full, dropped module status event for %q\n", ts, moduleID)
	}
}
