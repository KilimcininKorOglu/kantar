package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	gosync "sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

const (
	defaultMaxDepth      = 10
	defaultUpstreamDelay = 50 * time.Millisecond
	queueSize            = 256
)

// DependencyResolver is implemented by plugins that support dependency resolution.
type DependencyResolver interface {
	ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error)
}

// Job represents a sync request for a package and its dependencies.
type Job struct {
	PackageID   int64
	PackageName string
	Version     string
	Ecosystem   registry.EcosystemType
	ApprovedBy  string
	Options     SyncOptions
}

// SyncOptions configures the sync behavior.
type SyncOptions struct {
	MaxDepth        int
	IncludeDev      bool
	IncludeOptional bool
}

// Engine manages async dependency resolution jobs.
type Engine struct {
	queue     chan *jobEntry
	queries   *sqlc.Queries
	db        *sql.DB
	resolvers map[registry.EcosystemType]DependencyResolver
	auditLog  *audit.Logger
	logger    *slog.Logger
	wg        gosync.WaitGroup
}

type jobEntry struct {
	dbJobID int64
	job     *Job
}

// NewEngine creates a new sync engine.
func NewEngine(db *sql.DB, auditLog *audit.Logger, logger *slog.Logger) *Engine {
	return &Engine{
		queue:     make(chan *jobEntry, queueSize),
		queries:   sqlc.New(db),
		db:        db,
		resolvers: make(map[registry.EcosystemType]DependencyResolver),
		auditLog:  auditLog,
		logger:    logger,
	}
}

// RegisterResolver registers a dependency resolver for an ecosystem.
func (e *Engine) RegisterResolver(eco registry.EcosystemType, r DependencyResolver) {
	e.resolvers[eco] = r
}

