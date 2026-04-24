package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

// ── YAML config structs (mirrors agent.yaml) ──────────────────────────────────

type rawConfig struct {
	Server      rawServer      `yaml:"server"`
	Backup      rawBackup      `yaml:"backup"`
	Destination rawDestination `yaml:"destination"`
	Retry       rawRetry       `yaml:"retry"`
	Log         rawLog         `yaml:"log"`
}

type rawServer struct {
	URL      string `yaml:"url"`
	APIToken string `yaml:"api_token"`
	Timeout  string `yaml:"timeout"`
}

type rawBackup struct {
	SourcePath           string   `yaml:"source_path"`
	ExcludePatterns      []string `yaml:"exclude_patterns"`
	EncryptionPassphrase string   `yaml:"encryption_passphrase"`
	UseVSS               bool     `yaml:"use_vss"`
	Incremental          bool     `yaml:"incremental"`
	ScheduleInterval     string   `yaml:"schedule_interval"`
	RetentionDays        int      `yaml:"retention_days"`
}

type rawDestination struct {
	Type      string `yaml:"type"`
	LocalPath string `yaml:"local_path"`
	S3Bucket  string `yaml:"s3_bucket"`
	S3Region  string `yaml:"s3_region"`
	S3Prefix  string `yaml:"s3_prefix"`
	SFTPHost  string `yaml:"sftp_host"`
	SFTPPath  string `yaml:"sftp_path"`
	SFTPUser  string `yaml:"sftp_user"`
}

type rawRetry struct {
	MaxAttempts  int    `yaml:"max_attempts"`
	InitialDelay string `yaml:"initial_delay"`
}

type rawLog struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Path   string `yaml:"path"`
}

// ── Types exposed to frontend ────────────────────────────────────────────────

type UIConfig struct {
	// Server
	ServerURL string `json:"server_url"`
	APIToken  string `json:"api_token"`
	// Backup
	SourcePaths          []string `json:"source_paths"`
	ExcludePatterns      []string `json:"exclude_patterns"`
	EncryptionPassphrase string   `json:"encryption_passphrase"`
	UseVSS               bool     `json:"use_vss"`
	Incremental          bool     `json:"incremental"`
	ScheduleInterval     string   `json:"schedule_interval"`
	RetentionDays        int      `json:"retention_days"`
	// Destination
	DestType  string `json:"dest_type"`
	LocalPath string `json:"local_path"`
	S3Bucket  string `json:"s3_bucket"`
	S3Region  string `json:"s3_region"`
	S3Prefix  string `json:"s3_prefix"`
	SFTPHost  string `json:"sftp_host"`
	SFTPPath  string `json:"sftp_path"`
	SFTPUser  string `json:"sftp_user"`
	// Log
	LogLevel string `json:"log_level"`
	LogPath  string `json:"log_path"`
	// Meta
	ConfigPath string `json:"config_path"`
}

type ServiceStatus struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
}

type BackupStats struct {
	LastRun      string `json:"last_run"`
	NextRun      string `json:"next_run"`
	LastJobID    string `json:"last_job_id"`
	LastFiles    int    `json:"last_files"`
	LastBytes    int64  `json:"last_bytes"`
	LastDuration string `json:"last_duration"`
	LastErrors   int    `json:"last_errors"`
	TotalJobs    int    `json:"total_jobs"`
}

// ── App ──────────────────────────────────────────────────────────────────────

type App struct {
	ctx        context.Context
	configPath string
}

