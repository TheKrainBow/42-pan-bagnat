package core

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"backend/database"
)

// The "pb" socket of a Claude container: Claude Code hooks report the session
// status there (/opt/pb/hook.sh), and the `pb` command asks PB to deploy the
// module (/opt/pb/pb). Only that container has the socket mounted; requests
// also carry the container's token.

const (
	claudeHookMaxBody  = 1 << 20
	claudeTitleMaxRune = 80
)

type claudeHookServer struct {
	srv    *http.Server
	sock   string
	events chan claudeHookEvent
	done   chan struct{}
}

type claudeHookEvent struct {
	name string
	in   claudeHookInput
}

func startClaudeHookServer(rt *claudeRuntime, sock string, uid, gid int) (*claudeHookServer, error) {
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(sock, 0o600)
	if uid != 0 || gid != 0 {
		_ = os.Chown(sock, uid, gid)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/hook", rt.serveHook)
	mux.HandleFunc("/action/", rt.serveAction)
	h := &claudeHookServer{
		sock:   sock,
		srv:    &http.Server{Handler: rt.authorize(mux), ReadHeaderTimeout: 10 * time.Second},
		events: make(chan claudeHookEvent, 128),
		done:   make(chan struct{}),
	}
	// Events are applied one at a time, in the order claude sent them.
	go func() {
		for {
			select {
			case ev := <-h.events:
				rt.handleHook(ev.name, ev.in)
			case <-h.done:
				return
			}
		}
	}()
	go h.srv.Serve(ln)
	return h, nil
}

func (h *claudeHookServer) Close() {
	_ = h.srv.Close()
	_ = os.Remove(h.sock)
	close(h.done)
}

func (rt *claudeRuntime) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if r.Method != http.MethodPost || subtle.ConstantTimeCompare([]byte(got), []byte(rt.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// claudeHookInput is the part of the Claude Code hook payload PB uses.
type claudeHookInput struct {
	SessionID        string `json:"session_id"`
	Source           string `json:"source"`
	Reason           string `json:"reason"`
	Prompt           string `json:"prompt"`
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"`
	Error            string `json:"error"`
}

func (rt *claudeRuntime) serveHook(w http.ResponseWriter, r *http.Request) {
	var in claudeHookInput
	body, _ := io.ReadAll(io.LimitReader(r.Body, claudeHookMaxBody))
	_ = json.Unmarshal(body, &in)
	// Answer right away: hooks must never slow claude down.
	select {
	case rt.hook.events <- claudeHookEvent{name: r.URL.Query().Get("event"), in: in}:
	default:
		log.Printf("[claude] hook queue full on module %s, event dropped", rt.slug)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *claudeRuntime) handleHook(event string, in claudeHookInput) {
	sess, err := database.GetActiveClaudeSessionByContainer(rt.containerID)
	if err != nil || sess == nil {
		return
	}
	prev := sess.Status
	next := claudeNextStatus(prev, event, in)

	if event == "session_start" && claudeSessionIDRe.MatchString(in.SessionID) {
		newID, _ := GenerateULID(ClaudeSessionKind)
		if bound, err := database.BindClaudeSessionID(rt.containerID, in.SessionID, newID); err == nil && bound != nil {
			if bound.ID != sess.ID {
				if old, _ := database.GetClaudeSession(sess.ID); old != nil {
					emitClaudeStatus(*old)
				}
			}
			sess, prev = bound, bound.Status
		} else if err != nil {
			log.Printf("[claude] bind session %s: %v", in.SessionID, err)
		}
	}
	if event == "prompt" {
		if title := claudeTitle(in.Prompt); title != "" && sess.Title == "" {
			_ = database.SetClaudeSessionTitleIfEmpty(sess.ID, title)
		}
	}
	if next != "" && next != prev {
		_ = database.SetClaudeSessionStatus(sess.ID, next)
	} else {
		_ = database.TouchClaudeSession(sess.ID)
	}
	wasWorking := prev == database.ClaudeWorking || prev == database.ClaudeWaitingInput
	finished := (event == "stop" || event == "stop_failure") && wasWorking
	switch {
	case finished:
		_ = database.SetClaudeSessionUnseen(sess.ID, true)
	case event == "prompt" && sess.UnseenResult:
		// The owner is typing in the terminal: they have seen the result.
		_ = database.SetClaudeSessionUnseen(sess.ID, false)
	}
	cur, _ := database.GetClaudeSession(sess.ID)
	if cur == nil {
		return
	}
	if cur.Status != prev || cur.UnseenResult != sess.UnseenResult || event == "session_start" || event == "prompt" {
		emitClaudeStatus(*cur)
	}
	switch {
	case next == database.ClaudeWaitingInput && prev != next:
		msg := strings.TrimSpace(in.Message)
		if msg == "" {
			msg = "Claude is waiting for your answer."
		}
		notifyClaudeUser(*cur, "waiting", msg)
	case event == "stop" && wasWorking:
		notifyClaudeUser(*cur, "done", "Claude has finished its task.")
	case event == "stop_failure" && wasWorking:
		notifyClaudeUser(*cur, "error", "Claude stopped on an error ("+in.Error+").")
	}
}

// claudeNextStatus maps a hook event to the session's next status ("" keeps it).
func claudeNextStatus(cur, event string, in claudeHookInput) string {
	switch event {
	case "session_start":
		if cur == database.ClaudeStarting {
			return database.ClaudeIdle
		}
	case "prompt":
		return database.ClaudeWorking
	case "notification":
		// Claude waits at its prompt: no Stop hook fires when the user interrupts
		// a task (Esc), this reminder is the only sign that it is idle again.
		if in.NotificationType == "idle_prompt" {
			if cur == database.ClaudeWorking || cur == database.ClaudeWaitingInput {
				return database.ClaudeIdle
			}
			return ""
		}
		// Only a question asked during a task counts.
		if cur != database.ClaudeWorking {
			return ""
		}
		switch in.NotificationType {
		case "permission_prompt", "elicitation_dialog":
			return database.ClaudeWaitingInput
		case "":
			if strings.Contains(strings.ToLower(in.Message), "permission") {
				return database.ClaudeWaitingInput
			}
		}
	case "tool_done":
		// A tool ran after a permission prompt: the user answered.
		if cur == database.ClaudeWaitingInput {
			return database.ClaudeWorking
		}
	case "stop", "stop_failure":
		return database.ClaudeIdle
	case "session_end":
		// The container exits right after (watchClaudeContainers ends the
		// session); /clear and /resume start a new claude session in the same one.
		if in.Reason != "clear" && in.Reason != "resume" {
			return database.ClaudeIdle
		}
	}
	return ""
}

func claudeTitle(prompt string) string {
	line := strings.TrimSpace(prompt)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if utf8.RuneCountInString(line) <= claudeTitleMaxRune {
		return line
	}
	r := []rune(line)
	return strings.TrimSpace(string(r[:claudeTitleMaxRune-1])) + "…"
}

// serveAction runs a module deployment action for the `pb` command and streams
// the module log lines it produces. The last line is "PB_RESULT ok" or
// "PB_RESULT failed: <reason>".
func (rt *claudeRuntime) serveAction(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimPrefix(r.URL.Path, "/action/")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	flusher, _ := w.(http.Flusher)
	write := func(line string) {
		_, _ = io.WriteString(w, line+"\n")
		if flusher != nil {
			flusher.Flush()
		}
	}

	sess, err := database.GetActiveClaudeSessionByContainer(rt.containerID)
	if err != nil || sess == nil {
		write("PB_RESULT failed: no active session")
		return
	}
	if ok, err := database.UserHasRoleByID(context.Background(), sess.UserID, RoleIDAdmin); err != nil || !ok {
		write("PB_RESULT failed: " + sess.UserLogin + " is no longer an admin")
		return
	}
	module, err := GetModule(rt.moduleID)
	if err != nil || module.ID == "" {
		write("PB_RESULT failed: module not found")
		return
	}
	var run func(Module) error
	switch action {
	case "deploy":
		if module.IsDeploying {
			write("PB_RESULT failed: a deployment is already in progress")
			return
		}
		run = DeployModule
	case "rebuild":
		run = ComposeRebuild
	case "down":
		run = ComposeDown
	default:
		write("PB_RESULT failed: unknown action " + action)
		return
	}

	lines, untap := tapModuleLogs(module.ID)
	defer untap()
	LogModule(module.ID, "INFO", fmt.Sprintf("%s requested by Claude (session of %s)", action, sess.UserLogin), nil, nil)
	_ = database.TouchClaudeSession(sess.ID)

	done := make(chan error, 1)
	// The action is not tied to the request: it completes even if `pb` is interrupted.
	go func() { done <- run(module) }()
	for {
		select {
		case line := <-lines:
			write(line)
		case err := <-done:
			for drained := false; !drained; {
				select {
				case line := <-lines:
					write(line)
				default:
					drained = true
				}
			}
			if err != nil {
				write("PB_RESULT failed: " + err.Error())
			} else {
				write("PB_RESULT ok")
			}
			return
		}
	}
}
