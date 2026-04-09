// Package config manages the backup agent configuration.
// Uses Viper to load from YAML/TOML/ENV with sensible defaults.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ExtensionesNoComprimibles lists extensions that are already internally compressed.
var ExtensionesNoComprimibles = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".mp4":  true,
	".zip":  true,
	".gz":   true,
	".rar":  true,
	".7z":   true,
	".png":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".webm": true,
	".bz2":  true,
	".xz":   true,
	".zst":  true,
}

// Config is the root agent configuration.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Backup      BackupConfig      `mapstructure:"backup"`
	Destination DestinationConfig `mapstructure:"destination"`
	Retry       RetryConfig       `mapstructure:"retry"`
	Log         LogConfig         `mapstructure:"log"`
	Notify      NotifyConfig      `mapstructure:"notify"`
	Retention   RetentionConfig   `mapstructure:"retention"`
}

// ServerConfig holds connection parameters for the central BackupSMC server.
type ServerConfig struct {
	URL      string        `mapstructure:"url"`
	APIToken string        `mapstructure:"api_token"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// BackupConfig defines what gets backed up and how.
type BackupConfig struct {
	SourcePath           string          `mapstructure:"source_path"`
	SourcePaths          []string        `mapstructure:"source_paths"`
	ExcludePatterns      []string        `mapstructure:"exclude_patterns"`
	EncryptionPassphrase string          `mapstructure:"encryption_passphrase"`
	UseVSS               bool            `mapstructure:"use_vss"`
	Incremental          bool            `mapstructure:"incremental"`
	SkipCompressExts     map[string]bool `mapstructure:"skip_compress_exts"`
	ScheduleInterval     time.Duration   `mapstructure:"schedule_interval"`
	// VerifyAfterBackup re-reads and decrypts each file after writing to
	// confirm integrity. Slower but catches silent data corruption.
	VerifyAfterBackup bool `mapstructure:"verify_after_backup"`
	// ThrottleMbps limits upload speed (0 = unlimited).
	ThrottleMbps float64 `mapstructure:"throttle_mbps"`
	// PreScript is a command to run before the backup starts.
	// Non-zero exit aborts the backup.
	PreScript string `mapstructure:"pre_script"`
	// PostScript is a command to run after the backup completes (always, even on failure).
	PostScript string `mapstructure:"post_script"`
}

// RetentionConfig controls how long backups are kept.
type RetentionConfig struct {
	// Days is the simple max-age policy (0 = disabled, use GFS instead).
	Days int `mapstructure:"days"`
	// GFS enables Grandfather-Father-Son rotation.
	GFS GFSConfig `mapstructure:"gfs"`
}

// GFSConfig defines the Grandfather-Father-Son retention schedule.
type GFSConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	KeepDaily      int  `mapstructure:"keep_daily"`   // e.g. 7
	KeepWeekly     int  `mapstructure:"keep_weekly"`  // e.g. 4
	KeepMonthly    int  `mapstructure:"keep_monthly"` // e.g. 12
}

// DestinationConfig defines where backups are stored.
type DestinationConfig struct {
	Type      string `mapstructure:"type"`
	LocalPath string `mapstructure:"local_path"`
	S3Bucket  string `mapstructure:"s3_bucket"`
	S3Region  string `mapstructure:"s3_region"`
	S3Prefix  string `mapstructure:"s3_prefix"`
	SFTPHost  string `mapstructure:"sftp_host"`
	SFTPPort  int    `mapstructure:"sftp_port"`
	SFTPUser  string `mapstructure:"sftp_user"`
	SFTPPath  string `mapstructure:"sftp_path"`
	// SFTPPassword or SFTPKeyFile for authentication.
	SFTPPassword string `mapstructure:"sftp_password"`
	SFTPKeyFile  string `mapstructure:"sftp_key_file"`
}

// RetryConfig controls retries on transient failures.
type RetryConfig struct {
	MaxAttempts  int           `mapstructure:"max_attempts"`
	InitialDelay time.Duration `mapstructure:"initial_delay"`
}

// LogConfig controls the logger.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Path   string `mapstructure:"path"`
}

// NotifyConfig controls alerting on backup events.
type NotifyConfig struct {
	Email EmailConfig `mapstructure:"email"`
}

// EmailConfig configures SMTP notifications.
type EmailConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	SMTPHost  string   `mapstructure:"smtp_host"`
	SMTPPort  int      `mapstructure:"smtp_port"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
	From      string   `mapstructure:"from"`
	To        []string `mapstructure:"to"`
	// OnFailure sends an alert when a backup fails.
	OnFailure bool `mapstructure:"on_failure"`
	// OnSuccess sends a summary when a backup completes.
	OnSuccess bool `mapstructure:"on_success"`
}

