package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/backup/databases"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/backup/files"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/config"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/destination"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/reporter"
)

var (
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle = lipgloss.NewStyle().Bold(true)
)

// SourceResult holds the result of backing up one source.
type SourceResult struct {
	Source   string
	Files    int
	Bytes    int64
	Duration time.Duration
	Err      error
}

// Run executes the full backup pipeline.
func Run(cfg *config.Config) error {
	start := time.Now()

	// reporter (errors are non-fatal — offline mode ok)
	rep := reporter.New(reporter.Config{
		URL:   cfg.Server.URL,
		Token: cfg.Server.Token,
	})
	_ = rep.Heartbeat(cfg.Sources.Files.Paths)
	jobID, _ := rep.StartJob("backup")

	// staging dir
	stagingDir, err := os.MkdirTemp("", "backupsmc_*")
	if err != nil {
		return fmt.Errorf("staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	var results []SourceResult

	// ── files ─────────────────────────────────────────────────────────────────
	if cfg.Sources.Files.Enabled && len(cfg.Sources.Files.Paths) > 0 {
		fmt.Printf("  %s Archivos...\n", dimStyle.Render("→"))
		r, err := files.Backup(cfg.Sources.Files.Paths, cfg.Sources.Files.Exclude, stagingDir)
		results = append(results, SourceResult{
			Source: "files", Files: r.FilesTotal, Bytes: r.BytesTotal, Duration: r.Duration, Err: err,
		})
		if err != nil {
			fmt.Printf("  %s archivos: %v\n", errStyle.Render("✗"), err)
		} else {
			fmt.Printf("  %s archivos: %s en %s\n",
				okStyle.Render("✓"), fmtBytes(r.BytesTotal), r.Duration.Round(time.Millisecond))
		}
	}

	// ── databases ─────────────────────────────────────────────────────────────
	if db := cfg.Sources.Databases; db != (config.DatabasesConfig{}) {
		if db.PostgreSQL != nil && db.PostgreSQL.Enabled {
			fmt.Printf("  %s PostgreSQL...\n", dimStyle.Render("→"))
			rs, err := databases.BackupPostgreSQL(databases.PostgreSQLConfig{
				Host:      db.PostgreSQL.Host,
				Port:      strconv.Itoa(db.PostgreSQL.Port),
				User:      db.PostgreSQL.User,
				Password:  db.PostgreSQL.Password,
				Databases: db.PostgreSQL.Databases,
			}, stagingDir)
			for _, r := range rs {
				results = append(results, SourceResult{
					Source: "pg:" + r.Name, Bytes: r.Bytes, Duration: r.Duration,
				})
			}
			if err != nil {
				fmt.Printf("  %s postgresql: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s postgresql: %d bases\n", okStyle.Render("✓"), len(rs))
			}
		}

		if db.MySQL != nil && db.MySQL.Enabled {
			fmt.Printf("  %s MySQL...\n", dimStyle.Render("→"))
			rs, err := databases.BackupMySQL(databases.MySQLConfig{
				Host:      db.MySQL.Host,
				Port:      strconv.Itoa(db.MySQL.Port),
				User:      db.MySQL.User,
				Password:  db.MySQL.Password,
				Databases: db.MySQL.Databases,
			}, stagingDir)
			for _, r := range rs {
				results = append(results, SourceResult{
					Source: "mysql:" + r.Name, Bytes: r.Bytes, Duration: r.Duration,
				})
			}
			if err != nil {
				fmt.Printf("  %s mysql: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s mysql: %d bases\n", okStyle.Render("✓"), len(rs))
			}
		}

		if db.MongoDB != nil && db.MongoDB.Enabled {
			fmt.Printf("  %s MongoDB...\n", dimStyle.Render("→"))
			r, err := databases.BackupMongoDB(databases.MongoDBConfig{
				URI:       db.MongoDB.URI,
				Databases: db.MongoDB.Databases,
			}, stagingDir)
			results = append(results, SourceResult{
				Source: "mongodb", Bytes: r.Bytes, Duration: r.Duration, Err: err,
			})
			if err != nil {
				fmt.Printf("  %s mongodb: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s mongodb: %s\n", okStyle.Render("✓"), r.Archive)
			}
		}

		if db.Redis != nil && db.Redis.Enabled {
			fmt.Printf("  %s Redis...\n", dimStyle.Render("→"))
			r, err := databases.BackupRedis(databases.RedisConfig{
				Host:     db.Redis.Host,
				Port:     strconv.Itoa(db.Redis.Port),
				Password: db.Redis.Password,
				DataDir:  db.Redis.DataDir,
			}, stagingDir)
			results = append(results, SourceResult{
				Source: "redis", Bytes: r.Bytes, Duration: r.Duration, Err: err,
			})
			if err != nil {
				fmt.Printf("  %s redis: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s redis: %s\n", okStyle.Render("✓"), fmtBytes(r.Bytes))
			}
		}

		if db.SQLite != nil && db.SQLite.Enabled {
			fmt.Printf("  %s SQLite...\n", dimStyle.Render("→"))
			rs, err := databases.BackupSQLite(databases.SQLiteConfig{
				Files: db.SQLite.Files,
			}, stagingDir)
			for _, r := range rs {
				results = append(results, SourceResult{
					Source: "sqlite:" + r.Name, Bytes: r.Bytes, Duration: r.Duration,
				})
			}
			if err != nil {
				fmt.Printf("  %s sqlite: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s sqlite: %d archivos\n", okStyle.Render("✓"), len(rs))
			}
		}

		if db.Elasticsearch != nil && db.Elasticsearch.Enabled {
			fmt.Printf("  %s Elasticsearch...\n", dimStyle.Render("→"))
			r, err := databases.BackupElasticsearch(databases.ElasticsearchConfig{
				URL:      db.Elasticsearch.URL,
				RepoName: "backupsmc",
			}, stagingDir)
			results = append(results, SourceResult{
				Source: "elasticsearch", Duration: r.Duration, Err: err,
			})
			if err != nil {
				fmt.Printf("  %s elasticsearch: %v\n", errStyle.Render("✗"), err)
			} else {
				fmt.Printf("  %s elasticsearch: snapshot creado\n", okStyle.Render("✓"))
			}
		}
	}

	// ── upload to destination ─────────────────────────────────────────────────
	destCfg := buildDestConfig(cfg)
	writer, err := destination.New(destCfg)
	if err != nil {
		_ = rep.Fail(jobID, err.Error())
		return fmt.Errorf("destino: %w", err)
	}

	fmt.Printf("  %s Subiendo a %s...\n", dimStyle.Render("→"), writer.Name())

	entries, _ := os.ReadDir(stagingDir)
	var uploadErrors int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := writer.Write(filepath.Join(stagingDir, e.Name())); err != nil {
			fmt.Printf("  %s upload %s: %v\n", errStyle.Render("✗"), e.Name(), err)
			uploadErrors++
		}
	}

	// ── report ────────────────────────────────────────────────────────────────
	var totalFiles int
	var totalBytes int64
	var anyErr bool
	for _, r := range results {
		totalFiles += r.Files
		totalBytes += r.Bytes
		if r.Err != nil {
			anyErr = true
		}
	}

	if anyErr || uploadErrors > 0 {
		_ = rep.Fail(jobID, "backup completado con errores")
	} else {
		_ = rep.Complete(jobID, totalFiles, totalBytes)
	}

	PrintFinalReport(results, writer.Name(), time.Since(start))
	return nil
}

// PrintFinalReport prints the Option-B final screen.
func PrintFinalReport(results []SourceResult, dest string, total time.Duration) {
	fmt.Println()
	fmt.Println(boldStyle.Render("  Resumen de backup"))
	fmt.Println()

	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  %s %-25s %s\n",
				errStyle.Render("✗"), r.Source, errStyle.Render(r.Err.Error()))
		} else {
			fmt.Printf("  %s %-25s %s   %s\n",
				okStyle.Render("✓"), r.Source,
				fmtBytes(r.Bytes),
				dimStyle.Render(r.Duration.Round(time.Millisecond).String()))
		}
	}

	fmt.Println()
	fmt.Printf("  %-18s %s\n", "Destino:", dest)
	fmt.Printf("  %-18s %s\n", "Duracion total:", total.Round(time.Second))
	fmt.Println()
	fmt.Println(dimStyle.Render("  ─────────────────────────────────────────────"))
	fmt.Println(dimStyle.Render("  Comandos utiles:"))
	fmt.Println()
	fmt.Println("    backupsmc-agent run          ejecutar backup ahora")
	fmt.Println("    backupsmc-agent status        ver estado")
	fmt.Println("    backupsmc-agent logs          ver logs")
	fmt.Println("    backupsmc-agent service stop  pausar servicio")
	fmt.Println()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fmtBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func buildDestConfig(cfg *config.Config) destination.Config {
	d := destination.Config{Type: cfg.Destination.Type}
	if cfg.Destination.Local != nil {
		d.Local = destination.LocalConfig{Path: cfg.Destination.Local.Path}
	}
	if cfg.Destination.SFTP != nil {
		s := cfg.Destination.SFTP
		d.SFTP = destination.SFTPConfig{
			Host:     s.Host,
			Port:     strconv.Itoa(s.Port),
			User:     s.User,
			Password: s.Password,
			KeyFile:  s.KeyFile,
			Path:     s.Path,
		}
	}
	if cfg.Destination.S3 != nil {
		s := cfg.Destination.S3
		d.S3 = destination.S3Config{
			Bucket:    s.Bucket,
			Region:    s.Region,
			Endpoint:  s.Endpoint,
			AccessKey: s.AccessKey,
			SecretKey: s.SecretKey,
			Prefix:    s.Prefix,
		}
	}
	if cfg.Destination.NFS != nil {
		n := cfg.Destination.NFS
		d.NFS = destination.NFSConfig{
			Server:     n.Server,
			Share:      n.Export,
			MountPoint: n.MountPoint,
		}
	}
	if cfg.Destination.SMB != nil {
		s := cfg.Destination.SMB
		d.SMB = destination.SMBConfig{
			Share:      s.Share,
			MountPoint: s.MountPoint,
			User:       s.User,
			Password:   s.Password,
			Domain:     s.Domain,
		}
	}
	return d
}
