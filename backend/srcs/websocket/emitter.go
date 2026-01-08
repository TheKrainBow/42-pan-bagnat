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
		"id":      rec.ID,
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
		Topic:     "module:" + rec.ModuleID,
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}

	// Reliability over drop: block until queued
	Events <- evt
}

// SendContainerLogEvent builds and dispatches a WebSocket event for a container log line.
// moduleID is used for topic scoping; containerName labels the source.
func SendContainerLogEvent(moduleID, containerName, timestamp, message string) {
	payloadMap := map[string]any{
		"container": containerName,
		"message":   message,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("%s [ERROR] failed to marshal container_log payload: %v", timestamp, err)
		return
	}
	evt := Event{
		EventType: "container_log",
		ModuleID:  moduleID,
		Topic:     "container:" + moduleID + ":" + containerName,
		Timestamp: timestamp,
		Payload:   json.RawMessage(payloadBytes),
	}
	select {
	case Events <- evt:
	default:
		fmt.Printf("%s [WARN] WS event channel full, dropped container log for %q/%q\n", timestamp, moduleID, containerName)
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

	Events <- evt
}

// SendModuleDeletedEvent builds and dispatches a WebSocket event when a module is deleted.
// If the channel is full, the event is dropped and a warning is logged.
func SendModuleDeletedEvent(moduleID, moduleName string) {
	ts := time.Now().Format(time.RFC3339)

	payloadMap := map[string]any{
		"module_id":   moduleID,
		"module_name": moduleName,
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("%s [ERROR] failed to marshal module_deleted payload: %v", ts, err)
		return
	}

	evt := Event{
		EventType: "module_deleted",
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}

	Events <- evt
}

// SendContainersUpdatedEvent notifies clients that the container list for a module changed.
func SendContainersUpdatedEvent(moduleID string, containers []ContainerPayload) {
	ts := time.Now().Format(time.RFC3339)
	payloadBytes, err := json.Marshal(containers)
	if err != nil {
		log.Printf("%s [ERROR] failed to marshal containers_updated payload: %v", ts, err)
		return
	}
	evt := Event{
		EventType: "containers_updated",
		ModuleID:  moduleID,
		Topic:     "containers:" + moduleID,
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}
	Events <- evt
}

// SendModuleDeployStatus notifies clients of a module's deployment state in real-time.
// Topic: "module:<moduleID>"
// Payload: { module_id, is_deploying, last_deploy_status, last_deploy }
func SendModuleDeployStatus(moduleID string, isDeploying bool, lastDeployStatus string, lastDeploy string) {
	ts := time.Now().Format(time.RFC3339)
	payloadMap := map[string]any{
		"module_id":          moduleID,
		"is_deploying":       isDeploying,
		"last_deploy_status": lastDeployStatus,
		"last_deploy":        lastDeploy,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return
	}
	evt := Event{
		EventType: "module_deploy_status",
		ModuleID:  moduleID,
		Topic:     "module:" + moduleID,
		Timestamp: ts,
		Payload:   json.RawMessage(payloadBytes),
	}
	Events <- evt
}

// SendGenericModuleEvent sends a generic event with a payload map on topic module:<moduleID>.
func SendGenericModuleEvent(moduleID string, eventType string, payload map[string]any) {
	ts := time.Now().Format(time.RFC3339)
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	evt := Event{
		EventType: eventType,
		ModuleID:  moduleID,
		Topic:     "module:" + moduleID,
		Timestamp: ts,
		Payload:   json.RawMessage(b),
	}
	Events <- evt
}