// Load reads config from the given YAML file (or standard locations if empty).
func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.timeout", 30*time.Second)
	v.SetDefault("backup.use_vss", true)
	v.SetDefault("backup.incremental", true)
	v.SetDefault("backup.schedule_interval", 24*time.Hour)
	v.SetDefault("backup.verify_after_backup", false)
	v.SetDefault("backup.throttle_mbps", 0)
	v.SetDefault("retry.max_attempts", 3)
	v.SetDefault("retry.initial_delay", time.Second)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.path", "stdout")
	v.SetDefault("destination.type", "local")
	v.SetDefault("destination.sftp_port", 22)
	v.SetDefault("retention.days", 30)
	v.SetDefault("retention.gfs.enabled", false)
	v.SetDefault("retention.gfs.keep_daily", 7)
	v.SetDefault("retention.gfs.keep_weekly", 4)
	v.SetDefault("retention.gfs.keep_monthly", 12)
	v.SetDefault("notify.email.smtp_port", 587)
	v.SetDefault("notify.email.on_failure", true)
	v.SetDefault("notify.email.on_success", false)

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("agent")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.backupsmc")
		v.AddConfigPath("/etc/backupsmc")
	}

	v.SetEnvPrefix("BACKUPSMC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("deserializing config: %w", err)
	}

	if cfg.Backup.SkipCompressExts == nil {
		cfg.Backup.SkipCompressExts = make(map[string]bool)
	}
	for ext, val := range ExtensionesNoComprimibles {
		cfg.Backup.SkipCompressExts[ext] = val
	}

	return cfg, nil
}

// EffectiveSourcePaths returns the resolved list of source paths to back up.
func (c *Config) EffectiveSourcePaths() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range c.Backup.SourcePaths {
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	if c.Backup.SourcePath != "" && !seen[c.Backup.SourcePath] {
		out = append(out, c.Backup.SourcePath)
	}
	return out
}

// Validate checks that the minimum required configuration is present.
func (c *Config) Validate() error {
	if c.Backup.EncryptionPassphrase == "" {
		return fmt.Errorf("backup.encryption_passphrase is required")
	}
	if len(c.Backup.EncryptionPassphrase) < 16 {
		return fmt.Errorf("backup.encryption_passphrase must be at least 16 characters")
	}
	if len(c.EffectiveSourcePaths()) == 0 {
		return fmt.Errorf("backup.source_path (or source_paths) is required")
	}
	if (c.Destination.Type == "local" || c.Destination.Type == "") && c.Destination.LocalPath == "" {
		return fmt.Errorf("destination.local_path is required for local destination")
	}
	if c.Destination.Type == "s3" && c.Destination.S3Bucket == "" {
		return fmt.Errorf("destination.s3_bucket is required for S3 destination")
	}
	if c.Destination.Type == "sftp" {
		if c.Destination.SFTPHost == "" {
			return fmt.Errorf("destination.sftp_host is required for SFTP destination")
		}
		if c.Destination.SFTPUser == "" {
			return fmt.Errorf("destination.sftp_user is required for SFTP destination")
		}
	}
	if c.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be at least 1")
	}
	return nil
}
