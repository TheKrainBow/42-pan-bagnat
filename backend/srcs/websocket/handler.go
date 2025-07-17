package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WS upgrade failed", http.StatusBadRequest)
			return
		}

		fmt.Printf("WS connected: %s", conn.RemoteAddr())
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
