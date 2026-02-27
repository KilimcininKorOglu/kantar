package audit

import (
	"context"
	"testing"

	"github.com/KilimcininKorOglu/kantar/internal/database"
)

func TestLogEvent(t *testing.T) {
	db := database.NewTestDB(t)
	logger := NewLogger(db.Conn())
	ctx := context.Background()

	event := &Event{
		EventType: EventPackageApprove,
		Actor:     Actor{Username: "admin", IP: "10.0.1.1"},
		Resource:  Resource{Registry: "npm", Package: "express", Version: "4.18.2"},
		Result:    "success",
		Metadata:  map[string]any{"responseTimeMs": 12},
	}

	if err := logger.Log(ctx, event); err != nil {
		t.Fatalf("log: %v", err)
	}

	if event.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if event.PrevHash != "" {
		t.Error("expected empty prev hash for first entry")
	}
}

func TestHashChain(t *testing.T) {
	db := database.NewTestDB(t)
	logger := NewLogger(db.Conn())
	ctx := context.Background()

	// First event
	e1 := &Event{
		EventType: EventUserLogin,
		Actor:     Actor{Username: "admin"},
		Result:    "success",
	}
	logger.Log(ctx, e1)

	// Second event should reference first
	e2 := &Event{
		EventType: EventPackageDownload,
		Actor:     Actor{Username: "dev"},
		Resource:  Resource{Registry: "npm", Package: "lodash"},
		Result:    "success",
	}
	logger.Log(ctx, e2)

	if e2.PrevHash != e1.Hash {
		t.Errorf("expected prev hash %q, got %q", e1.Hash, e2.PrevHash)
	}

	// Third event
	e3 := &Event{
		EventType: EventPackageApprove,
		Actor:     Actor{Username: "admin"},
		Result:    "success",
	}
	logger.Log(ctx, e3)

	if e3.PrevHash != e2.Hash {
		t.Errorf("expected prev hash %q, got %q", e2.Hash, e3.PrevHash)
	}
}

func TestVerify(t *testing.T) {
	db := database.NewTestDB(t)
	logger := NewLogger(db.Conn())
	ctx := context.Background()

	logger.Log(ctx, &Event{
		EventType: EventUserLogin,
		Actor:     Actor{Username: "admin"},
		Result:    "success",
	})

	result, err := logger.Verify(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid chain")
	}
	if result.TotalEntries != 1 {
		t.Errorf("expected 1 entry, got %d", result.TotalEntries)
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	e := &Event{
		EventType: EventPackageDownload,
		Actor:     Actor{Username: "user1"},
		Resource:  Resource{Registry: "npm", Package: "test"},
		Result:    "success",
		PrevHash:  "abc123",
	}
	e.Timestamp = e.Timestamp // ensure zero time

	hash1 := computeHash(e)
	hash2 := computeHash(e)

	if hash1 != hash2 {
		t.Error("expected deterministic hash")
	}
	if len(hash1) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64 char hash, got %d", len(hash1))
	}
}