// Start launches worker goroutines and recovers stale jobs.
func (e *Engine) Start(ctx context.Context, workers int) {
	// Recover stale running jobs from previous crash
	e.queries.MarkStaleSyncJobsFailed(ctx)

	for i := 0; i < workers; i++ {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			for {
				select {
				case entry := <-e.queue:
					e.processSyncJob(ctx, entry)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	e.logger.Info("sync engine started", "workers", workers)
}

// Enqueue creates a sync job in the database and queues it for processing.
// Returns the job ID immediately.
func (e *Engine) Enqueue(ctx context.Context, job *Job) (int64, error) {
	resolver := e.resolvers[job.Ecosystem]
	if resolver == nil {
		return 0, fmt.Errorf("no resolver registered for ecosystem %s", job.Ecosystem)
	}

	dbJob, err := e.queries.CreateSyncJob(ctx, sqlc.CreateSyncJobParams{
		RegistryType: string(job.Ecosystem),
		PackageName:  job.PackageName,
	})
	if err != nil {
		return 0, fmt.Errorf("creating sync job: %w", err)
	}

	entry := &jobEntry{dbJobID: dbJob.ID, job: job}

	select {
	case e.queue <- entry:
		return dbJob.ID, nil
	default:
		// Queue full — mark job as failed
		e.queries.UpdateSyncJobStatus(ctx, sqlc.UpdateSyncJobStatusParams{
			Status:         "failed",
			PackagesSynced: 0,
			ErrorMessage:   "sync queue is full, try again later",
			ID:             dbJob.ID,
		})
		return dbJob.ID, fmt.Errorf("sync queue is full")
	}
}

func (e *Engine) processSyncJob(ctx context.Context, entry *jobEntry) {
	job := entry.job
	jobID := entry.dbJobID

	e.logger.Info("sync job started",
		"jobId", jobID,
		"package", job.PackageName,
		"ecosystem", job.Ecosystem,
	)

	e.queries.UpdateSyncJobStatus(ctx, sqlc.UpdateSyncJobStatusParams{
		Status:         "running",
		PackagesSynced: 0,
		ErrorMessage:   "",
		ID:             jobID,
	})

	maxDepth := job.Options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	visited := make(map[string]struct{})
	synced := int64(0)
	var errors []string

	type treeNode struct {
		name         string
		versionRange string
		depth        int
	}

	queue := []treeNode{{
		name:         job.PackageName,
		versionRange: job.Version,
		depth:        0,
	}}

	resolver := e.resolvers[job.Ecosystem]

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.depth > maxDepth {
			continue
		}

		// Resolve dependencies from upstream
		deps, resolvedVersion, err := resolver.ResolveDependencies(ctx, node.name, node.versionRange)
		if err != nil {
			e.logger.Warn("failed to resolve dependencies",
				"package", node.name,
				"range", node.versionRange,
				"error", err,
			)
			errors = append(errors, fmt.Sprintf("%s@%s: %v", node.name, node.versionRange, err))
			continue
		}

		visitKey := node.name + "@" + resolvedVersion
		if _, seen := visited[visitKey]; seen {
			continue
		}
		visited[visitKey] = struct{}{}

		// Upsert the package in DB and approve it
		pkg, err := e.queries.UpsertPackage(ctx, sqlc.UpsertPackageParams{
			RegistryType: string(job.Ecosystem),
			Name:         node.name,
			RequestedBy:  job.ApprovedBy,
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("upsert %s: %v", node.name, err))
			continue
		}

		// Auto-approve if still pending
		if pkg.Status == "pending" {
			e.queries.UpdatePackageStatus(ctx, sqlc.UpdatePackageStatusParams{
				Status:        "approved",
				ApprovedBy:    job.ApprovedBy + " (auto-sync)",
				BlockedReason: "",
				ID:            pkg.ID,
			})
		}

		// Upsert version
		ver, err := e.queries.UpsertPackageVersion(ctx, sqlc.UpsertPackageVersionParams{
			PackageID: pkg.ID,
			Version:   resolvedVersion,
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("upsert version %s@%s: %v", node.name, resolvedVersion, err))
			continue
		}

		// Insert dependencies
		for _, dep := range deps {
			depOptional := int64(0)
			if dep.Optional {
				depOptional = 1
			}
			depDev := int64(0)
			if dep.Dev {
				depDev = 1
			}
			e.queries.InsertPackageDependency(ctx, sqlc.InsertPackageDependencyParams{
				VersionID:       ver.ID,
				DepName:         dep.Name,
				DepVersionRange: dep.VersionRange,
				DepOptional:     depOptional,
				DepDev:          depDev,
			})
		}

		synced++

		// Update job progress
		e.queries.UpdateSyncJobStatus(ctx, sqlc.UpdateSyncJobStatusParams{
			Status:         "running",
			PackagesSynced: synced,
			ErrorMessage:   "",
			ID:             jobID,
		})

		// Enqueue child dependencies
		for _, dep := range deps {
			if dep.Dev && !job.Options.IncludeDev {
				continue
			}
			if dep.Optional && !job.Options.IncludeOptional {
				continue
			}
			queue = append(queue, treeNode{
				name:         dep.Name,
				versionRange: dep.VersionRange,
				depth:        node.depth + 1,
			})
		}

		// Rate limiting
		time.Sleep(defaultUpstreamDelay)
	}

	// Finalize job
	errMsg := ""
	status := "done"
	if len(errors) > 0 {
		errMsg = fmt.Sprintf("%d errors: %s", len(errors), joinErrors(errors, "; "))
		if synced == 0 {
			status = "failed"
		}
	}

	e.queries.UpdateSyncJobStatus(ctx, sqlc.UpdateSyncJobStatusParams{
		Status:         status,
		PackagesSynced: synced,
		ErrorMessage:   errMsg,
		ID:             jobID,
	})

	if e.auditLog != nil {
		e.auditLog.Log(ctx, &audit.Event{
			EventType: audit.EventRegistrySync,
			Actor:     audit.Actor{Username: job.ApprovedBy},
			Resource:  audit.Resource{Registry: string(job.Ecosystem), Package: job.PackageName},
			Result:    status,
			Metadata:  map[string]any{"synced": synced, "jobId": jobID},
		})
	}

	e.logger.Info("sync job completed",
		"jobId", jobID,
		"package", job.PackageName,
		"synced", synced,
		"status", status,
	)
}

func joinErrors(errs []string, sep string) string {
	if len(errs) > 5 {
		errs = errs[:5]
		errs = append(errs, "...")
	}
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += sep
		}
		result += e
	}
	return result
}
