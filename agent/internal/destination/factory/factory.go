// Package factory creates the right destination.Writer based on config.
package factory

import (
	"fmt"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/destination"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/local"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/s3"
	sftpdest "github.com/smcsoluciones/backup-system/agent/internal/destination/sftp"
)

// New returns a destination.Writer for the configured destination type.
func New(cfg *config.Config) (destination.Writer, error) {
	switch cfg.Destination.Type {
	case "s3":
		if cfg.Destination.S3Bucket == "" {
			return nil, fmt.Errorf("destination: s3_bucket is required for S3 destination")
		}
		region := cfg.Destination.S3Region
		if region == "" {
			region = "us-east-1"
		}
		return s3.New(cfg.Destination.S3Bucket, region, cfg.Destination.S3Prefix)

	case "sftp":
		d := cfg.Destination
		port := d.SFTPPort
		if port == 0 {
			port = 22
		}
		return sftpdest.New(d.SFTPHost, port, d.SFTPUser, d.SFTPPassword, d.SFTPKeyFile, d.SFTPPath)

	case "local", "":
		if cfg.Destination.LocalPath == "" {
			return nil, fmt.Errorf("destination: local_path is required for local destination")
		}
		return local.New(cfg.Destination.LocalPath)

	default:
		return nil, fmt.Errorf("destination: unknown type %q (supported: local, s3, sftp)", cfg.Destination.Type)
	}
}
