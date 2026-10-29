package core

import (
	"backend/database"
	"strings"
	"testing"
	"time"
)

func TestClaudeReapDecision(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }
	idleTTL, maxWork := claudeCfg.idleTTL, claudeCfg.maxWork

	cases := []struct {
		name     string
		sess     database.ClaudeSession
		attached int
		running  bool
		want     string
	}{
		{"dead container ends the session",
			database.ClaudeSession{Status: database.ClaudeWorking, LastActivityAt: now}, 0, false, database.ClaudeStopped},
		{"idle below TTL is kept",
			database.ClaudeSession{Status: database.ClaudeIdle, LastActivityAt: *ago(idleTTL - time.Minute)}, 0, true, ""},
		{"idle above TTL without client is stopped",
			database.ClaudeSession{Status: database.ClaudeIdle, LastActivityAt: *ago(idleTTL + time.Minute)}, 0, true, database.ClaudeStopped},
		{"idle above TTL with a client attached is kept",
			database.ClaudeSession{Status: database.ClaudeIdle, LastActivityAt: *ago(idleTTL + time.Minute)}, 1, true, ""},
		{"recent detach restarts the idle countdown",
			database.ClaudeSession{Status: database.ClaudeIdle, LastActivityAt: *ago(idleTTL * 3), LastAttachedAt: ago(time.Minute)}, 0, true, ""},
		{"starting (login in progress) follows the idle rule",
			database.ClaudeSession{Status: database.ClaudeStarting, LastActivityAt: *ago(idleTTL + time.Minute)}, 0, true, database.ClaudeStopped},
		{"working is never reaped below the cap, even long idle without client",
			database.ClaudeSession{Status: database.ClaudeWorking, LastActivityAt: *ago(idleTTL * 10), WorkStartedAt: ago(maxWork - time.Minute)}, 0, true, ""},
		{"waiting_input is never reaped below the cap",
			database.ClaudeSession{Status: database.ClaudeWaitingInput, LastActivityAt: *ago(idleTTL * 10), WorkStartedAt: ago(time.Hour)}, 0, true, ""},
		{"working above the cap is aborted",
			database.ClaudeSession{Status: database.ClaudeWorking, LastActivityAt: now, WorkStartedAt: ago(maxWork + time.Minute)}, 1, true, database.ClaudeAborted},
		{"waiting_input above the cap is aborted",
			database.ClaudeSession{Status: database.ClaudeWaitingInput, LastActivityAt: now, WorkStartedAt: ago(maxWork + time.Minute)}, 0, true, database.ClaudeAborted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := claudeReapDecision(tc.sess, tc.attached, tc.running, now); got != tc.want {
				t.Fatalf("claudeReapDecision() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClaudeNextStatus(t *testing.T) {
	cases := []struct {
		cur, event string
		in         claudeHookInput
		want       string
	}{
		{database.ClaudeStarting, "session_start", claudeHookInput{}, database.ClaudeIdle},
		{database.ClaudeWorking, "session_start", claudeHookInput{Source: "compact"}, ""},
		{database.ClaudeIdle, "prompt", claudeHookInput{}, database.ClaudeWorking},
		{database.ClaudeWorking, "notification", claudeHookInput{NotificationType: "permission_prompt"}, database.ClaudeWaitingInput},
		{database.ClaudeWorking, "notification", claudeHookInput{NotificationType: "elicitation_dialog"}, database.ClaudeWaitingInput},
		// The idle reminder arrives after Stop and must not flip back to waiting_input.
		{database.ClaudeIdle, "notification", claudeHookInput{NotificationType: "idle_prompt"}, ""},
		{database.ClaudeIdle, "notification", claudeHookInput{NotificationType: "permission_prompt"}, ""},
		// After an interrupt (Esc) no Stop fires: the idle reminder brings it back to idle.
		{database.ClaudeWorking, "notification", claudeHookInput{NotificationType: "idle_prompt"}, database.ClaudeIdle},
		{database.ClaudeWaitingInput, "notification", claudeHookInput{NotificationType: "idle_prompt"}, database.ClaudeIdle},
		{database.ClaudeWorking, "stop_failure", claudeHookInput{Error: "rate_limit"}, database.ClaudeIdle},
		{database.ClaudeWaitingInput, "tool_done", claudeHookInput{}, database.ClaudeWorking},
		{database.ClaudeWorking, "tool_done", claudeHookInput{}, ""},
		{database.ClaudeWorking, "stop", claudeHookInput{}, database.ClaudeIdle},
		{database.ClaudeWaitingInput, "stop", claudeHookInput{}, database.ClaudeIdle},
		{database.ClaudeIdle, "session_end", claudeHookInput{Reason: "clear"}, ""},
		{database.ClaudeIdle, "session_end", claudeHookInput{Reason: "prompt_input_exit"}, database.ClaudeIdle},
		{database.ClaudeIdle, "bogus", claudeHookInput{}, ""},
	}
	for _, tc := range cases {
		if got := claudeNextStatus(tc.cur, tc.event, tc.in); got != tc.want {
			t.Errorf("claudeNextStatus(%q, %q, %+v) = %q, want %q", tc.cur, tc.event, tc.in, got, tc.want)
		}
	}
}

func TestClaudeTitle(t *testing.T) {
	if got := claudeTitle("  fix the login bug\nmore details"); got != "fix the login bug" {
		t.Errorf("first line expected, got %q", got)
	}
	long := strings.Repeat("é", 200)
	got := claudeTitle(long)
	if n := len([]rune(got)); n != claudeTitleMaxRune || !strings.HasSuffix(got, "…") {
		t.Errorf("title not truncated to %d runes: %d %q", claudeTitleMaxRune, n, got)
	}
}

func TestClaudeNamesAreDockerSafe(t *testing.T) {
	mod := Module{Slug: "watchdog"}
	if got := claudeContainerName(mod, "user_01HXYZ"); got != "pb-claude-watchdog-user_01hxyz" {
		t.Errorf("unexpected container name %q", got)
	}
	if got := claudeHomeVolume("user_01HXYZ"); got != "pb-claude-home-user_01hxyz" {
		t.Errorf("unexpected volume name %q", got)
	}
}
