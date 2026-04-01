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
// Compressing these formats would waste CPU without reducing size.
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
}

// ServerConfig holds connection parameters for the central BackupSMC server.
type ServerConfig struct {
	URL      string        `mapstructure:"url"`       // e.g. https://backup.smcsoluciones.com
	APIToken string        `mapstructure:"api_token"` // Bearer token for agent authentication
	Timeout  time.Duration `mapstructure:"timeout"`
}

// BackupConfig defines what gets backed up and how it is processed.
type BackupConfig struct {
	SourcePath           string          `mapstructure:"source_path"`            // root directory to back up
	ExcludePatterns      []string        `mapstructure:"exclude_patterns"`       // glob patterns to skip
	EncryptionPassphrase string          `mapstructure:"encryption_passphrase"`  // AES-256-GCM passphrase
	UseVSS               bool            `mapstructure:"use_vss"`                // enable VSS snapshots (Windows)
	Incremental          bool            `mapstructure:"incremental"`            // incremental vs full backup
	SkipCompressExts     map[string]bool `mapstructure:"skip_compress_exts"`     // extensions to skip compression
}

// DestinationConfig defines where backups are stored.
type DestinationConfig struct {
	Type      string `mapstructure:"type"`       // "local" | "s3" | "sftp"
	LocalPath string `mapstructure:"local_path"` // used when type = "local"
	S3Bucket  string `mapstructure:"s3_bucket"`
	S3Region  string `mapstructure:"s3_region"`
	S3Prefix  string `mapstructure:"s3_prefix"`
	SFTPHost  string `mapstructure:"sftp_host"`
	SFTPPath  string `mapstructure:"sftp_path"`
	SFTPUser  string `mapstructure:"sftp_user"`
}

// RetryConfig controls retries on transient failures.
type RetryConfig struct {
	MaxAttempts  int           `mapstructure:"max_attempts"`
	InitialDelay time.Duration `mapstructure:"initial_delay"`
}

// LogConfig controls the logger behaviour.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug | info | warn | error
	Format string `mapstructure:"format"` // json | console
	Path   string `mapstructure:"path"`   // file path or "stdout"
}

// Load reads config from the given YAML file (or standard locations if empty).
// Environment variables with prefix BACKUPSMC_ override file values.
func Load(path string) (*Config, error) {
	v := viper.New()

	// Sensible defaults
	v.SetDefault("server.timeout", 30*time.Second)
	v.SetDefault("backup.use_vss", true)
	v.SetDefault("backup.incremental", true)
	v.SetDefault("retry.max_attempts", 3)
	v.SetDefault("retry.initial_delay", time.Second)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.path", "stdout")
	v.SetDefault("destination.type", "local")

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("agent")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.backupsmc")
		v.AddConfigPath("/etc/backupsmc")
	}

	// BACKUPSMC_BACKUP_ENCRYPTION_PASSPHRASE → backup.encryption_passphrase
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

	// Merge static skip-compress map with any user-defined extensions
	if cfg.Backup.SkipCompressExts == nil {
		cfg.Backup.SkipCompressExts = make(map[string]bool)
	}
	for ext, v := range ExtensionesNoComprimibles {
		cfg.Backup.SkipCompressExts[ext] = v
	}

	return cfg, nil
}

// Validate checks that the minimum required configuration is present.
func (c *Config) Validate() error {
	if c.Backup.EncryptionPassphrase == "" {
		return fmt.Errorf("backup.encryption_passphrase is required")
	}
	if len(c.Backup.EncryptionPassphrase) < 16 {
		return fmt.Errorf("backup.encryption_passphrase must be at least 16 characters")
	}
	if c.Backup.SourcePath == "" {
		return fmt.Errorf("backup.source_path is required")
	}
	if c.Destination.Type == "local" && c.Destination.LocalPath == "" {
		return fmt.Errorf("destination.local_path is required for local destination")
	}
	if c.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be at least 1")
	}
	return nil
}
