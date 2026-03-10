package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebrandon1/ztp-dashboard/internal/ws"
)

func tempDB(t *testing.T) *EventStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewEventStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestInsertAndQuery(t *testing.T) {
	s := tempDB(t)

	events := []ws.WatchEvent{
		{EventType: "ADDED", ResourceType: "ManagedCluster", Name: "cluster1", Severity: "good", Summary: "cluster joined", Timestamp: "2025-01-01T10:00:00Z"},
		{EventType: "MODIFIED", ResourceType: "Policy", Name: "policy1", Severity: "bad", Summary: "non-compliant", Timestamp: "2025-01-01T11:00:00Z"},
		{EventType: "MODIFIED", ResourceType: "ManagedCluster", Name: "cluster2", Severity: "warning", Summary: "cluster degraded", Timestamp: "2025-01-01T12:00:00Z"},
	}
	for _, e := range events {
		if err := s.Insert(e); err != nil {
			t.Fatal(err)
		}
	}

	// Query all
	results, total, err := s.Query(EventQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// Should be reverse chronological
	if results[0].Name != "cluster2" {
		t.Fatalf("expected first result to be cluster2, got %s", results[0].Name)
	}

	// Query by severity
	results, total, err = s.Query(EventQuery{Severity: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || results[0].Name != "policy1" {
		t.Fatalf("severity filter failed: total=%d", total)
	}

	// Query by resource type
	_, total, err = s.Query(EventQuery{ResourceType: "ManagedCluster"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("resource_type filter failed: total=%d", total)
	}

	// Text search
	results, total, err = s.Query(EventQuery{Query: "degraded"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || results[0].Name != "cluster2" {
		t.Fatalf("text search failed: total=%d", total)
	}
}

func TestPagination(t *testing.T) {
	s := tempDB(t)

	for i := range 10 {
		_ = s.Insert(ws.WatchEvent{
			EventType:    "ADDED",
			ResourceType: "ManagedCluster",
			Name:         "cluster",
			Severity:     "good",
			Timestamp:    time.Date(2025, 1, 1, i, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
	}

	results, total, err := s.Query(EventQuery{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 10 {
		t.Fatalf("expected total 10, got %d", total)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	results, _, err = s.Query(EventQuery{Limit: 3, Offset: 9})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result at offset 9, got %d", len(results))
	}
}

func TestPurge(t *testing.T) {
	s := tempDB(t)

	old := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	recent := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)

	_ = s.Insert(ws.WatchEvent{EventType: "ADDED", ResourceType: "ManagedCluster", Name: "old", Severity: "good", Timestamp: old})
	_ = s.Insert(ws.WatchEvent{EventType: "ADDED", ResourceType: "ManagedCluster", Name: "recent", Severity: "good", Timestamp: recent})

	deleted, err := s.Purge(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	results, total, err := s.Query(EventQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || results[0].Name != "recent" {
		t.Fatalf("purge failed: total=%d", total)
	}
}

func TestGetStats(t *testing.T) {
	s := tempDB(t)

	now := time.Now().UTC()
	events := []ws.WatchEvent{
		{EventType: "ADDED", ResourceType: "ManagedCluster", Name: "c1", Severity: "good", Timestamp: now.Add(-1 * time.Hour).Format(time.RFC3339)},
		{EventType: "MODIFIED", ResourceType: "Policy", Name: "p1", Severity: "bad", Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{EventType: "MODIFIED", ResourceType: "Policy", Name: "p2", Severity: "bad", Timestamp: now.Add(-3 * time.Hour).Format(time.RFC3339)},
		{EventType: "DELETED", ResourceType: "ManagedCluster", Name: "c2", Severity: "warning", Timestamp: now.Add(-4 * time.Hour).Format(time.RFC3339)},
	}
	for _, e := range events {
		_ = s.Insert(e)
	}

	stats, err := s.GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 4 {
		t.Fatalf("expected total 4, got %d", stats.Total)
	}
	if stats.BySeverity["bad"] != 2 {
		t.Fatalf("expected 2 bad, got %d", stats.BySeverity["bad"])
	}
	if stats.ByResourceType["Policy"] != 2 {
		t.Fatalf("expected 2 Policy, got %d", stats.ByResourceType["Policy"])
	}
}

func TestNewEventStoreCreatesFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "test.db")

	// Ensure parent directory exists for SQLite
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := NewEventStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("expected database file to be created")
	}
}
