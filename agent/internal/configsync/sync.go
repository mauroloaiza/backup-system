// Package configsync polls the BackupSMC server for remote configuration
// changes and applies them to the local agent.yaml.
//
// Design:
//   - The agent remains the source of truth for secrets (passphrase, SFTP
//     password, SMTP creds). Those never travel over the wire.
//   - Server config has a monotonic `version`. Agent stores the last-applied
//     version in a small sidecar file next to the yaml so restarts don't
//     re-pull stale snapshots.
//   - On a new version, we:
//       1. Merge the remote payload into the loaded *config.Config in memory
//          (so the running process picks up live-adjustable fields on next tick).
//       2. Re-serialize the merged yaml to disk (so restarts pick up
//          fields that can't change live, like log format/path).
//       3. Bump the sidecar version.
//
// Live-adjustable fields (in-memory swap takes effect on next backup tick):
//   - backup.source_paths / exclude_patterns
//   - backup.schedule_interval / use_vss / incremental / verify_after_backup
//   - backup.throttle_mbps / pre_script / post_script
//   - retention.* / retry.* / log.level / notify.email toggles
//
// Fields that require a restart (yaml written, but not hot-applied):
//   - destination.* (would require re-opening the backend)
package configsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
)

// Payload mirrors server schemas.NodeConfigPayload (only fields we care about).
type Payload struct {
	SourcePaths             []string    `json:"source_paths"`
	ExcludePatterns         []string    `json:"exclude_patterns"`
	ScheduleIntervalMinutes int         `json:"schedule_interval_minutes"`
	UseVSS                  bool        `json:"use_vss"`
	Incremental             bool        `json:"incremental"`
	VerifyAfterBackup       bool        `json:"verify_after_backup"`
	ThrottleMbps            float64     `json:"throttle_mbps"`
	PreScript               string      `json:"pre_script"`
	PostScript              string      `json:"post_script"`
	RetentionDays           int         `json:"retention_days"`
	GFSEnabled              bool        `json:"gfs_enabled"`
	GFSKeepDaily            int         `json:"gfs_keep_daily"`
	GFSKeepWeekly           int         `json:"gfs_keep_weekly"`
	GFSKeepMonthly          int         `json:"gfs_keep_monthly"`
	RetryMaxAttempts        int         `json:"retry_max_attempts"`
	RetryInitialDelaySecs   int         `json:"retry_initial_delay_seconds"`
	LogLevel                string      `json:"log_level"`
	Email                   EmailToggle `json:"email"`
}

type EmailToggle struct {
	Enabled   bool     `json:"enabled"`
	OnFailure bool     `json:"on_failure"`
	OnSuccess bool     `json:"on_success"`
	To        []string `json:"to"`
}

type pullResponse struct {
	NodeID  string  `json:"node_id"`
	Version int     `json:"version"`
	Payload Payload `json:"payload"`
}

// Syncer polls the server and applies remote config changes.
type Syncer struct {
	serverURL string
	apiToken  string
	nodeID    string
	yamlPath  string
	interval  time.Duration
	log       *zap.Logger

	mu             sync.RWMutex
	cfg            *config.Config // live-adjustable reference; readers use Snapshot()
	currentVersion int
	client         *http.Client
}

// New creates a Syncer. yamlPath is the resolved agent.yaml path (so we can
// write back). If empty, disk writeback is skipped (memory-only apply).
func New(serverURL, apiToken, nodeID, yamlPath string, cfg *config.Config, log *zap.Logger) *Syncer {
	return &Syncer{
		serverURL: serverURL,
		apiToken:  apiToken,
		nodeID:    nodeID,
		yamlPath:  yamlPath,
		interval:  60 * time.Second,
		log:       log,
		cfg:       cfg,
		client:    &http.Client{Timeout: 8 * time.Second},
	}
}

// Snapshot returns a pointer to the (possibly updated) config. The underlying
// struct is swapped atomically on apply — callers should dereference on each
// use rather than caching fields.
func (s *Syncer) Snapshot() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Start runs the poll loop until ctx is cancelled. No-op if serverURL empty.
func (s *Syncer) Start(ctx context.Context) {
	if s.serverURL == "" {
		return
	}
	s.currentVersion = s.readVersionFile()
	s.log.Info("configsync started",
		zap.String("node_id", s.nodeID),
		zap.Int("current_version", s.currentVersion),
		zap.Duration("interval", s.interval),
	)
	go func() {
		// First poll after a short delay so initial register settles.
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if err := s.pollOnce(ctx); err != nil {
					s.log.Debug("configsync poll failed", zap.Error(err))
				}
				timer.Reset(s.interval)
			}
		}
	}()
}

func (s *Syncer) pollOnce(ctx context.Context) error {
	u, err := url.Parse(s.serverURL + "/api/v1/nodes/" + url.PathEscape(s.nodeID) + "/config/pull")
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("current_version", fmt.Sprintf("%d", s.currentVersion))
	u.RawQuery = q.Encode()

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
		return nil // up to date
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var pr pullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if pr.Version <= s.currentVersion {
		return nil
	}

	s.log.Info("configsync applying new version",
		zap.Int("from_version", s.currentVersion),
		zap.Int("to_version", pr.Version),
	)

	s.applyInMemory(pr.Payload)
	if err := s.writeBackYAML(pr.Payload); err != nil {
		s.log.Warn("configsync yaml writeback failed (memory still updated)", zap.Error(err))
	}
	s.currentVersion = pr.Version
	_ = s.writeVersionFile(pr.Version)
	return nil
}

