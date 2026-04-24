// Package restoresync polls the BackupSMC server for queued restore requests
// and executes them locally via the embedded restore engine.
//
// Design mirrors configsync: a single goroutine, 60s interval, non-blocking
// failures just get logged.
//
// Server contract:
//   - GET  /api/v1/nodes/{id}/restore/pending  → 204 if idle, else {id,source_job_id,target_path,filter_pattern,dry_run}
//   - POST /api/v1/restore/{id}/progress       → {status, message, files_restored, bytes_restored}
//
// The server atomically flips the returned row to "running" when we pull it,
// so we own it from that moment. If the agent crashes mid-restore the row
// stays "running" server-side — operator can cancel+recreate (v0.11.2 could
// add lease expiry).
package restoresync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/restore"
	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/factory"
)

// PendingRestore matches server schemas.RestorePendingOut.
type PendingRestore struct {
	ID            int    `json:"id"`
	SourceJobID   string `json:"source_job_id"`
	TargetPath    string `json:"target_path"`
	FilterPattern string `json:"filter_pattern"`
	DryRun        bool   `json:"dry_run"`
}

// progressReport matches server schemas.RestoreProgressUpdate.
type progressReport struct {
	Status         string `json:"status"` // running | completed | failed
	Message        string `json:"message,omitempty"`
	FilesRestored  int    `json:"files_restored"`
	BytesRestored  int64  `json:"bytes_restored"`
}

// Syncer polls the server and executes restores.
type Syncer struct {
	serverURL string
	apiToken  string
	nodeID    string
	interval  time.Duration
	log       *zap.Logger

	// cfg is a live pointer (possibly swapped by configsync). We read
	// Backup.EncryptionPassphrase and the Destination block at execution
	// time, so config updates are honored.
	cfg *config.Config

	mu     sync.Mutex // serialises restore executions (one at a time)
	client *http.Client
}

// New creates a Syncer. Call Start to begin polling.
func New(serverURL, apiToken, nodeID string, cfg *config.Config, log *zap.Logger) *Syncer {
	return &Syncer{
		serverURL: serverURL,
		apiToken:  apiToken,
		nodeID:    nodeID,
		interval:  60 * time.Second,
		log:       log,
		cfg:       cfg,
		// Restore can take a while; keep the poll+report client snappy but
		// not cruel. The restore itself bypasses this HTTP client.
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Start runs the poll loop until ctx is cancelled. No-op if serverURL empty.
func (s *Syncer) Start(ctx context.Context) {
	if s.serverURL == "" {
		return
	}
	s.log.Info("restoresync started",
		zap.String("node_id", s.nodeID),
		zap.Duration("interval", s.interval),
	)
	go func() {
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if err := s.pollOnce(ctx); err != nil {
					s.log.Debug("restoresync poll failed", zap.Error(err))
				}
				timer.Reset(s.interval)
			}
		}
	}()
}

// pollOnce fetches one pending restore (if any) and executes it.
func (s *Syncer) pollOnce(ctx context.Context) error {
	u, err := url.Parse(s.serverURL + "/api/v1/nodes/" + url.PathEscape(s.nodeID) + "/restore/pending")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Agent-Token", s.apiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var pr PendingRestore
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// Serialise restores — agent code isn't designed for concurrent restores
	// against the same destination.
	if !s.mu.TryLock() {
		s.log.Warn("restoresync: another restore is in progress, skipping this tick",
			zap.Int("restore_id", pr.ID),
		)
		// We already took ownership from the queue; report failure so the
		// user isn't stuck waiting indefinitely.
		_ = s.report(ctx, pr.ID, "failed", "agent busy with another restore", 0, 0)
		return nil
	}
	defer s.mu.Unlock()

	s.runRestore(ctx, pr)
	return nil
}

// runRestore executes the restore and reports progress + terminal status.
func (s *Syncer) runRestore(ctx context.Context, pr PendingRestore) {
	s.log.Info("restoresync executing",
		zap.Int("restore_id", pr.ID),
		zap.String("source_job_id", pr.SourceJobID),
		zap.String("target", pr.TargetPath),
		zap.String("filter", pr.FilterPattern),
		zap.Bool("dry_run", pr.DryRun),
	)

	// Report "running" (server already set it, but re-assert so the message is populated).
	_ = s.report(ctx, pr.ID, "running", "agente iniciando restauración", 0, 0)

	if s.cfg == nil {
		_ = s.report(ctx, pr.ID, "failed", "agente sin configuración cargada", 0, 0)
		return
	}

	dest, err := factory.New(s.cfg)
	if err != nil {
		s.log.Error("restoresync: open destination", zap.Error(err))
		_ = s.report(ctx, pr.ID, "failed", fmt.Sprintf("destino: %v", err), 0, 0)
		return
	}
	defer dest.Close()

	engine := restore.New(dest, s.log)
	result, err := engine.Run(ctx, restore.Options{
		JobID:             pr.SourceJobID,
		TargetPath:        pr.TargetPath,
		Passphrase:        s.cfg.Backup.EncryptionPassphrase,
		Filter:            pr.FilterPattern,
		OverwriteExisting: false,
		DryRun:            pr.DryRun,
		RestoreACLs:       true,
	})

	if err != nil {
		s.log.Error("restoresync: restore failed", zap.Error(err))
		_ = s.report(ctx, pr.ID, "failed", err.Error(), 0, 0)
		return
	}

	// Compute restored bytes from manifest entries that were actually restored.
	// The restore engine doesn't expose byte counts today; approximate with file count.
	// (v0.11.2 could surface bytes from FileResult.)
	msg := fmt.Sprintf("restaurados=%d omitidos=%d errores=%d duración=%.1fs",
		result.RestoredFiles, result.SkippedFiles, result.ErrorFiles,
		result.FinishedAt.Sub(result.StartedAt).Seconds())
	if pr.DryRun {
		msg = "[DRY-RUN] " + msg
	}

	status := "completed"
	if result.ErrorFiles > 0 {
		status = "failed"
		msg = fmt.Sprintf("%d archivo(s) con error — %s", result.ErrorFiles, msg)
	}

	if err := s.report(ctx, pr.ID, status, msg, result.RestoredFiles, 0); err != nil {
		s.log.Warn("restoresync: report failed (restore itself was OK)", zap.Error(err))
	}
}

// report POSTs a progress update to /restore/{id}/progress.
func (s *Syncer) report(ctx context.Context, id int, status, message string, files int, bytes int64) error {
	body := progressReport{
		Status:        status,
		Message:       message,
		FilesRestored: files,
		BytesRestored: bytes,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	u := s.serverURL + "/api/v1/restore/" + fmt.Sprintf("%d", id) + "/progress"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes2reader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", s.apiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("report: server returned %d", resp.StatusCode)
	}
	return nil
}

// bytes2reader wraps a byte slice as an io.Reader without importing "bytes"
// in the hot section of this file (kept inline for readability).
func bytes2reader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
