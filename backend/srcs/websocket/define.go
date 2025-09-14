package websocket

import "encoding/json"

// Event is what your modules POST via the webhook.
type Event struct {
    EventType string          `json:"eventType"`
    ModuleID  string          `json:"module_id,omitempty"`
    // Topic is a routing key like "module:<id>" or "container:<id>:<name>"
    Topic     string          `json:"topic,omitempty"`
    Timestamp string          `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
}

type ActionType string

const (
	ActionSubscribe   ActionType = "subscribe"
	ActionUnsubscribe ActionType = "unsubscribe"
)

// ControlMessage is the JSON shape clients send over WS
// to subscribe/unsubscribe to a module.
type ControlMessage struct {
    Action   ActionType `json:"action"`
    ModuleID string     `json:"module_id"`
}

// For convenience, allow marshalling a ControlMessage.
func (c ControlMessage) Bytes() []byte {
	b, _ := json.Marshal(c)
	return b
}

var Secret []byte = []byte("MokkoIsNotFat")

// Lightweight container payload for WS updates
type ContainerPayload struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Reason string `json:"reason"`
    Since  string `json:"since"`
}
