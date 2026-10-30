package modules

import (
	"backend/api/auth"
	"backend/core"
	"backend/database"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

const (
	claudePingInterval = 30 * time.Second
	claudeAdminRecheck = 15 * time.Second
)

type claudeSessionsResponse struct {
	Sessions []database.ClaudeSession `json:"sessions"`
	// Occupant is whoever currently holds the module's Claude container (maybe the caller).
	Occupant *core.ClaudeOccupant `json:"occupant"`
	Me       string               `json:"me"`
}

type postClaudeSessionRequest struct {
	Resume string `json:"resume"`
}

func claudeUser(w http.ResponseWriter, r *http.Request) (*core.User, bool) {
	user, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok || user == nil {
		auth.WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "You must be logged in.")
		return nil, false
	}
	return user, true
}

func claudeModule(w http.ResponseWriter, r *http.Request) (core.Module, bool) {
	mod, err := core.GetModule(chi.URLParam(r, "moduleID"))
	if err != nil || mod.ID == "" {
		http.Error(w, "module not found", http.StatusNotFound)
		return mod, false
	}
	return mod, true
}

// claudeOwnSession loads {sessionID} and checks it belongs to the caller on this module.
func claudeOwnSession(w http.ResponseWriter, r *http.Request, user *core.User, mod core.Module) (*database.ClaudeSession, bool) {
	sess, err := core.GetClaudeSession(chi.URLParam(r, "sessionID"))
	if err != nil {
		http.Error(w, "failed to load session", http.StatusInternalServerError)
		return nil, false
	}
	if sess == nil || sess.ModuleID != mod.ID || sess.UserID != user.ID {
		http.Error(w, "session not found", http.StatusNotFound)
		return nil, false
	}
	return sess, true
}

func writeClaudeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GetClaudeSessions lists the caller's Claude sessions on a module.
// @Security     AdminAuth
// @Summary      List Claude sessions
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Success      200        {object}  claudeSessionsResponse
// @Router       /admin/modules/{moduleID}/claude/sessions [get]
func GetClaudeSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	sessions, err := core.ListClaudeSessions(mod.ID, user.ID)
	if err != nil {
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	occupant, err := core.ClaudeModuleOccupant(mod.ID)
	if err != nil {
		http.Error(w, "failed to load module status", http.StatusInternalServerError)
		return
	}
	writeClaudeJSON(w, http.StatusOK, claudeSessionsResponse{Sessions: sessions, Occupant: occupant, Me: user.ID})
}

// PostClaudeSession starts a new Claude session, or resumes one with {"resume": "<id>"}.
// @Security     AdminAuth
// @Summary      Start or resume a Claude session
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Success      201        {object}  database.ClaudeSession
// @Failure      409        {object}  core.ClaudeOccupant "Module already used by someone"
// @Router       /admin/modules/{moduleID}/claude/sessions [post]
func PostClaudeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	var req postClaudeSessionRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
	}
	sess, err := core.StartClaudeSession(r.Context(), *user, mod, req.Resume)
	switch {
	case errors.Is(err, core.ErrClaudeModuleBusy):
		occupant, _ := core.ClaudeModuleOccupant(mod.ID)
		writeClaudeJSON(w, http.StatusConflict, map[string]any{"error": "module_busy", "occupant": occupant})
		return
	case errors.Is(err, core.ErrClaudeLimitReached):
		auth.WriteJSONError(w, http.StatusServiceUnavailable, "claude_limit", "Too many Claude sessions are running, retry later.")
		return
	case errors.Is(err, core.ErrClaudeNotResumable):
		auth.WriteJSONError(w, http.StatusBadRequest, "not_resumable", "This session cannot be resumed.")
		return
	case err != nil:
		log.Printf("[claude] %s: start failed on module %s: %v", user.FtLogin, mod.Slug, err)
		auth.WriteJSONError(w, http.StatusInternalServerError, "claude_start_failed", "Unable to start Claude.")
		return
	}
	writeClaudeJSON(w, http.StatusCreated, sess)
}