func NewApp() *App {
	return &App{
		configPath: resolveConfigPath(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startTray()
}

func (a *App) shutdown(ctx context.Context) {}

// ── Config ───────────────────────────────────────────────────────────────────

func resolveConfigPath() string {
	candidates := []string{
		filepath.Join(os.Getenv("PROGRAMDATA"), "BackupSMC", "agent.yaml"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "BackupSMC", "Agent", "agent.yaml"),
		"agent.yaml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func (a *App) GetConfigPath() string { return a.configPath }

func (a *App) SetConfigPath(path string) { a.configPath = path }

func (a *App) GetConfig() UIConfig {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return UIConfig{ConfigPath: a.configPath}
	}
	var raw rawConfig
	_ = yaml.Unmarshal(data, &raw)

	sourcePaths := []string{}
	if raw.Backup.SourcePath != "" {
		sourcePaths = []string{raw.Backup.SourcePath}
	}

	return UIConfig{
		ServerURL:            raw.Server.URL,
		APIToken:             raw.Server.APIToken,
		SourcePaths:          sourcePaths,
		ExcludePatterns:      raw.Backup.ExcludePatterns,
		EncryptionPassphrase: raw.Backup.EncryptionPassphrase,
		UseVSS:               raw.Backup.UseVSS,
		Incremental:          raw.Backup.Incremental,
		ScheduleInterval:     raw.Backup.ScheduleInterval,
		RetentionDays:        raw.Backup.RetentionDays,
		DestType:             raw.Destination.Type,
		LocalPath:            raw.Destination.LocalPath,
		S3Bucket:             raw.Destination.S3Bucket,
		S3Region:             raw.Destination.S3Region,
		S3Prefix:             raw.Destination.S3Prefix,
		SFTPHost:             raw.Destination.SFTPHost,
		SFTPPath:             raw.Destination.SFTPPath,
		SFTPUser:             raw.Destination.SFTPUser,
		LogLevel:             raw.Log.Level,
		LogPath:              raw.Log.Path,
		ConfigPath:           a.configPath,
	}
}

func (a *App) SaveConfig(cfg UIConfig) string {
	raw := rawConfig{
		Server: rawServer{
			URL:      cfg.ServerURL,
			APIToken: cfg.APIToken,
			Timeout:  "30s",
		},
		Backup: rawBackup{
			SourcePath:           firstOf(cfg.SourcePaths),
			ExcludePatterns:      cfg.ExcludePatterns,
			EncryptionPassphrase: cfg.EncryptionPassphrase,
			UseVSS:               cfg.UseVSS,
			Incremental:          cfg.Incremental,
			ScheduleInterval:     cfg.ScheduleInterval,
			RetentionDays:        cfg.RetentionDays,
		},
		Destination: rawDestination{
			Type:      cfg.DestType,
			LocalPath: cfg.LocalPath,
			S3Bucket:  cfg.S3Bucket,
			S3Region:  cfg.S3Region,
			S3Prefix:  cfg.S3Prefix,
			SFTPHost:  cfg.SFTPHost,
			SFTPPath:  cfg.SFTPPath,
			SFTPUser:  cfg.SFTPUser,
		},
		Retry: rawRetry{MaxAttempts: 3, InitialDelay: "1s"},
		Log: rawLog{
			Level:  cfg.LogLevel,
			Format: "console",
			Path:   cfg.LogPath,
		},
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return "Error serializing config: " + err.Error()
	}

	// Try direct write first
	if err := os.WriteFile(a.configPath, data, 0o644); err != nil {
		// Fall back: write to C:\Temp (always writable), then copy via elevated PS1 script
		tmp := `C:\Temp\bsmc_cfg_save.yaml`
		ps1 := `C:\Temp\bsmc_cfg_save.ps1`
		_ = os.MkdirAll(`C:\Temp`, 0o755)
		if writeErr := os.WriteFile(tmp, data, 0o644); writeErr != nil {
			return "Error writing temp config: " + writeErr.Error()
		}
		scriptBody := fmt.Sprintf("Copy-Item '%s' '%s' -Force", tmp, a.configPath)
		if writeErr := os.WriteFile(ps1, []byte(scriptBody), 0o644); writeErr != nil {
			return "Error writing save script: " + writeErr.Error()
		}
		out, elErr := hiddenCmd("powershell", "-Command",
			fmt.Sprintf("Start-Process powershell -Verb RunAs -Wait -ArgumentList '-NoProfile -ExecutionPolicy Bypass -File %s'", ps1),
		).CombinedOutput()
		if elErr != nil {
			return "Error saving (need admin): " + string(out)
		}
	}

	return ""
}

func firstOf(ss []string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// ── Service control ──────────────────────────────────────────────────────────

func (a *App) GetServiceStatus() ServiceStatus {
	out, _ := hiddenCmd("sc.exe", "query", "BackupSMC").CombinedOutput()
	s := string(out)
	if strings.Contains(s, "1060") || strings.Contains(s, "FAILED") {
		return ServiceStatus{Installed: false, Running: false, Status: "Not installed"}
	}
	running := strings.Contains(s, "RUNNING")
	status := "Stopped"
	if running {
		status = "Running"
	}
	return ServiceStatus{Installed: true, Running: running, Status: status}
}

func (a *App) StartService() string {
	return runSC("start", "BackupSMC")
}

func (a *App) StopService() string {
	return runSC("stop", "BackupSMC")
}

func runSC(args ...string) string {
	out, err := hiddenCmd("sc.exe", args...).CombinedOutput()
	if err != nil {
		// Try elevated
		script := fmt.Sprintf("sc.exe %s", strings.Join(args, " "))
		hiddenCmd("powershell", "-Command",
			fmt.Sprintf("Start-Process powershell -Verb RunAs -Wait -ArgumentList '-NoProfile -Command \"%s\"'", script),
		).Run()
		return ""
	}
	_ = out
	return ""
}

// ── Run backup now ───────────────────────────────────────────────────────────

func (a *App) RunBackupNow() string {
	agentExe := filepath.Join(filepath.Dir(a.configPath), "backupsmc-agent.exe")
	if _, err := os.Stat(agentExe); err != nil {
		// Fallback: look in Program Files
		agentExe = filepath.Join(os.Getenv("PROGRAMFILES"), "BackupSMC", "Agent", "backupsmc-agent.exe")
	}
	go func() {
		out, err := hiddenCmd(agentExe, "run", "-c", a.configPath).CombinedOutput()
		msg := string(out)
		if err != nil {
			msg = "Error: " + err.Error() + "\n" + msg
		}
		wailsruntime.EventsEmit(a.ctx, "backup:done", msg)
	}()
	return "started"
}

// ── Logs ─────────────────────────────────────────────────────────────────────

func (a *App) GetLogs(n int) []string {
	cfg := a.GetConfig()
	logPath := cfg.LogPath
	if logPath == "" || logPath == "stdout" {
		return []string{"Log output goes to stdout (not a file). Change log.path in config."}
	}

	f, err := os.Open(logPath)
	if err != nil {
		return []string{"Cannot open log file: " + err.Error()}
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// ── Stats ────────────────────────────────────────────────────────────────────

func (a *App) GetStats() BackupStats {
	cfg := a.GetConfig()
	destPath := cfg.LocalPath
	if destPath == "" {
		return BackupStats{}
	}

	jobsDir := filepath.Join(destPath, "jobs")
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		return BackupStats{}
	}

	totalJobs := 0
	var runs []struct {
		jobID string
		ts    time.Time
		path  string
	}

	for _, jobEntry := range entries {
		if !jobEntry.IsDir() {
			continue
		}
		tsEntries, err := os.ReadDir(filepath.Join(jobsDir, jobEntry.Name()))
		if err != nil {
			continue
		}
		for _, tsEntry := range tsEntries {
			if !tsEntry.IsDir() {
				continue
			}
			t, err := time.Parse("20060102T150405Z", tsEntry.Name())
			if err != nil {
				continue
			}
			totalJobs++
			runs = append(runs, struct {
				jobID string
				ts    time.Time
				path  string
			}{jobEntry.Name(), t, filepath.Join(jobsDir, jobEntry.Name(), tsEntry.Name())})
		}
	}

	if len(runs) == 0 {
		return BackupStats{TotalJobs: totalJobs}
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].ts.After(runs[j].ts) })
	last := runs[0]

	// Count files and bytes in last run
	var fileCount int
	var totalBytes int64
	_ = filepath.Walk(filepath.Join(last.path, "data"), func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			fileCount++
			totalBytes += info.Size()
		}
		return nil
	})

	nextRun := "—"
	interval := cfg.ScheduleInterval
	if interval != "" {
		d, err := time.ParseDuration(interval)
		if err == nil {
			next := last.ts.Add(d)
			if next.After(time.Now()) {
				nextRun = next.Format("02 Jan 15:04")
			} else {
				nextRun = "Soon"
			}
		}
	}

	return BackupStats{
		LastRun:   last.ts.Format("02 Jan 2006 15:04"),
		NextRun:   nextRun,
		LastJobID: last.jobID[:8] + "...",
		LastFiles: fileCount,
		LastBytes: totalBytes,
		TotalJobs: totalJobs,
	}
}

// ── Folder picker ────────────────────────────────────────────────────────────

func (a *App) BrowseFolder() string {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select folder",
	})
	if err != nil {
		return ""
	}
	return dir
}
