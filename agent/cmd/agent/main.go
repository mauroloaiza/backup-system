package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	backup "github.com/smcsoluciones/backup-system/agent/internal/backup"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/retention"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/restore"
	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/configsync"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/factory"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/local"
	"github.com/smcsoluciones/backup-system/agent/internal/noderegister"
	"github.com/smcsoluciones/backup-system/agent/internal/notify"
	"github.com/smcsoluciones/backup-system/agent/internal/reporter"
	"github.com/smcsoluciones/backup-system/agent/internal/winsvc"
)

var (
	cfgFile string
	nodeID  string
)

func main() {
	// When launched by Windows SCM, Cobra flag parsing is skipped.
	// Pre-parse -c / --config from os.Args so setup() finds the right file.
	if winsvc.IsRunningAsService() {
		args := os.Args[1:]
		for i, a := range args {
			if (a == "-c" || a == "--config") && i+1 < len(args) {
				cfgFile = args[i+1]
				break
			}
		}
		if err := runServiceMode(); err != nil {
			os.Exit(1)
		}
		return
	}

	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "backupsmc-agent",
		Short: "BackupSMC Agent — enterprise backup agent for Windows servers",
		Long: `BackupSMC Agent performs encrypted, incremental backups of Windows
file systems with VSS (Volume Shadow Copy) support.`,
	}

	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (YAML)")
	root.PersistentFlags().StringVar(&nodeID, "node-id", "", "unique node identifier (default: hostname)")

	root.AddCommand(
		runCmd(),
		serviceCmd(),
		installServiceCmd(),
		uninstallServiceCmd(),
		startServiceCmd(),
		stopServiceCmd(),
		restoreCmd(),
		validateCmd(),
		versionCmd(),
	)

	return root
}

// ── run ───────────────────────────────────────────────────────────────────────

func runCmd() *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a single backup job",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := setup()
			if err != nil {
				return err
			}
			defer log.Sync() //nolint:errcheck

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return runOnce(ctx, cfg, log, jobID)
		},
	}
	cmd.Flags().StringVar(&jobID, "job-id", "", "job ID (generated if empty)")
	return cmd
}

// ── service ───────────────────────────────────────────────────────────────────

func serviceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "service",
		Short: "Run as a background service (scheduler loop)",
		Long:  "Runs the backup agent in scheduler mode — executes backups on the configured interval.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if winsvc.IsRunningAsService() {
				return runServiceMode()
			}
			return runSchedulerLoop()
		},
	}
}

// runSchedulerLoop runs the backup on the configured interval until SIGTERM.
func runSchedulerLoop() error {
	cfg, log, err := setup()
	if err != nil {
		return err
	}
	defer log.Sync() //nolint:errcheck

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	interval := cfg.Backup.ScheduleInterval
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	retentionDesc := fmt.Sprintf("%d days", cfg.Retention.Days)
	if cfg.Retention.GFS.Enabled {
		retentionDesc = fmt.Sprintf("GFS (daily=%d weekly=%d monthly=%d)",
			cfg.Retention.GFS.KeepDaily,
			cfg.Retention.GFS.KeepWeekly,
			cfg.Retention.GFS.KeepMonthly,
		)
	}
	log.Info("scheduler started",
		zap.Duration("interval", interval),
		zap.String("retention", retentionDesc),
	)

	// Register node + start periodic heartbeat (every 5 min).
	noderegister.StartHeartbeat(ctx, cfg.Server.URL, cfg.Server.APIToken,
		resolveNodeID(), cfg.EffectiveSourcePaths(),
		noderegister.BuildDestinations(cfg.Destination))

	// Start remote config syncer (polls /nodes/{id}/config/pull every 60s).
	// Mutations to cfg happen under the Syncer's lock; subsequent backup
	// runs see the updated values on the next tick.
	syncer := configsync.New(cfg.Server.URL, cfg.Server.APIToken,
		resolveNodeID(), resolvedConfigPath(), cfg, log)
	syncer.Start(ctx)

	// Run immediately on start.
	if err := runOnce(ctx, cfg, log, ""); err != nil {
		log.Error("backup run failed", zap.Error(err))
	}
	runRetention(cfg, log)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("scheduler stopped")
			return nil
		case <-ticker.C:
			if err := runOnce(ctx, cfg, log, ""); err != nil {
				log.Error("scheduled backup failed", zap.Error(err))
			}
			runRetention(cfg, log)
		}
	}
}

