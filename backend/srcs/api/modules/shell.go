package modules

import (
	"backend/api/auth"
	"backend/core"
	"backend/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

const (
	shellIdleTimeout    = 30 * time.Minute
	shellAdminRecheck   = 15 * time.Second
	shellMaxMessageSize = 64 * 1024
)

// shellUpgrader only accepts same-site origins: the session is a cookie, so a
// permissive origin check would let any website drive a root shell (CSWSH).
var shellUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     shellOriginAllowed,
}

func shellOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.TrimSpace(os.Getenv("HOST_NAME"))
	if host == "" {
		return false
	}
	return strings.EqualFold(u.Hostname(), host) || strings.EqualFold(u.Host, host)
}

type shellControl struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

// GET /admin/modules/{moduleID}/shell (WebSocket)
// Binary frames carry raw terminal input/output, text frames carry JSON control
// messages ({"type":"resize","cols":80,"rows":24}).
// The route lives under /api/v1/admin, so Auth + Blacklist + Admin middlewares
// have already run; the admin role is still re-checked before and during the session.
func ModuleShell(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok || user == nil || !userIsAdmin(r, user) {
		auth.WriteJSONError(w, http.StatusForbidden, "admin_required", "You are not allowed to view this content.")
		return
	}
	if !shellOriginAllowed(r) {
		auth.WriteJSONError(w, http.StatusForbidden, "bad_origin", "Cross-origin shell access is not allowed.")
		return
	}
	mod, err := core.GetModule(chi.URLParam(r, "moduleID"))
	if err != nil || mod.ID == "" {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}

	conn, err := shellUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(shellMaxMessageSize)

	var writeMu sync.Mutex
	send := func(msgType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteMessage(msgType, data)
	}
	fail := func(msg string) {
		_ = send(websocket.TextMessage, []byte(`{"type":"error","message":`+jsonString(msg)+`}`))
		_ = send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}

	sh, err := core.OpenModuleShell(r.Context(), mod, 80, 24)
	if err != nil {
		log.Printf("[shell] %s: open failed for module %s: %v", user.FtLogin, mod.Slug, err)
		if errors.Is(err, core.ErrShellLimitReached) {
			fail("Too many open shells, close one and retry.")
		} else {
			fail("Unable to start the shell.")
		}
		return
	}
	defer sh.Close()
	log.Printf("[shell] %s opened a shell on module %s", user.FtLogin, mod.Slug)
	defer log.Printf("[shell] %s closed the shell on module %s", user.FtLogin, mod.Slug)

	deadline := time.Now().Add(core.ShellMaxLifetime)
	_ = send(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"session","expires_in":%d}`, int(time.Until(deadline).Seconds()))))

	done := make(chan struct{})
	var once sync.Once
	stop := func() { once.Do(func() { close(done); conn.Close() }) }
	defer stop()

	var activityMu sync.Mutex
	lastActivity := time.Now()
	touch := func() {
		activityMu.Lock()
		lastActivity = time.Now()
		activityMu.Unlock()
	}

	// shell -> browser
	go func() {
		defer stop()
		buf := make([]byte, 8192)
		for {
			n, err := sh.Read(buf)
			if n > 0 {
				if werr := send(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Hard session time limit.
	go func() {
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
			fail("Session time limit reached.")
			stop()
		}
	}()

	// Revoke the session as soon as the admin role is lost, or it goes idle.
	go func() {
		t := time.NewTicker(shellAdminRecheck)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if !userIsAdmin(r, user) {
					log.Printf("[shell] %s lost admin role, closing shell on module %s", user.FtLogin, mod.Slug)
					stop()
					return
				}
				activityMu.Lock()
				idle := time.Since(lastActivity)
				activityMu.Unlock()
				if idle > shellIdleTimeout {
					stop()
					return
				}
			}
		}
	}()

	// browser -> shell
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		touch()
		switch msgType {
		case websocket.BinaryMessage:
			if _, err := sh.Write(data); err != nil {
				return
			}
		case websocket.TextMessage:
			var ctl shellControl
			if json.Unmarshal(data, &ctl) == nil && ctl.Type == "resize" && ctl.Cols > 0 && ctl.Rows > 0 && ctl.Cols < 1000 && ctl.Rows < 1000 {
				_ = sh.Resize(ctl.Cols, ctl.Rows)
			}
		}
	}
}

func userIsAdmin(r *http.Request, user *core.User) bool {
	isAdmin, err := database.UserHasRoleByID(context.Background(), user.ID, core.RoleIDAdmin)
	return err == nil && isAdmin
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
