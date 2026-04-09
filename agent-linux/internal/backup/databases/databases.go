package databases

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Result holds the outcome of a single database backup.
type Result struct {
	Name     string
	Archive  string
	Bytes    int64
	Duration time.Duration
}

// ── PostgreSQL ────────────────────────────────────────────────────────────────

type PostgreSQLConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Databases []string
}

func BackupPostgreSQL(cfg PostgreSQLConfig, destDir string) ([]Result, error) {
	port := cfg.Port
	if port == "" {
		port = "5432"
	}
	var results []Result
	for _, db := range cfg.Databases {
		r, err := backupPgDB(cfg.Host, port, cfg.User, cfg.Password, db, destDir)
		if err != nil {
			return results, fmt.Errorf("postgresql %s: %w", db, err)
		}
		results = append(results, r)
	}
	return results, nil
}

func backupPgDB(host, port, user, password, dbName, destDir string) (Result, error) {
	start := time.Now()
	out := filepath.Join(destDir, fmt.Sprintf("pg_%s_%s.dump", dbName, timestamp()))
	cmd := exec.Command("pg_dump",
		"-h", host,
		"-p", port,
		"-U", user,
		"-Fc",
		"-f", out,
		dbName,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	if b, err := cmd.CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("%w\n%s", err, b)
	}
	info, _ := os.Stat(out)
	return Result{Name: dbName, Archive: out, Bytes: info.Size(), Duration: time.Since(start)}, nil
}

// ── MySQL ─────────────────────────────────────────────────────────────────────

type MySQLConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Databases []string
}

func BackupMySQL(cfg MySQLConfig, destDir string) ([]Result, error) {
	port := cfg.Port
	if port == "" {
		port = "3306"
	}
	var results []Result
	for _, db := range cfg.Databases {
		r, err := backupMySQLDB(cfg.Host, port, cfg.User, cfg.Password, db, destDir)
		if err != nil {
			return results, fmt.Errorf("mysql %s: %w", db, err)
		}
		results = append(results, r)
	}
	return results, nil
}

func backupMySQLDB(host, port, user, password, dbName, destDir string) (Result, error) {
	start := time.Now()
	out := filepath.Join(destDir, fmt.Sprintf("mysql_%s_%s.sql.gz", dbName, timestamp()))

	f, err := os.Create(out)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	dump := exec.Command("mysqldump",
		"-h", host,
		"-P", port,
		"-u", user,
		"-p"+password,
		"--single-transaction",
		"--routines",
		"--triggers",
		dbName,
	)
	dump.Stdout = gw
	if b, err := dump.CombinedOutput(); err != nil {
		os.Remove(out)
		return Result{}, fmt.Errorf("%w\n%s", err, b)
	}

	info, _ := os.Stat(out)
	return Result{Name: dbName, Archive: out, Bytes: info.Size(), Duration: time.Since(start)}, nil
}

// ── MongoDB ───────────────────────────────────────────────────────────────────

type MongoDBConfig struct {
	URI     string
	Databases []string
}

func BackupMongoDB(cfg MongoDBConfig, destDir string) (Result, error) {
	start := time.Now()
	outDir := filepath.Join(destDir, "mongo_"+timestamp())
	args := []string{"--gzip", "--out", outDir}
	if cfg.URI != "" {
		args = append(args, "--uri", cfg.URI)
	}
	for _, db := range cfg.Databases {
		args = append(args, "--db", db)
	}
	cmd := exec.Command("mongodump", args...)
	if b, err := cmd.CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("mongodump: %w\n%s", err, b)
	}
	return Result{Name: "mongodb", Archive: outDir, Duration: time.Since(start)}, nil
}

// ── Redis ─────────────────────────────────────────────────────────────────────

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DataDir  string // usually /var/lib/redis
}

func BackupRedis(cfg RedisConfig, destDir string) (Result, error) {
	start := time.Now()
	port := cfg.Port
	if port == "" {
		port = "6379"
	}
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "/var/lib/redis"
	}

	// trigger BGSAVE
	args := []string{"-h", cfg.Host, "-p", port}
	if cfg.Password != "" {
		args = append(args, "-a", cfg.Password)
	}
	args = append(args, "BGSAVE")
	if b, err := exec.Command("redis-cli", args...).CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("BGSAVE: %w\n%s", err, b)
	}

	// copy dump.rdb
	src := filepath.Join(dataDir, "dump.rdb")
	dst := filepath.Join(destDir, fmt.Sprintf("redis_dump_%s.rdb", timestamp()))
	if err := copyFile(src, dst); err != nil {
		return Result{}, fmt.Errorf("copiar dump.rdb: %w", err)
	}
	info, _ := os.Stat(dst)
	return Result{Name: "redis", Archive: dst, Bytes: info.Size(), Duration: time.Since(start)}, nil
}

// ── SQLite ────────────────────────────────────────────────────────────────────

type SQLiteConfig struct {
	Files []string
}

func BackupSQLite(cfg SQLiteConfig, destDir string) ([]Result, error) {
	var results []Result
	for _, dbFile := range cfg.Files {
		start := time.Now()
		base := strings.TrimSuffix(filepath.Base(dbFile), filepath.Ext(dbFile))
		out := filepath.Join(destDir, fmt.Sprintf("sqlite_%s_%s.db", base, timestamp()))

		// try VACUUM INTO first
		err := sqliteVacuumInto(dbFile, out)
		if err != nil {
			// fallback: plain file copy
			err = copyFile(dbFile, out)
		}
		if err != nil {
			return results, fmt.Errorf("sqlite %s: %w", dbFile, err)
		}
		info, _ := os.Stat(out)
		results = append(results, Result{
			Name: base, Archive: out, Bytes: info.Size(), Duration: time.Since(start),
		})
	}
	return results, nil
}

func sqliteVacuumInto(src, dst string) error {
	cmd := exec.Command("sqlite3", src, fmt.Sprintf("VACUUM INTO '%s'", dst))
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, b)
	}
	return nil
}

// ── Elasticsearch ─────────────────────────────────────────────────────────────

type ElasticsearchConfig struct {
	URL          string
	RepoName     string
	SnapshotName string
}

func BackupElasticsearch(cfg ElasticsearchConfig, destDir string) (Result, error) {
	start := time.Now()
	snapName := cfg.SnapshotName
	if snapName == "" {
		snapName = "backup_" + timestamp()
	}
	url := fmt.Sprintf("%s/_snapshot/%s/%s", cfg.URL, cfg.RepoName, snapName)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("elasticsearch snapshot: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("elasticsearch %d: %s", resp.StatusCode, body)
	}
	return Result{
		Name:     "elasticsearch",
		Archive:  fmt.Sprintf("%s/_snapshot/%s/%s", cfg.URL, cfg.RepoName, snapName),
		Duration: time.Since(start),
	}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func timestamp() string {
	return time.Now().Format("20060102_150405")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