// runServiceMode wraps the scheduler loop inside the Windows SCM handler.
func runServiceMode() error {
	_, log, err := setup()
	if err != nil {
		return err
	}
	defer log.Sync() //nolint:errcheck

	_, cancel := context.WithCancel(context.Background())

	handler := &winsvc.Handler{
		Run:  func() error { return runSchedulerLoop() },
		Stop: cancel,
	}
	return winsvc.RunAsService(handler)
}

// ── install-service ───────────────────────────────────────────────────────────

func installServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-service",
		Short: "Install BackupSMC as a Windows service (run as Administrator)",
		RunE: func(cmd *cobra.Command, args []string) error {
			exePath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("cannot determine exe path: %w", err)
			}
			exePath, _ = filepath.Abs(exePath)

			cfgAbs := cfgFile
			if cfgAbs == "" {
				cfgAbs = filepath.Join(filepath.Dir(exePath), "agent.yaml")
			}
			cfgAbs, _ = filepath.Abs(cfgAbs)

			if err := winsvc.Install(exePath, cfgAbs); err != nil {
				return err
			}
			fmt.Printf("✓ Service %q installed\n", winsvc.ServiceName)
			fmt.Printf("  Binary : %s\n", exePath)
			fmt.Printf("  Config : %s\n", cfgAbs)
			fmt.Println("\nStart it with:")
			fmt.Printf("  backupsmc-agent.exe start-service\n")
			fmt.Printf("  — or —\n")
			fmt.Printf("  sc start %s\n", winsvc.ServiceName)
			return nil
		},
	}
}

func uninstallServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-service",
		Short: "Remove the BackupSMC Windows service (run as Administrator)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := winsvc.Uninstall(); err != nil {
				return err
			}
			fmt.Printf("✓ Service %q removed\n", winsvc.ServiceName)
			return nil
		},
	}
}

func startServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start-service",
		Short: "Start the BackupSMC Windows service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := winsvc.Start(); err != nil {
				return err
			}
			fmt.Printf("✓ Service %q started\n", winsvc.ServiceName)
			return nil
		},
	}
}

func stopServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop-service",
		Short: "Stop the BackupSMC Windows service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := winsvc.Stop(); err != nil {
				return err
			}
			fmt.Printf("✓ Service %q stopped\n", winsvc.ServiceName)
			return nil
		},
	}
}

// ── restore ───────────────────────────────────────────────────────────────────

