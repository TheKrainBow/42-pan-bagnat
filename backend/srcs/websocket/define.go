package websocket

import "encoding/json"

// Event is what your modules POST via the webhook.
type Event struct {
	EventType string          `json:"eventType"`
	ModuleID  string          `json:"module_id,omitempty"`
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
