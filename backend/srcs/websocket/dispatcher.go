package websocket

import (
    "encoding/json"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var (
    // map of conn → set of subscribed topics
    subs   = make(map[*websocket.Conn]map[string]bool)
    subsMu sync.Mutex

    // Events is the channel your webhook pushes into. Buffered to avoid drops.
    Events = make(chan Event, 10000)

    // Optional hooks to react to subscribe/unsubscribe with current topic count
    OnSubscribe   func(topic string, count int)
    OnUnsubscribe func(topic string, count int)
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
    // capture topics to notify after removal
    topics := subs[conn]
    delete(subs, conn)
    // compute counts and call hook
    if OnUnsubscribe != nil {
        for topic := range topics {
            cnt := 0
            for _, t := range subs {
                if t[topic] { cnt++ }
            }
            OnUnsubscribe(topic, cnt)
        }
    }
    subsMu.Unlock()
}

// Subscribe adds module to conn’s interest set.
func Subscribe(conn *websocket.Conn, module string) {
    subsMu.Lock()
    subs[conn][module] = true
    // count subscribers for this topic
    cnt := 0
    for _, topics := range subs {
        if topics[module] {
            cnt++
        }
    }
    if OnSubscribe != nil {
        OnSubscribe(module, cnt)
    }
    subsMu.Unlock()
}

// Unsubscribe removes module from conn’s interest set.
func Unsubscribe(conn *websocket.Conn, module string) {
    subsMu.Lock()
    delete(subs[conn], module)
    // count remaining subscribers for this topic
    cnt := 0
    for _, topics := range subs {
        if topics[module] {
            cnt++
        }
    }
    if OnUnsubscribe != nil {
        OnUnsubscribe(module, cnt)
    }
    subsMu.Unlock()
}

// Dispatch runs in a goroutine and fans out incoming Events to interested clients.
func Dispatch() {
    for evt := range Events {
        msg, _ := json.Marshal(evt)
        subsMu.Lock()
        for conn, modules := range subs {
            // If Topic is empty, broadcast. Otherwise only to subscribers of that topic.
            if evt.Topic == "" || modules[evt.Topic] {
                // avoid blocking indefinitely on slow clients
                _ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
                _ = conn.WriteMessage(websocket.TextMessage, msg)
            }
        }
        subsMu.Unlock()
    }
}
