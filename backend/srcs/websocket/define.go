package websocket

import (
    "encoding/json"
    "os"
)

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

// Secret is the HMAC secret used to verify webhook signatures.
// It is populated from the WEBHOOK_SECRET environment variable at startup.
var Secret []byte

func init() {
    if s := os.Getenv("WEBHOOK_SECRET"); s != "" {
        Secret = []byte(s)
    }
}

// Lightweight container payload for WS updates
type ContainerPayload struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Reason string `json:"reason"`
    Since  string `json:"since"`
}