func restoreCmd() *cobra.Command {
	var (
		jobID      string
		targetPath string
		passphrase string
		filter     string
		overwrite  bool
		dryRun     bool
		restoreACL bool
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore files from a backup job",
		Example: `  backupsmc-agent restore -c agent.yaml --job-id <id> --target C:\Restore
  backupsmc-agent restore -c agent.yaml --job-id <id> --target C:\Restore --filter "docs/*"
  backupsmc-agent restore -c agent.yaml --job-id <id> --target C:\Restore --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := setup()
			if err != nil {
				return err
			}
			defer log.Sync() //nolint:errcheck

			if passphrase == "" {
				passphrase = cfg.Backup.EncryptionPassphrase
			}

			dest, err := local.New(cfg.Destination.LocalPath)
			if err != nil {
				return fmt.Errorf("destination: %w", err)
			}
			defer dest.Close()

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			engine := restore.New(dest, log)
			result, err := engine.Run(ctx, restore.Options{
				JobID:             jobID,
				TargetPath:        targetPath,
				Passphrase:        passphrase,
				Filter:            filter,
				OverwriteExisting: overwrite,
				DryRun:            dryRun,
				RestoreACLs:       restoreACL,
			})
			if err != nil {
				return err
			}

			prefix := ""
			if dryRun {
				prefix = "[DRY RUN] "
			}
			fmt.Printf("\n%sRestore: %d restaurados, %d omitidos, %d errores — %.2fs\n",
				prefix, result.RestoredFiles, result.SkippedFiles, result.ErrorFiles,
				result.FinishedAt.Sub(result.StartedAt).Seconds(),
			)
			if result.ErrorFiles > 0 {
				return fmt.Errorf("%d archivo(s) no se pudieron restaurar", result.ErrorFiles)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "job ID a restaurar (requerido)")
	cmd.Flags().StringVar(&targetPath, "target", "", "directorio destino (requerido)")
	cmd.Flags().StringVar(&passphrase, "passphrase", "", "passphrase de descifrado (default: desde config)")
	cmd.Flags().StringVar(&filter, "filter", "", "patrón glob para restaurar subconjunto (ej. 'docs/*')")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "sobreescribir archivos existentes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "listar qué se restauraría sin escribir nada")
	cmd.Flags().BoolVar(&restoreACL, "restore-acl", true, "restaurar ACLs de Windows")
	_ = cmd.MarkFlagRequired("job-id")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

// ── validate / version ────────────────────────────────────────────────────────

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the agent configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := setup()
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "Configuración inválida:\n  %v\n", err)
				return err
			}
			fmt.Println("✓ Configuración válida")
			for i, p := range cfg.EffectiveSourcePaths() {
				if i == 0 {
					fmt.Printf("  Fuente       : %s\n", p)
				} else {
					fmt.Printf("               + %s\n", p)
				}
			}
			destLabel := cfg.Destination.LocalPath
			if cfg.Destination.Type == "s3" {
				destLabel = "s3://" + cfg.Destination.S3Bucket + "/" + cfg.Destination.S3Prefix
			}
			fmt.Printf("  Destino      : %s (%s)\n", destLabel, cfg.Destination.Type)
			fmt.Printf("  Intervalo    : %s\n", cfg.Backup.ScheduleInterval)
			if cfg.Retention.GFS.Enabled {
				fmt.Printf("  Retención    : GFS (daily=%d weekly=%d monthly=%d)\n",
					cfg.Retention.GFS.KeepDaily,
					cfg.Retention.GFS.KeepWeekly,
					cfg.Retention.GFS.KeepMonthly,
				)
			} else {
				fmt.Printf("  Retención    : %d días\n", cfg.Retention.Days)
			}
			fmt.Printf("  VSS          : %v\n", cfg.Backup.UseVSS)
			fmt.Printf("  Incremental  : %v\n", cfg.Backup.Incremental)
			fmt.Printf("  Throttle     : %.1f MB/s\n", cfg.Backup.ThrottleMbps)
			fmt.Printf("  Verify       : %v\n", cfg.Backup.VerifyAfterBackup)
			return nil
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print agent version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("BackupSMC Agent v0.1.0")
			fmt.Println("SMC Soluciones — https://smcsoluciones.com")
		},
	}
}

// ── internal helpers ──────────────────────────────────────────────────────────

func runOnce(ctx context.Context, cfg *config.Config, log *zap.Logger, jobID string) error {
	if jobID == "" {
		jobID = uuid.New().String()
	}

	dest, err := factory.New(cfg)
	if err != nil {
		return fmt.Errorf("destination: %w", err)
	}
	defer dest.Close()

	rep := reporter.New(cfg.Server.URL, cfg.Server.APIToken, 5*time.Second, log)
	defer rep.Stop()

	notifier := notify.New(cfg.Notify, log)

	engine := backup.New(cfg, dest, rep, log, resolveNodeID())
	result, err := engine.Run(ctx, jobID)

	if err != nil {
		notify.WriteEventLog("error", fmt.Sprintf("Backup FAILED job=%s: %v", jobID, err))
		notifier.Notify(notify.BackupEvent{
			JobID:     jobID,
			NodeID:    resolveNodeID(),
			Status:    "failed",
			Errors:    []string{err.Error()},
			StartedAt: time.Now().UTC(),
		})
		return err
	}

	status := "completed"
	if len(result.Errors) > 0 {
		status = "warning"
	}

	evtMsg := fmt.Sprintf("Backup %s job=%s files=%d bytes=%d duration=%s",
		status, result.JobID, result.ChangedFiles, result.ChangedBytes,
		result.FinishedAt.Sub(result.StartedAt).Round(time.Second))
	evtType := "info"
	if status == "warning" {
		evtType = "warning"
	}
	notify.WriteEventLog(evtType, evtMsg)

	notifier.Notify(notify.BackupEvent{
		JobID:        result.JobID,
		NodeID:       resolveNodeID(),
		Status:       status,
		ChangedFiles: result.ChangedFiles,
		ChangedBytes: result.ChangedBytes,
		TotalFiles:   result.TotalFiles,
		Duration:     result.FinishedAt.Sub(result.StartedAt),
		Errors:       result.Errors,
		StartedAt:    result.StartedAt,
	})

	log.Info("backup finished",
		zap.String("job_id", result.JobID),
		zap.Int64("changed_files", result.ChangedFiles),
		zap.Int64("changed_bytes", result.ChangedBytes),
		zap.Int("errors", len(result.Errors)),
		zap.String("manifest", result.ManifestPath),
		zap.Duration("duration", result.FinishedAt.Sub(result.StartedAt)),
	)
	for _, e := range result.Errors {
		log.Warn("  " + e)
	}
	return nil
}

func runRetention(cfg *config.Config, log *zap.Logger) {
	if !cfg.Retention.GFS.Enabled && cfg.Retention.Days <= 0 {
		return
	}
	dest, err := factory.New(cfg)
	if err != nil {
		log.Warn("retention: open destination", zap.Error(err))
		return
	}
	defer dest.Close()
	if err := retention.Apply(dest, cfg.Retention, log); err != nil {
		log.Warn("retention: apply policy", zap.Error(err))
	}
}

func setup() (*config.Config, *zap.Logger, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, nil, fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("config validate: %w", err)
	}
	log, err := buildLogger(cfg.Log)
	if err != nil {
		return nil, nil, fmt.Errorf("logger: %w", err)
	}
	return cfg, log, nil
}

func buildLogger(lc config.LogConfig) (*zap.Logger, error) {
	level := zap.InfoLevel
	_ = level.UnmarshalText([]byte(lc.Level))

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var enc zapcore.Encoder
	if lc.Format == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enc = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		enc = zapcore.NewJSONEncoder(encoderCfg)
	}

	var sink zapcore.WriteSyncer
	if lc.Path != "" && lc.Path != "stdout" {
		f, err := os.OpenFile(lc.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
		if err != nil {
			return nil, err
		}
		sink = zapcore.AddSync(f)
	} else {
		sink = zapcore.AddSync(os.Stdout)
	}

	return zap.New(zapcore.NewCore(enc, sink, level), zap.AddCaller()), nil
}

// resolvedConfigPath returns an absolute path to the agent.yaml in use, or ""
// if running purely from defaults / env (no file to write back to).
func resolvedConfigPath() string {
	if cfgFile != "" {
		if abs, err := filepath.Abs(cfgFile); err == nil {
			return abs
		}
		return cfgFile
	}
	// Match the search order in config.Load()
	candidates := []string{
		"agent.yaml",
		filepath.Join(os.Getenv("HOME"), ".backupsmc", "agent.yaml"),
		"/etc/backupsmc/agent.yaml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	return ""
}

func resolveNodeID() string {
	if nodeID != "" {
		return nodeID
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return uuid.New().String()
}
