package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sebrandon1/ztp-dashboard/internal/ws"
)

// EventStore persists watch events in an embedded SQLite database.
type EventStore struct {
	db *sql.DB
	mu sync.Mutex
}

// EventQuery holds search/filter parameters for listing events.
type EventQuery struct {
	Query        string
	Severity     string
	ResourceType string
	From         string
	To           string
	Limit        int
	Offset       int
}

// EventStats holds event count breakdowns.
type EventStats struct {
	BySeverity     map[string]int `json:"bySeverity"`
	ByResourceType map[string]int `json:"byResourceType"`
	Total          int            `json:"total"`
}

// NewEventStore opens or creates the SQLite database at the given path.
func NewEventStore(dbPath string) (*EventStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening event store: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, err
	}

	return &EventStore{db: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		name TEXT NOT NULL,
		namespace TEXT NOT NULL DEFAULT '',
		summary TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL DEFAULT 'neutral',
		insight TEXT NOT NULL DEFAULT '',
		data TEXT,
		timestamp TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
	CREATE INDEX IF NOT EXISTS idx_events_resource_type ON events(resource_type);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("creating event schema: %w", err)
	}
	return nil
}

// Insert stores a watch event.
func (s *EventStore) Insert(event ws.WatchEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	var dataJSON []byte
	if event.Data != nil {
		var err error
		dataJSON, err = json.Marshal(event.Data)
		if err != nil {
			dataJSON = nil
		}
	}

	_, err := s.db.Exec(
		`INSERT INTO events (event_type, resource_type, name, namespace, summary, severity, insight, data, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventType, event.ResourceType, event.Name, event.Namespace,
		event.Summary, event.Severity, event.Insight, string(dataJSON), event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}
	return nil
}

// Query searches and paginates events.
func (s *EventStore) Query(q EventQuery) ([]ws.WatchEvent, int, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 500 {
		q.Limit = 500
	}

	where, args := buildWhere(q)

	// Get total count
	var total int
	countSQL := "SELECT COUNT(*) FROM events" + where
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting events: %w", err)
	}

	// Get results
	querySQL := "SELECT event_type, resource_type, name, namespace, summary, severity, insight, data, timestamp FROM events" +
		where + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []ws.WatchEvent
	for rows.Next() {
		var e ws.WatchEvent
		var dataStr sql.NullString
		if err := rows.Scan(&e.EventType, &e.ResourceType, &e.Name, &e.Namespace,
			&e.Summary, &e.Severity, &e.Insight, &dataStr, &e.Timestamp); err != nil {
			return nil, 0, fmt.Errorf("scanning event row: %w", err)
		}
		if dataStr.Valid && dataStr.String != "" {
			var data any
			if err := json.Unmarshal([]byte(dataStr.String), &data); err == nil {
				e.Data = data
			}
		}
		events = append(events, e)
	}

	return events, total, nil
}

func buildWhere(q EventQuery) (string, []any) {
	var conditions []string
	var args []any

	if q.Severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, q.Severity)
	}
	if q.ResourceType != "" {
		conditions = append(conditions, "resource_type = ?")
		args = append(args, q.ResourceType)
	}
	if q.Query != "" {
		conditions = append(conditions, "(name LIKE ? OR summary LIKE ? OR resource_type LIKE ?)")
		like := "%" + q.Query + "%"
		args = append(args, like, like, like)
	}
	if q.From != "" {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, q.From)
	}
	if q.To != "" {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, q.To)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(" WHERE ")
	for i, c := range conditions {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(c)
	}
	return sb.String(), args
}

// GetStats returns event counts by severity and resource type for the last 24 hours.
func (s *EventStore) GetStats() (*EventStats, error) {
	cutoff := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)

	stats := &EventStats{
		BySeverity:     make(map[string]int),
		ByResourceType: make(map[string]int),
	}

	// By severity
	rows, err := s.db.Query(
		"SELECT severity, COUNT(*) FROM events WHERE timestamp >= ? GROUP BY severity", cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying severity stats: %w", err)
	}
	for rows.Next() {
		var sev string
		var count int
		if err := rows.Scan(&sev, &count); err == nil {
			stats.BySeverity[sev] = count
			stats.Total += count
		}
	}
	_ = rows.Close()

	// By resource type
	rows, err = s.db.Query(
		"SELECT resource_type, COUNT(*) FROM events WHERE timestamp >= ? GROUP BY resource_type", cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying resource type stats: %w", err)
	}
	for rows.Next() {
		var rt string
		var count int
		if err := rows.Scan(&rt, &count); err == nil {
			stats.ByResourceType[rt] = count
		}
	}
	_ = rows.Close()

	return stats, nil
}

// Purge removes events older than the given duration.
func (s *EventStore) Purge(maxAge time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM events WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("purging old events: %w", err)
	}
	return result.RowsAffected()
}

// StartPurgeLoop runs a background goroutine that purges events older than maxAge.
func (s *EventStore) StartPurgeLoop(maxAge time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			deleted, err := s.Purge(maxAge)
			if err != nil {
				slog.Error("event purge failed", "error", err)
			} else if deleted > 0 {
				slog.Info("purged old events", "deleted", deleted)
			}
		}
	}()
}

// Close closes the database connection.
func (s *EventStore) Close() error {
	return s.db.Close()
}
