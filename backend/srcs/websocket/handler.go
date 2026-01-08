package websocket

import (
	"backend/database"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authenticate via session cookie/header before upgrading
		sid := readSessionID(r)
		if sid == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sess, err := database.GetSession(sid)
		if err != nil || sess == nil || sess.ExpiresAt.Before(time.Now()) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WS upgrade failed", http.StatusBadRequest)
			return
		}

		fmt.Printf("WS connected: %s\n", conn.RemoteAddr())
		RegisterConn(conn)
		defer UnregisterConn(conn)

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ctl ControlMessage
			if err := json.Unmarshal(data, &ctl); err == nil {
				fmt.Printf("Received message: %s | %s\n", ctl.ModuleID, ctl.Action)
				switch ctl.Action {
				case ActionSubscribe:
					Subscribe(conn, ctl.ModuleID)
				case ActionUnsubscribe:
					Unsubscribe(conn, ctl.ModuleID)
				default:
					fmt.Printf("BAD ACTION!\n")
				}
			}
		}
	}
}

// readSessionID replicates core.ReadSessionIDFromCookie without importing core to avoid cycles.
func readSessionID(r *http.Request) string {
	if c, err := r.Cookie("session_id"); err == nil && c.Value != "" {
		return c.Value
	}
	if v := r.Header.Get("X-Session-Id"); v != "" {
		return v
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}
