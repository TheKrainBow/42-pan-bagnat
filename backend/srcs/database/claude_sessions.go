package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

// Claude session statuses (see db/migrations/37_claude_sessions.up.sql).
const (
	ClaudeStarting     = "starting"
	ClaudeIdle         = "idle"
	ClaudeWorking      = "working"
	ClaudeWaitingInput = "waiting_input"
	ClaudeStopped      = "stopped"
	ClaudeAborted      = "aborted"
)

// ErrClaudeModuleBusy is returned when another session already holds the
// module's container (claude_one_active_per_module unique index).
var ErrClaudeModuleBusy = errors.New("module already has an active claude session")

type ClaudeSession struct {
	ID              string     `json:"id"`
	ModuleID        string     `json:"module_id"`
	UserID          string     `json:"user_id"`
	UserLogin       string     `json:"user_login"`
	ClaudeSessionID string     `json:"claude_session_id"`
	Title           string     `json:"title"`
	Status          string     `json:"status"`
	ContainerID     string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       time.Time  `json:"started_at"`
	LastActivityAt  time.Time  `json:"last_activity_at"`
	LastAttachedAt  *time.Time `json:"last_attached_at"`
	WorkStartedAt   *time.Time `json:"work_started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	// UnseenResult: a task ended and the owner has not looked at the Claude tab since.
	UnseenResult bool `json:"unseen_result"`
}

// Active reports whether the session currently owns a running container.
func (s ClaudeSession) Active() bool {
	return s.ContainerID != "" && IsClaudeActiveStatus(s.Status)
}

func IsClaudeActiveStatus(status string) bool {
	switch status {
	case ClaudeStarting, ClaudeIdle, ClaudeWorking, ClaudeWaitingInput:
		return true
	}
	return false
}

const claudeSessionColumns = `
	cs.id, cs.module_id, cs.user_id, COALESCE(u.ft_login, ''), COALESCE(cs.claude_session_id, ''),
	cs.title, cs.status, COALESCE(cs.container_id, ''), cs.created_at, cs.started_at, cs.last_activity_at,
	cs.last_attached_at, cs.work_started_at, cs.ended_at, cs.unseen_result`

const claudeSessionFrom = `
	FROM claude_sessions cs
	LEFT JOIN users u ON u.id = cs.user_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClaudeSession(row rowScanner) (ClaudeSession, error) {
	var s ClaudeSession
	var attached, workStarted, ended sql.NullTime
	err := row.Scan(&s.ID, &s.ModuleID, &s.UserID, &s.UserLogin, &s.ClaudeSessionID,
		&s.Title, &s.Status, &s.ContainerID, &s.CreatedAt, &s.StartedAt, &s.LastActivityAt,
		&attached, &workStarted, &ended, &s.UnseenResult)
	if attached.Valid {
		s.LastAttachedAt = &attached.Time
	}
	if workStarted.Valid {
		s.WorkStartedAt = &workStarted.Time
	}
	if ended.Valid {
		s.EndedAt = &ended.Time
	}
	return s, err
}

func queryClaudeSessions(query string, args ...any) ([]ClaudeSession, error) {
	rows, err := mainDB.Query(`SELECT `+claudeSessionColumns+claudeSessionFrom+` `+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClaudeSession{}
	for rows.Next() {
		s, err := scanClaudeSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func queryClaudeSession(query string, args ...any) (*ClaudeSession, error) {
	s, err := scanClaudeSession(mainDB.QueryRow(`SELECT `+claudeSessionColumns+claudeSessionFrom+` `+query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func isClaudeBusyErr(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "claude_one_active_per_module"
}

func GetClaudeSession(id string) (*ClaudeSession, error) {
	return queryClaudeSession(`WHERE cs.id = $1`, id)
}

// GetActiveClaudeSessionByModule returns the session holding the module's container, if any.
func GetActiveClaudeSessionByModule(moduleID string) (*ClaudeSession, error) {
	return queryClaudeSession(`WHERE cs.module_id = $1 AND cs.container_id IS NOT NULL
		AND cs.status IN ('starting','idle','working','waiting_input')`, moduleID)
}

func GetActiveClaudeSessionByContainer(containerID string) (*ClaudeSession, error) {
	return queryClaudeSession(`WHERE cs.container_id = $1
		AND cs.status IN ('starting','idle','working','waiting_input')`, containerID)
}

func ListClaudeSessions(moduleID, userID string) ([]ClaudeSession, error) {
	return queryClaudeSessions(`WHERE cs.module_id = $1 AND cs.user_id = $2
		ORDER BY cs.last_activity_at DESC`, moduleID, userID)
}

// ListUserClaudeSessions returns a user's sessions on every module: the ones
// holding a container first, then the most recently active, up to limit.
func ListUserClaudeSessions(userID string, limit int) ([]ClaudeSession, error) {
	return queryClaudeSessions(`WHERE cs.user_id = $1
		ORDER BY (cs.container_id IS NOT NULL) DESC, cs.last_activity_at DESC LIMIT $2`, userID, limit)
}

// ListActiveClaudeSessions returns every session that holds a container.
func ListActiveClaudeSessions() ([]ClaudeSession, error) {
	return queryClaudeSessions(`WHERE cs.container_id IS NOT NULL ORDER BY cs.created_at`)
}

// ListExpiredClaudeSessions returns inactive sessions whose last activity is older than before.
func ListExpiredClaudeSessions(before time.Time) ([]ClaudeSession, error) {
	return queryClaudeSessions(`WHERE cs.container_id IS NULL AND cs.last_activity_at < $1`, before)
}

// CreateClaudeSession inserts a new session already bound to its future container
// (container_id is a placeholder until the container exists), which takes the
// module lock atomically.
func CreateClaudeSession(id, moduleID, userID, lockID string) error {
	_, err := mainDB.Exec(`
		INSERT INTO claude_sessions (id, module_id, user_id, status, container_id)
		VALUES ($1, $2, $3, 'starting', $4)`, id, moduleID, userID, lockID)
	if isClaudeBusyErr(err) {
		return ErrClaudeModuleBusy
	}
	return err
}

// ReopenClaudeSession takes the module lock for an existing (stopped) session.
func ReopenClaudeSession(id, lockID string) error {
	res, err := mainDB.Exec(`
		UPDATE claude_sessions
		   SET status = 'starting', container_id = $2, ended_at = NULL,
		       work_started_at = NULL, last_activity_at = now(), started_at = now()
		 WHERE id = $1 AND container_id IS NULL`, id, lockID)
	if isClaudeBusyErr(err) {
		return ErrClaudeModuleBusy
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrClaudeModuleBusy
	}
	return nil
}

func SetClaudeSessionContainer(id, containerID string) error {
	_, err := mainDB.Exec(`UPDATE claude_sessions SET container_id = $2 WHERE id = $1`, id, containerID)
	return err
}

// SetClaudeSessionStatus moves an active session to status. work_started_at
// tracks the start of an uninterrupted working/waiting_input run (max work cap).
func SetClaudeSessionStatus(id, status string) error {
	_, err := mainDB.Exec(`
		UPDATE claude_sessions
		   SET status = $2,
		       last_activity_at = now(),
		       work_started_at = CASE
		         WHEN $2 IN ('working','waiting_input') THEN COALESCE(work_started_at, now())
		         ELSE NULL END
		 WHERE id = $1 AND container_id IS NOT NULL`, id, status)
	return err
}

func TouchClaudeSession(id string) error {
	_, err := mainDB.Exec(`UPDATE claude_sessions SET last_activity_at = now() WHERE id = $1`, id)
	return err
}

func SetClaudeSessionTitleIfEmpty(id, title string) error {
	_, err := mainDB.Exec(`UPDATE claude_sessions SET title = $2 WHERE id = $1 AND title = ''`, id, title)
	return err
}

func MarkClaudeSessionAttached(id string) error {
	_, err := mainDB.Exec(`UPDATE claude_sessions SET last_attached_at = now() WHERE id = $1`, id)
	return err
}

// EndClaudeSession releases the container (and the module lock) with a final status.
func EndClaudeSession(id, status string) error {
	_, err := mainDB.Exec(`
		UPDATE claude_sessions
		   SET status = $2, container_id = NULL, ended_at = now(), work_started_at = NULL
		 WHERE id = $1`, id, status)
	return err
}

func SetClaudeSessionUnseen(id string, unseen bool) error {
	_, err := mainDB.Exec(`UPDATE claude_sessions SET unseen_result = $2 WHERE id = $1`, id, unseen)
	return err
}

// MarkClaudeSessionsSeen clears the unseen flag of a user's sessions on a module
// and returns the ids that changed.
func MarkClaudeSessionsSeen(moduleID, userID string) ([]string, error) {
	rows, err := mainDB.Query(`
		UPDATE claude_sessions SET unseen_result = false
		 WHERE module_id = $1 AND user_id = $2 AND unseen_result
		RETURNING id`, moduleID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListUnseenClaudeSessions returns a user's ended sessions whose result is unseen.
func ListUnseenClaudeSessions(userID string) ([]ClaudeSession, error) {
	return queryClaudeSessions(`WHERE cs.user_id = $1 AND cs.unseen_result AND cs.container_id IS NULL`, userID)
}

func DeleteClaudeSession(id string) error {
	_, err := mainDB.Exec(`DELETE FROM claude_sessions WHERE id = $1`, id)
	return err
}

// BindClaudeSessionID attaches the claude session id reported by SessionStart to
// the session that holds containerID. A new id on a session that already has one
// (/clear, /resume inside claude) moves the container to the matching session,
// created if needed, and ends the previous one. Returns the session now holding
// the container. newID is used when a session has to be created.
func BindClaudeSessionID(containerID, claudeSessionID, newID string) (*ClaudeSession, error) {
	tx, err := mainDB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var curID, moduleID, userID string
	var curClaudeID sql.NullString
	err = tx.QueryRow(`
		SELECT id, module_id, user_id, claude_session_id FROM claude_sessions
		 WHERE container_id = $1 AND status IN ('starting','idle','working','waiting_input')
		 FOR UPDATE`, containerID).Scan(&curID, &moduleID, &userID, &curClaudeID)
	if err != nil {
		return nil, err
	}
	if curClaudeID.String == claudeSessionID {
		return GetClaudeSession(curID)
	}

	var targetID string
	err = tx.QueryRow(`
		SELECT id FROM claude_sessions
		 WHERE module_id = $1 AND user_id = $2 AND claude_session_id = $3
		 FOR UPDATE`, moduleID, userID, claudeSessionID).Scan(&targetID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	switch {
	case targetID == "" && !curClaudeID.Valid:
		// Fresh start: the current session gets its claude id.
		if _, err := tx.Exec(`UPDATE claude_sessions SET claude_session_id = $2, last_activity_at = now() WHERE id = $1`, curID, claudeSessionID); err != nil {
			return nil, err
		}
		targetID = curID
	default:
		// Release the current session first (unique index), then hand the container over.
		if !curClaudeID.Valid {
			if _, err := tx.Exec(`DELETE FROM claude_sessions WHERE id = $1`, curID); err != nil {
				return nil, err
			}
		} else if _, err := tx.Exec(`
			UPDATE claude_sessions SET status = 'stopped', container_id = NULL, ended_at = now(), work_started_at = NULL
			 WHERE id = $1`, curID); err != nil {
			return nil, err
		}
		if targetID == "" {
			targetID = newID
			if _, err := tx.Exec(`
				INSERT INTO claude_sessions (id, module_id, user_id, claude_session_id, status, container_id)
				VALUES ($1, $2, $3, $4, 'idle', $5)`, targetID, moduleID, userID, claudeSessionID, containerID); err != nil {
				return nil, err
			}
		} else if _, err := tx.Exec(`
			UPDATE claude_sessions SET status = 'idle', container_id = $2, ended_at = NULL,
			       work_started_at = NULL, last_activity_at = now(), started_at = now()
			 WHERE id = $1`, targetID, containerID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetClaudeSession(targetID)
}
