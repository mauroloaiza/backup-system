package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	backup "github.com/smcsoluciones/backup-system/agent/internal/backup"
	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/local"
	"github.com/smcsoluciones/backup-system/agent/internal/reporter"
)

var (
	cfgFile string
	nodeID  string
)

func main() {
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
		restoreCmd(),
		validateCmd(),
		versionCmd(),
	)

	return root
}

// ── run ────────────────────────────────────────────────────────────────────────

func runCmd() *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a backup job",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := setup()
			if err != nil {
				return err
			}
			defer log.Sync() //nolint:errcheck

			if jobID == "" {
				jobID = uuid.New().String()
			}

			// Destination
			dest, err := local.New(cfg.Destination.LocalPath)
			if err != nil {
				return fmt.Errorf("destination: %w", err)
			}
			defer dest.Close()

			// Reporter (rate-limit: 5s between progress POSTs)
			rep := reporter.New(
				cfg.Server.URL,
				cfg.Server.APIToken,
				5*time.Second,
				log,
			)
			defer rep.Stop()

			engine := backup.New(cfg, dest, rep, log, resolveNodeID())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Graceful shutdown on SIGINT/SIGTERM
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs
				log.Info("received shutdown signal, finishing current file...")
				cancel()
			}()

			result, err := engine.Run(ctx, jobID)
			if err != nil {
				log.Error("backup failed", zap.Error(err))
				return err
			}

			log.Info("backup finished",
				zap.String("job_id", result.JobID),
				zap.Int64("changed_files", result.ChangedFiles),
				zap.Int64("changed_bytes", result.ChangedBytes),
				zap.Int("errors", len(result.Errors)),
				zap.String("manifest", result.ManifestPath),
				zap.Duration("duration", result.FinishedAt.Sub(result.StartedAt)),
			)

			if len(result.Errors) > 0 {
				log.Warn("backup completed with errors")
				for _, e := range result.Errors {
					log.Warn("  " + e)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "job ID (generated if empty)")
	return cmd
}

// ── restore ───────────────────────────────────────────────────────────────────

func restoreCmd() *cobra.Command {
	var (
		jobID      string
		targetPath string
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore files from a backup job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobID == "" {
				return fmt.Errorf("--job-id is required")
			}
			if targetPath == "" {
				return fmt.Errorf("--target is required")
			}

			_, log, err := setup()
			if err != nil {
				return err
			}
			defer log.Sync() //nolint:errcheck

			log.Info("restore not yet implemented",
				zap.String("job_id", jobID),
				zap.String("target", targetPath),
			)
			fmt.Println("Restore command — coming in v0.2.0")
			return nil
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "job ID to restore")
	cmd.Flags().StringVar(&targetPath, "target", "", "target directory for restored files")
	_ = cmd.MarkFlagRequired("job-id")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

// ── validate ──────────────────────────────────────────────────────────────────

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
				fmt.Fprintf(os.Stderr, "Configuration invalid:\n  %v\n", err)
				return err
			}
			fmt.Println("Configuration is valid.")
			return nil
		},
	}
}

// ── version ───────────────────────────────────────────────────────────────────

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print agent version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("BackupSMC Agent v0.1.0")
		},
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
	if err := level.UnmarshalText([]byte(lc.Level)); err != nil {
		level = zap.InfoLevel
	}

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

	core := zapcore.NewCore(enc, sink, level)
	return zap.New(core, zap.AddCaller()), nil
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