// applyInMemory updates the live *config.Config under lock.
func (s *Syncer) applyInMemory(p Payload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg == nil {
		return
	}
	c := s.cfg

	c.Backup.SourcePaths = p.SourcePaths
	c.Backup.ExcludePatterns = p.ExcludePatterns
	if p.ScheduleIntervalMinutes > 0 {
		c.Backup.ScheduleInterval = time.Duration(p.ScheduleIntervalMinutes) * time.Minute
	}
	c.Backup.UseVSS = p.UseVSS
	c.Backup.Incremental = p.Incremental
	c.Backup.VerifyAfterBackup = p.VerifyAfterBackup
	c.Backup.ThrottleMbps = p.ThrottleMbps
	c.Backup.PreScript = p.PreScript
	c.Backup.PostScript = p.PostScript

	c.Retention.Days = p.RetentionDays
	c.Retention.GFS.Enabled = p.GFSEnabled
	c.Retention.GFS.KeepDaily = p.GFSKeepDaily
	c.Retention.GFS.KeepWeekly = p.GFSKeepWeekly
	c.Retention.GFS.KeepMonthly = p.GFSKeepMonthly

	if p.RetryMaxAttempts > 0 {
		c.Retry.MaxAttempts = p.RetryMaxAttempts
	}
	if p.RetryInitialDelaySecs > 0 {
		c.Retry.InitialDelay = time.Duration(p.RetryInitialDelaySecs) * time.Second
	}

	if p.LogLevel != "" {
		c.Log.Level = p.LogLevel
	}

	c.Notify.Email.Enabled = p.Email.Enabled
	c.Notify.Email.OnFailure = p.Email.OnFailure
	c.Notify.Email.OnSuccess = p.Email.OnSuccess
	if len(p.Email.To) > 0 {
		c.Notify.Email.To = p.Email.To
	}
}

// yamlDocument mirrors just the editable fields of the yaml file.
// We read the existing file, patch only these keys, and write it back,
// so we preserve secrets (encryption_passphrase, sftp_password, smtp creds)
// and any fields we don't manage here.
type yamlDocument map[string]any

// writeBackYAML merges the payload into the existing yaml on disk.
// Preserves any keys we don't touch (secrets, destination details, etc).
func (s *Syncer) writeBackYAML(p Payload) error {
	if s.yamlPath == "" {
		return nil
	}
	raw, err := os.ReadFile(s.yamlPath)
	if err != nil {
		return fmt.Errorf("read yaml: %w", err)
	}
	var doc yamlDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	if doc == nil {
		doc = yamlDocument{}
	}

	backup := ensureMap(doc, "backup")
	backup["source_paths"] = p.SourcePaths
	backup["exclude_patterns"] = p.ExcludePatterns
	if p.ScheduleIntervalMinutes > 0 {
		backup["schedule_interval"] = fmt.Sprintf("%dm", p.ScheduleIntervalMinutes)
	}
	backup["use_vss"] = p.UseVSS
	backup["incremental"] = p.Incremental
	backup["verify_after_backup"] = p.VerifyAfterBackup
	backup["throttle_mbps"] = p.ThrottleMbps
	backup["pre_script"] = p.PreScript
	backup["post_script"] = p.PostScript

	retention := ensureMap(doc, "retention")
	retention["days"] = p.RetentionDays
	gfs := ensureMap(retention, "gfs")
	gfs["enabled"] = p.GFSEnabled
	gfs["keep_daily"] = p.GFSKeepDaily
	gfs["keep_weekly"] = p.GFSKeepWeekly
	gfs["keep_monthly"] = p.GFSKeepMonthly

	retry := ensureMap(doc, "retry")
	if p.RetryMaxAttempts > 0 {
		retry["max_attempts"] = p.RetryMaxAttempts
	}
	if p.RetryInitialDelaySecs > 0 {
		retry["initial_delay"] = fmt.Sprintf("%ds", p.RetryInitialDelaySecs)
	}

	if p.LogLevel != "" {
		log := ensureMap(doc, "log")
		log["level"] = p.LogLevel
	}

	notify := ensureMap(doc, "notify")
	email := ensureMap(notify, "email")
	email["enabled"] = p.Email.Enabled
	email["on_failure"] = p.Email.OnFailure
	email["on_success"] = p.Email.OnSuccess
	if len(p.Email.To) > 0 {
		email["to"] = p.Email.To
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	// Atomic write
	tmp := s.yamlPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.yamlPath)
}

// ensureMap returns doc[key] as a map[string]any, creating/coercing if needed.
// Yaml decodes into map[string]any or map[any]any depending on version; we
// normalize to map[string]any on write.
func ensureMap(doc map[string]any, key string) map[string]any {
	if existing, ok := doc[key]; ok {
		if m, ok := existing.(map[string]any); ok {
			return m
		}
		// Coerce map[any]any (yaml.v3 can produce this if mixed key types)
		if m, ok := existing.(map[any]any); ok {
			out := make(map[string]any, len(m))
			for k, v := range m {
				out[fmt.Sprintf("%v", k)] = v
			}
			doc[key] = out
			return out
		}
	}
	m := map[string]any{}
	doc[key] = m
	return m
}

// ── sidecar version file ─────────────────────────────────────────────────────

func (s *Syncer) versionFilePath() string {
	if s.yamlPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(s.yamlPath), ".backupsmc-config-version")
}

func (s *Syncer) readVersionFile() int {
	p := s.versionFilePath()
	if p == "" {
		return 0
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	var v int
	_, _ = fmt.Sscanf(string(raw), "%d", &v)
	return v
}

func (s *Syncer) writeVersionFile(v int) error {
	p := s.versionFilePath()
	if p == "" {
		return nil
	}
	return os.WriteFile(p, []byte(fmt.Sprintf("%d", v)), 0o600)
}
