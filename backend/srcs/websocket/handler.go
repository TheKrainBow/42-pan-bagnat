package websocket

import (
	"backend/database"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// roleIDAdmin mirrors core.RoleIDAdmin; duplicated here to avoid an import cycle
// (core imports this package to stream container logs).
const roleIDAdmin = "roles_admin"

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
					// All current topics ("module:", "container:", "containers:") carry
					// module-management data that is admin-only over the REST API
					// (see /api/v1/admin/modules/...), so gate every subscribe the same way.
					if isAdminOnlyTopic(ctl.ModuleID) && !isAdminSession(sess) {
						fmt.Printf("WS subscribe denied (not admin): %s\n", ctl.ModuleID)
						continue
					}
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

// isAdminOnlyTopic reports whether a topic exposes module-management data that
// is admin-only over the REST API (module status/logs, container status/logs).
func isAdminOnlyTopic(topic string) bool {
	return strings.HasPrefix(topic, "module:") ||
		strings.HasPrefix(topic, "container:") ||
		strings.HasPrefix(topic, "containers:")
}

// isAdminSession reports whether the session's user currently holds the admin role.
func isAdminSession(sess *database.Session) bool {
	user, err := database.GetUserByLogin(sess.Login)
	if err != nil || user == nil {
		return false
	}
	isAdmin, err := database.UserHasRoleByID(context.Background(), user.ID, roleIDAdmin)
	if err != nil {
		return false
	}
	return isAdmin
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
