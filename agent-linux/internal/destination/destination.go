package destination

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Writer copies a backup file to its final destination.
type Writer interface {
	Write(localPath string) error
	Name() string
}

// Config mirrors config.DestinationConfig.
type Config struct {
	Type  string // local | sftp | s3 | nfs | smb
	Local LocalConfig
	SFTP  SFTPConfig
	S3    S3Config
	NFS   NFSConfig
	SMB   SMBConfig
}

type LocalConfig struct {
	Path string
}

type SFTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	KeyFile  string
	Path     string
}

type S3Config struct {
	Bucket   string
	Region   string
	Endpoint string
	AccessKey string
	SecretKey string
	Prefix   string
}

type NFSConfig struct {
	Server string
	Share  string
	MountPoint string
}

type SMBConfig struct {
	Server     string
	Share      string
	User       string
	Password   string
	Domain     string
	MountPoint string
}

// New returns the appropriate Writer for the given config.
func New(cfg Config) (Writer, error) {
	switch cfg.Type {
	case "local":
		return &localWriter{path: cfg.Local.Path}, nil
	case "sftp":
		return &sftpWriter{cfg: cfg.SFTP}, nil
	case "s3":
		return &s3Writer{cfg: cfg.S3}, nil
	case "nfs":
		return &nfsWriter{cfg: cfg.NFS}, nil
	case "smb":
		return &smbWriter{cfg: cfg.SMB}, nil
	default:
		return nil, fmt.Errorf("tipo de destino desconocido: %s", cfg.Type)
	}
}

// ── local ─────────────────────────────────────────────────────────────────────

type localWriter struct{ path string }

func (w *localWriter) Name() string { return "local:" + w.path }

func (w *localWriter) Write(localPath string) error {
	if err := os.MkdirAll(w.path, 0o750); err != nil {
		return err
	}
	dst := filepath.Join(w.path, filepath.Base(localPath))
	return copyFile(localPath, dst)
}

// ── sftp ──────────────────────────────────────────────────────────────────────

type sftpWriter struct{ cfg SFTPConfig }

func (w *sftpWriter) Name() string { return "sftp://" + w.cfg.Host + w.cfg.Path }

func (w *sftpWriter) Write(localPath string) error {
	port := w.cfg.Port
	if port == "" {
		port = "22"
	}
	remote := w.cfg.User + "@" + w.cfg.Host + ":" + filepath.Join(w.cfg.Path, filepath.Base(localPath))
	args := []string{"-P", port, "-o", "StrictHostKeyChecking=no"}
	if w.cfg.KeyFile != "" {
		args = append(args, "-i", w.cfg.KeyFile)
	}
	args = append(args, localPath, remote)
	cmd := exec.Command("scp", args...)
	if w.cfg.Password != "" {
		cmd = exec.Command("sshpass", append([]string{"-p", w.cfg.Password, "scp"}, args...)...)
	}
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("scp: %w\n%s", err, b)
	}
	return nil
}

// ── s3 ────────────────────────────────────────────────────────────────────────

type s3Writer struct{ cfg S3Config }

func (w *s3Writer) Name() string { return "s3://" + w.cfg.Bucket }

func (w *s3Writer) Write(localPath string) error {
	key := filepath.Join(w.cfg.Prefix, filepath.Base(localPath))
	args := []string{"s3", "cp", localPath, fmt.Sprintf("s3://%s/%s", w.cfg.Bucket, key)}
	if w.cfg.Region != "" {
		args = append(args, "--region", w.cfg.Region)
	}
	if w.cfg.Endpoint != "" {
		args = append(args, "--endpoint-url", w.cfg.Endpoint)
	}
	cmd := exec.Command("aws", args...)
	cmd.Env = append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+w.cfg.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+w.cfg.SecretKey,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("aws s3 cp: %w\n%s", err, b)
	}
	return nil
}

// ── nfs ───────────────────────────────────────────────────────────────────────

type nfsWriter struct{ cfg NFSConfig }

func (w *nfsWriter) Name() string { return "nfs://" + w.cfg.Server + w.cfg.Share }

func (w *nfsWriter) Write(localPath string) error {
	mp := w.cfg.MountPoint
	if mp == "" {
		mp = "/mnt/backupsmc_nfs"
	}
	if err := os.MkdirAll(mp, 0o750); err != nil {
		return err
	}
	mount := exec.Command("mount", "-t", "nfs",
		w.cfg.Server+":"+w.cfg.Share, mp,
	)
	if b, err := mount.CombinedOutput(); err != nil {
		return fmt.Errorf("nfs mount: %w\n%s", err, b)
	}
	defer exec.Command("umount", mp).Run()

	return copyFile(localPath, filepath.Join(mp, filepath.Base(localPath)))
}

// ── smb ───────────────────────────────────────────────────────────────────────

type smbWriter struct{ cfg SMBConfig }

func (w *smbWriter) Name() string { return "smb://" + w.cfg.Server + "/" + w.cfg.Share }

func (w *smbWriter) Write(localPath string) error {
	mp := w.cfg.MountPoint
	if mp == "" {
		mp = "/mnt/backupsmc_smb"
	}
	if err := os.MkdirAll(mp, 0o750); err != nil {
		return err
	}
	opts := fmt.Sprintf("username=%s,password=%s", w.cfg.User, w.cfg.Password)
	if w.cfg.Domain != "" {
		opts += ",domain=" + w.cfg.Domain
	}
	mount := exec.Command("mount", "-t", "cifs",
		"//"+w.cfg.Server+"/"+w.cfg.Share, mp,
		"-o", opts,
	)
	if b, err := mount.CombinedOutput(); err != nil {
		return fmt.Errorf("smb mount: %w\n%s", err, b)
	}
	defer exec.Command("umount", mp).Run()

	return copyFile(localPath, filepath.Join(mp, filepath.Base(localPath)))
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
