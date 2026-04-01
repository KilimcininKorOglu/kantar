// Package audit provides structured audit logging with hash-chain tamper detection.
package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

// EventType defines the type of audit event.
type EventType string

const (
	EventPackageDownload EventType = "package.download"
	EventPackageUpload   EventType = "package.upload"
	EventPackageDelete   EventType = "package.delete"
	EventPackageApprove  EventType = "package.approve"
	EventPackageBlock    EventType = "package.block"
	EventPolicyViolation EventType = "policy.violation"
	EventPolicyUpdate    EventType = "policy.update"
	EventUserLogin       EventType = "user.login"
	EventUserCreate      EventType = "user.create"
	EventUserTokenCreate EventType = "user.token.create"
	EventRegistrySync    EventType = "registry.sync"
	EventRegistryConfig  EventType = "registry.config.update"
	EventSystemGC        EventType = "system.gc"
	EventSystemBackup    EventType = "system.backup"
)

// Actor identifies who performed the action.
type Actor struct {
	Username  string `json:"username"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

// Resource identifies the affected resource.
type Resource struct {
	Registry string `json:"registry,omitempty"`
	Package  string `json:"package,omitempty"`
	Version  string `json:"version,omitempty"`
}

// Event represents a single audit log entry.
type Event struct {
	Timestamp time.Time      `json:"timestamp"`
	EventType EventType      `json:"event"`
	Actor     Actor          `json:"actor"`
	Resource  Resource       `json:"resource"`
	Result    string         `json:"result"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	PrevHash  string         `json:"prevHash"`
	Hash      string         `json:"hash"`
}

// Logger handles writing and querying audit logs.
type Logger struct {
	mu       sync.Mutex
	queries  *sqlc.Queries
	prevHash string
}

// NewLogger creates a new audit Logger.
func NewLogger(db *sql.DB) *Logger {
	q := sqlc.New(db)

	// Load the last hash for chain continuity
	lastLog, err := q.GetLatestAuditLog(context.Background())
	prevHash := ""
	if err == nil {
		prevHash = lastLog.Hash
	}

	return &Logger{
		queries:  q,
		prevHash: prevHash,
	}
}

// Log records an audit event.
func (l *Logger) Log(ctx context.Context, event *Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	event.Timestamp = time.Now().UTC()
	event.PrevHash = l.prevHash
	event.Hash = computeHash(event)

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = l.queries.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		Event:            string(event.EventType),
		ActorUsername:    event.Actor.Username,
		ActorIp:          event.Actor.IP,
		ActorUserAgent:   event.Actor.UserAgent,
		ResourceRegistry: event.Resource.Registry,
		ResourcePackage:  event.Resource.Package,
		ResourceVersion:  event.Resource.Version,
		Result:           event.Result,
		MetadataJson:     string(metadataJSON),
		PrevHash:         event.PrevHash,
		Hash:             event.Hash,
	})
	if err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	l.prevHash = event.Hash
	return nil
}

// computeHash generates SHA-256 hash for the event (hash-chain).
func computeHash(event *Event) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.Actor.Username,
		event.Resource.Registry,
		event.Resource.Package,
		event.Resource.Version,
		event.Result,
		event.PrevHash,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// VerifyResult holds the result of chain verification.
type VerifyResult struct {
	Valid        bool   `json:"valid"`
	TotalEntries int64  `json:"totalEntries"`
	ErrorAt      int64  `json:"errorAt,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// Verify checks the integrity of the audit log hash chain by iterating
// all entries in batches and recomputing each hash from the previous entry.
func (l *Logger) Verify(ctx context.Context) (*VerifyResult, error) {
	count, err := l.queries.CountAuditLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting audit logs: %w", err)
	}

	result := &VerifyResult{
		Valid:        true,
		TotalEntries: count,
	}

	if count == 0 {
		return result, nil
	}

	const batchSize int64 = 500
	var prevHash string
	var checked int64

	for offset := int64(0); offset < count; offset += batchSize {
		logs, err := l.queries.ListAuditLogsAsc(ctx, sqlc.ListAuditLogsAscParams{
			Limit:  batchSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching audit logs at offset %d: %w", offset, err)
		}

		for _, entry := range logs {
			checked++

			// Verify prev_hash chain link
			if entry.PrevHash != prevHash {
				result.Valid = false
				result.ErrorAt = checked
				result.ErrorMessage = fmt.Sprintf("chain broken at entry %d: expected prev_hash %q, got %q", entry.ID, prevHash, entry.PrevHash)
				return result, nil
			}

			// Recompute hash and verify
			recomputed := computeHash(&Event{
				Timestamp: entry.Timestamp,
				EventType: EventType(entry.Event),
				Actor: Actor{
					Username:  entry.ActorUsername,
					IP:        entry.ActorIp,
					UserAgent: entry.ActorUserAgent,
				},
				Resource: Resource{
					Registry: entry.ResourceRegistry,
					Package:  entry.ResourcePackage,
					Version:  entry.ResourceVersion,
				},
				Result:   entry.Result,
				PrevHash: entry.PrevHash,
			})

			if entry.Hash != recomputed {
				result.Valid = false
				result.ErrorAt = checked
				result.ErrorMessage = fmt.Sprintf("hash mismatch at entry %d: stored %q, computed %q", entry.ID, entry.Hash, recomputed)
				return result, nil
			}

			prevHash = entry.Hash
		}
	}

	return result, nil
}