// StopClaudeSession stops a session's container; a working session requires
// ?force=true. ?kill=true destroys the container right away (hung session).
// @Security     AdminAuth
// @Summary      Stop a Claude session
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Param        sessionID  path      string  true  "Session ID"
// @Param        force      query     bool    false "Stop even if Claude is working"
// @Param        kill       query     bool    false "Destroy the container immediately"
// @Success      204        ""
// @Failure      409        "Claude is working"
// @Router       /admin/modules/{moduleID}/claude/sessions/{sessionID}/stop [post]
func StopClaudeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	sess, ok := claudeOwnSession(w, r, user, mod)
	if !ok {
		return
	}
	kill := r.URL.Query().Get("kill") == "true"
	err := core.StopClaudeSession(*sess, kill || r.URL.Query().Get("force") == "true", kill)
	switch {
	case errors.Is(err, core.ErrClaudeWorking):
		auth.WriteJSONError(w, http.StatusConflict, "claude_working", "Claude is working: confirm to stop it anyway.")
	case errors.Is(err, core.ErrClaudeNotActive):
		w.WriteHeader(http.StatusNoContent)
	case err != nil:
		log.Printf("[claude] %s: stop failed on module %s: %v", user.FtLogin, mod.Slug, err)
		auth.WriteJSONError(w, http.StatusInternalServerError, "claude_stop_failed", "Docker could not remove the Claude container, the session is kept locked. Retry, or contact an administrator.")
	default:
		log.Printf("[claude] %s stopped session %s on module %s (kill=%t)", user.FtLogin, sess.ID, mod.Slug, kill)
		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteClaudeSession deletes a session and its transcript (stopping it first if needed).
// @Security     AdminAuth
// @Summary      Delete a Claude session
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Param        sessionID  path      string  true  "Session ID"
// @Param        force      query     bool    false "Delete even if Claude is working"
// @Success      204        ""
// @Router       /admin/modules/{moduleID}/claude/sessions/{sessionID} [delete]
func DeleteClaudeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	sess, ok := claudeOwnSession(w, r, user, mod)
	if !ok {
		return
	}
	err := core.DeleteClaudeSession(*sess, r.URL.Query().Get("force") == "true")
	switch {
	case errors.Is(err, core.ErrClaudeWorking):
		auth.WriteJSONError(w, http.StatusConflict, "claude_working", "Claude is working: confirm to stop it anyway.")
	case err != nil:
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetClaudeActive lists the running Claude sessions of every module, plus the
// caller's ended sessions with an unseen result (modules list badges).
// @Security     AdminAuth
// @Summary      Claude activity
// @Tags         Modules
// @Success      200        {object}  object "{me, sessions}"
// @Router       /admin/claude/active [get]
func GetClaudeActive(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	list, err := core.ListClaudeActivity(user.ID)
	if err != nil {
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	writeClaudeJSON(w, http.StatusOK, map[string]any{"me": user.ID, "sessions": list})
}

// GetClaudeUserSessions lists the caller's Claude sessions on every module
// (running first, then the most recent), for the Claude page.
// @Security     AdminAuth
// @Summary      My Claude sessions
// @Tags         Modules
// @Success      200        {object}  object "{me, sessions}"
// @Router       /admin/claude/sessions [get]
func GetClaudeUserSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	list, err := core.ListUserClaudeSessions(user.ID)
	if err != nil {
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	writeClaudeJSON(w, http.StatusOK, map[string]any{"me": user.ID, "sessions": list})
}

// PostClaudeSeen marks the caller's Claude results on a module as seen (Claude tab displayed).
// @Security     AdminAuth
// @Summary      Mark Claude results as seen
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Success      204        ""
// @Router       /admin/modules/{moduleID}/claude/seen [post]
func PostClaudeSeen(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	if err := core.MarkClaudeSeen(mod.ID, user.ID); err != nil {
		http.Error(w, "failed to update sessions", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClaudeTerminal attaches the browser to the session's tmux (WebSocket). Same
// framing as the Shell tab: binary frames carry raw terminal I/O, text frames
// JSON control messages ({"type":"resize","cols":80,"rows":24}). Closing it only
// detaches: claude keeps running in its container.
// @Security     AdminAuth
// @Summary      Claude terminal (WebSocket)
// @Tags         Modules
// @Param        moduleID   path      string  true  "Module ID"
// @Param        sessionID  path      string  true  "Session ID"
// @Router       /admin/modules/{moduleID}/claude/sessions/{sessionID}/terminal [get]
func ClaudeTerminal(w http.ResponseWriter, r *http.Request) {
	user, ok := claudeUser(w, r)
	if !ok {
		return
	}
	if !userIsAdmin(r, user) {
		auth.WriteJSONError(w, http.StatusForbidden, "admin_required", "You are not allowed to view this content.")
		return
	}
	if !shellOriginAllowed(r) {
		auth.WriteJSONError(w, http.StatusForbidden, "bad_origin", "Cross-origin terminal access is not allowed.")
		return
	}
	mod, ok := claudeModule(w, r)
	if !ok {
		return
	}
	sess, ok := claudeOwnSession(w, r, user, mod)
	if !ok {
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

	term, err := core.AttachClaudeTerminal(r.Context(), *sess, 80, 24)
	if err != nil {
		if errors.Is(err, core.ErrClaudeNotActive) {
			fail("This session is not running.")
		} else {
			log.Printf("[claude] %s: attach failed on module %s: %v", user.FtLogin, mod.Slug, err)
			fail("Unable to attach to Claude.")
		}
		return
	}
	defer term.Close()

	done := make(chan struct{})
	var once sync.Once
	stop := func() { once.Do(func() { close(done); conn.Close() }) }
	defer stop()

	// claude -> browser
	go func() {
		defer stop()
		buf := make([]byte, 8192)
		for {
			n, err := term.Read(buf)
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

	// Keep proxies from closing an idle connection, and revoke on admin role loss.
	go func() {
		ping := time.NewTicker(claudePingInterval)
		recheck := time.NewTicker(claudeAdminRecheck)
		defer ping.Stop()
		defer recheck.Stop()
		for {
			select {
			case <-done:
				return
			case <-ping.C:
				if send(websocket.PingMessage, nil) != nil {
					stop()
					return
				}
			case <-recheck.C:
				if !userIsAdmin(r, user) {
					log.Printf("[claude] %s lost admin role, detaching from module %s", user.FtLogin, mod.Slug)
					stop()
					return
				}
			}
		}
	}()

	// browser -> claude
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch msgType {
		case websocket.BinaryMessage:
			if _, err := term.Write(data); err != nil {
				return
			}
		case websocket.TextMessage:
			var ctl shellControl
			if json.Unmarshal(data, &ctl) == nil && ctl.Type == "resize" && ctl.Cols > 0 && ctl.Rows > 0 && ctl.Cols < 1000 && ctl.Rows < 1000 {
				_ = term.Resize(ctl.Cols, ctl.Rows)
			}
		}
	}
}
