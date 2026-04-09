// Package notify sends backup event notifications via email and Windows Event Log.
package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
)

// BackupEvent describes the outcome of a backup job for notification purposes.
type BackupEvent struct {
	JobID        string
	NodeID       string
	Status       string // "completed" | "failed" | "warning"
	ChangedFiles int64
	ChangedBytes int64
	TotalFiles   int64
	Duration     time.Duration
	Errors       []string
	StartedAt    time.Time
}

// Notifier sends notifications based on config.
type Notifier struct {
	cfg config.NotifyConfig
	log *zap.Logger
}

// New creates a Notifier.
func New(cfg config.NotifyConfig, log *zap.Logger) *Notifier {
	return &Notifier{cfg: cfg, log: log}
}

// Notify sends appropriate notifications for the given event.
func (n *Notifier) Notify(ev BackupEvent) {
	ec := n.cfg.Email
	if !ec.Enabled || len(ec.To) == 0 {
		return
	}

	switch ev.Status {
	case "failed":
		if ec.OnFailure {
			if err := n.sendEmail(ec, subjectFail(ev), bodyFail(ev)); err != nil {
				n.log.Warn("notify: email send failed", zap.Error(err))
			}
		}
	case "completed":
		if ec.OnSuccess {
			if err := n.sendEmail(ec, subjectOK(ev), bodyOK(ev)); err != nil {
				n.log.Warn("notify: email send success", zap.Error(err))
			}
		}
	case "warning":
		if ec.OnFailure { // warnings go through the failure channel
			if err := n.sendEmail(ec, subjectWarn(ev), bodyWarn(ev)); err != nil {
				n.log.Warn("notify: email send warning", zap.Error(err))
			}
		}
	}
}

func (n *Notifier) sendEmail(ec config.EmailConfig, subject, body string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		ec.From,
		strings.Join(ec.To, ", "),
		subject,
		body,
	)

	host := ec.SMTPHost
	port := ec.SMTPPort
	addr := fmt.Sprintf("%s:%d", host, port)

	var auth smtp.Auth
	if ec.Username != "" {
		auth = smtp.PlainAuth("", ec.Username, ec.Password, host)
	}

	// Try STARTTLS on port 587, plain TLS on port 465, plain on others
	if port == 465 {
		return sendTLS(addr, host, auth, ec.From, ec.To, []byte(msg))
	}
	return smtp.SendMail(addr, auth, ec.From, ec.To, []byte(msg))
}

func sendTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, r := range to {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("smtp rcpt %q: %w", r, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

// hostName returns the hostname part of an addr (ignores error).
func hostName(addr string) string {
	h, _, _ := net.SplitHostPort(addr)
	return h
}

// ── Message templates ─────────────────────────────────────────────────────────

func fmtBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func subjectOK(ev BackupEvent) string {
	return fmt.Sprintf("[BackupSMC] ✅ Backup completado — %s", ev.NodeID)
}
func subjectFail(ev BackupEvent) string {
	return fmt.Sprintf("[BackupSMC] ❌ Backup FALLIDO — %s", ev.NodeID)
}
func subjectWarn(ev BackupEvent) string {
	return fmt.Sprintf("[BackupSMC] ⚠️  Backup con advertencias — %s", ev.NodeID)
}

func bodyOK(ev BackupEvent) string {
	return fmt.Sprintf(`Resumen de backup

Nodo        : %s
Job ID      : %s
Estado      : Completado ✅
Inicio      : %s
Duración    : %s
Archivos    : %d cambiados / %d total
Datos       : %s

---
BackupSMC · SMC Soluciones
`,
		ev.NodeID, ev.JobID,
		ev.StartedAt.Format("02 Jan 2006 15:04:05"),
		ev.Duration.Round(time.Second),
		ev.ChangedFiles, ev.TotalFiles,
		fmtBytes(ev.ChangedBytes),
	)
}

func bodyFail(ev BackupEvent) string {
	errSummary := strings.Join(ev.Errors, "\n  - ")
	return fmt.Sprintf(`⚠️  El backup ha FALLADO

Nodo        : %s
Job ID      : %s
Estado      : Fallido ❌
Inicio      : %s
Duración    : %s

Errores:
  - %s

Revisa los logs del agente para más detalles.

---
BackupSMC · SMC Soluciones
`,
		ev.NodeID, ev.JobID,
		ev.StartedAt.Format("02 Jan 2006 15:04:05"),
		ev.Duration.Round(time.Second),
		errSummary,
	)
}

func bodyWarn(ev BackupEvent) string {
	errSummary := strings.Join(ev.Errors, "\n  - ")
	return fmt.Sprintf(`El backup completó con advertencias

Nodo        : %s
Job ID      : %s
Estado      : Advertencias ⚠️
Inicio      : %s
Duración    : %s
Archivos    : %d cambiados / %d total
Datos       : %s

Archivos con error:
  - %s

---
BackupSMC · SMC Soluciones
`,
		ev.NodeID, ev.JobID,
		ev.StartedAt.Format("02 Jan 2006 15:04:05"),
		ev.Duration.Round(time.Second),
		ev.ChangedFiles, ev.TotalFiles,
		fmtBytes(ev.ChangedBytes),
		errSummary,
	)
}

// keep unused import happy
var _ = hostName
