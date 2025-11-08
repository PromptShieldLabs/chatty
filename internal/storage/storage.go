package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultDirName  = ".local/share/chatty"
	defaultFileName = "chatty.db"
	timestampLayout = time.RFC3339
)

// Store wraps access to the persistent conversation database.
type Store struct {
	db *sql.DB
}

// Message represents a persisted chat message.
type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// SessionSummary describes a saved conversation.
type SessionSummary struct {
	ID           int64
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
}

// Transcript bundles a session summary with its messages.
type Transcript struct {
	Summary  SessionSummary
	Messages []Message
}

// Open initialises the storage layer, creating the database if necessary.
func Open(path string) (*Store, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", resolved)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL journal: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close releases underlying database resources.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
            updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
        );`,
		`CREATE TABLE IF NOT EXISTS messages (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id INTEGER NOT NULL,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
            FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
        );`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}

	return nil
}

// CreateSession inserts a new conversation row and returns its identifier.
func (s *Store) CreateSession(ctx context.Context, name string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("storage not initialised")
	}

	title := strings.TrimSpace(name)
	if title == "" {
		title = fmt.Sprintf("Session %s", time.Now().Format("2006-01-02 15:04"))
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO sessions(name) VALUES (?)`, title)
	if err != nil {
		return 0, fmt.Errorf("insert session: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve session id: %w", err)
	}

	return id, nil
}

// UpdateSessionName updates the stored name for a session.
func (s *Store) UpdateSessionName(ctx context.Context, id int64, name string) error {
	if s == nil || s.db == nil {
		return errors.New("storage not initialised")
	}
	if id <= 0 {
		return errors.New("invalid session id")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("session name cannot be empty")
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET name = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ','now')) WHERE id = ?`, trimmed, id); err != nil {
		return fmt.Errorf("update session name: %w", err)
	}

	return nil
}

// AppendMessage appends a message to the specified session.
func (s *Store) AppendMessage(ctx context.Context, sessionID int64, message Message) error {
	if s == nil || s.db == nil {
		return errors.New("storage not initialised")
	}
	if sessionID <= 0 {
		return errors.New("invalid session id")
	}
	if strings.TrimSpace(message.Role) == "" {
		return errors.New("message role cannot be empty")
	}

	if _, err := s.db.ExecContext(ctx, `INSERT INTO messages(session_id, role, content) VALUES (?, ?, ?)`, sessionID, message.Role, message.Content); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ','now')) WHERE id = ?`, sessionID); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return nil
}

// ListSessions returns stored conversations ordered by most recent activity.
func (s *Store) ListSessions(ctx context.Context, limit int) ([]SessionSummary, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("storage not initialised")
	}

	baseQuery := `SELECT s.id, s.name, s.created_at, s.updated_at, COUNT(m.id) AS message_count
        FROM sessions s
        LEFT JOIN messages m ON m.session_id = s.id
        GROUP BY s.id
        ORDER BY s.updated_at DESC`

	var rows *sql.Rows
	var err error
	if limit > 0 {
		baseQuery += " LIMIT ?"
		rows, err = s.db.QueryContext(ctx, baseQuery, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, baseQuery)
	}
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	summaries := make([]SessionSummary, 0, 8)
	for rows.Next() {
		var summary SessionSummary
		var created, updated string
		if err := rows.Scan(&summary.ID, &summary.Name, &created, &updated, &summary.MessageCount); err != nil {
			return nil, fmt.Errorf("scan session summary: %w", err)
		}
		summary.CreatedAt, err = parseTimestamp(created)
		if err != nil {
			return nil, err
		}
		summary.UpdatedAt, err = parseTimestamp(updated)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session summaries: %w", err)
	}

	return summaries, nil
}

// LoadSession fetches the session metadata and full transcript for the given identifier.
func (s *Store) LoadSession(ctx context.Context, id int64) (*Transcript, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("storage not initialised")
	}
	if id <= 0 {
		return nil, errors.New("invalid session id")
	}

	var summary SessionSummary
	var created, updated string
	row := s.db.QueryRowContext(ctx, `SELECT s.id, s.name, s.created_at, s.updated_at, COUNT(m.id) AS message_count
        FROM sessions s
        LEFT JOIN messages m ON m.session_id = s.id
        WHERE s.id = ?
        GROUP BY s.id`, id)
	if err := row.Scan(&summary.ID, &summary.Name, &created, &updated, &summary.MessageCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session %d not found", id)
		}
		return nil, fmt.Errorf("select session: %w", err)
	}

	var err error
	summary.CreatedAt, err = parseTimestamp(created)
	if err != nil {
		return nil, err
	}
	summary.UpdatedAt, err = parseTimestamp(updated)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`, id)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0, summary.MessageCount)
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(&msg.Role, &msg.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msg.CreatedAt, err = parseTimestamp(createdAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return &Transcript{Summary: summary, Messages: messages}, nil
}

func resolvePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		trimmed = filepath.Join(home, defaultDirName, defaultFileName)
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create storage directory: %w", err)
	}

	return absPath, nil
}

func parseTimestamp(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(timestampLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", value, err)
	}
	return t, nil
}
