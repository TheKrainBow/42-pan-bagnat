package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	// map of conn → set of subscribed modules
	subs   = make(map[*websocket.Conn]map[string]bool)
	subsMu sync.Mutex

	// Events is the channel your webhook pushes into
	Events = make(chan Event)
)

// RegisterConn adds a new WS connection.
func RegisterConn(conn *websocket.Conn) {
	subsMu.Lock()
	subs[conn] = make(map[string]bool)
	subsMu.Unlock()
}

// UnregisterConn removes a WS connection.
func UnregisterConn(conn *websocket.Conn) {
	subsMu.Lock()
	delete(subs, conn)
	subsMu.Unlock()
}

// Subscribe adds module to conn’s interest set.
func Subscribe(conn *websocket.Conn, module string) {
	subsMu.Lock()
	subs[conn][module] = true
	subsMu.Unlock()
}

// Unsubscribe removes module from conn’s interest set.
func Unsubscribe(conn *websocket.Conn, module string) {
	subsMu.Lock()
	delete(subs[conn], module)
	subsMu.Unlock()
}

// Dispatch runs in a goroutine and fans out incoming Events to interested clients.
func Dispatch() {
	for evt := range Events {
		msg, _ := json.Marshal(evt)
		subsMu.Lock()
		for conn, modules := range subs {
			if evt.EventType != "log" || modules[evt.ModuleID] {
				conn.WriteMessage(websocket.TextMessage, msg)
			}
		}
		subsMu.Unlock()
	}
}
