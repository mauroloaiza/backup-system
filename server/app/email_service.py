"""
Outbound email via SMTP (stdlib) — mirrors SMC Desk's email.service.

Provider modes:
  - "smtp"       : use the host/port/secure fields directly
  - "office365"  : forced host=smtp.office365.com, port=587, STARTTLS
"""
from __future__ import annotations
import logging
import smtplib
import ssl
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.utils import formataddr
from typing import Optional

from sqlalchemy.orm import Session
from app.models import EmailSettings

log = logging.getLogger("backupsmc.email")

MASK = "••••••••"


def get_config(db: Session) -> Optional[EmailSettings]:
    return db.query(EmailSettings).order_by(EmailSettings.id.asc()).first()


def is_configured(cfg: Optional[EmailSettings]) -> bool:
    if not cfg:
        return False
    # Office365 only needs user + pass + from_email (host is implicit)
    if cfg.outbound_provider == "office365":
        return bool(cfg.smtp_user and cfg.smtp_pass and cfg.from_email)
    return bool(cfg.smtp_host and cfg.smtp_user and cfg.smtp_pass and cfg.from_email)


def send_mail(
    db: Session,
    to: str,
    subject: str,
    html: str,
    text: Optional[str] = None,
) -> None:
    """
    Send a single email. Raises on failure — caller decides how to react.
    """
    cfg = get_config(db)
    if not cfg or not is_configured(cfg):
        raise RuntimeError("SMTP no configurado.")

    host = cfg.smtp_host or ""
    port = cfg.smtp_port or 587
    secure = bool(cfg.smtp_secure)

    if cfg.outbound_provider == "office365":
        host = "smtp.office365.com"
        port = 587
        secure = False  # STARTTLS

    if not host:
        raise RuntimeError("SMTP: falta el host en la configuración.")
    if not cfg.smtp_user or not cfg.smtp_pass:
        raise RuntimeError("SMTP: falta usuario o contraseña.")

    from_addr = formataddr((cfg.from_name or "BackupSMC", cfg.from_email or cfg.smtp_user))

    msg = MIMEMultipart("alternative")
    msg["Subject"] = subject
    msg["From"] = from_addr
    msg["To"] = to
    if text:
        msg.attach(MIMEText(text, "plain", "utf-8"))
    msg.attach(MIMEText(html, "html", "utf-8"))

    context = ssl.create_default_context()

    try:
        if secure:
            # SMTPS — implicit TLS (port 465)
            with smtplib.SMTP_SSL(host, port, context=context, timeout=15) as s:
                s.login(cfg.smtp_user, cfg.smtp_pass)
                s.send_message(msg)
        else:
            # Plain SMTP + STARTTLS upgrade (typical 587)
            with smtplib.SMTP(host, port, timeout=15) as s:
                s.ehlo()
                s.starttls(context=context)
                s.ehlo()
                s.login(cfg.smtp_user, cfg.smtp_pass)
                s.send_message(msg)
    except Exception as e:
        log.error("smtp send failed host=%s port=%s: %s", host, port, e)
        raise RuntimeError(f"SMTP error: {e}") from e

    log.info("email sent to=%s subject=%r", to, subject)


# ── Notification bodies ───────────────────────────────────────────────────────

def render_backup_failed(run_node: str, error: str) -> tuple[str, str, str]:
    subject = f"[BackupSMC] Backup fallido · {run_node}"
    html = f"""
    <div style="font-family:-apple-system,Segoe UI,sans-serif;max-width:560px;margin:auto">
      <div style="background:#fee2e2;border-radius:12px;padding:20px">
        <h2 style="color:#b91c1c;margin:0 0 8px 0">❌ Backup fallido</h2>
        <p style="color:#7f1d1d;margin:0"><strong>Nodo:</strong> {run_node}</p>
      </div>
      <div style="padding:16px 4px">
        <p style="color:#374151;font-size:14px">El backup terminó con error:</p>
        <pre style="background:#f3f4f6;border-radius:8px;padding:12px;font-size:12px;white-space:pre-wrap;color:#991b1b">{error}</pre>
      </div>
      <hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0"/>
      <p style="color:#9ca3af;font-size:12px">Enviado por BackupSMC</p>
    </div>
    """
    text = f"Backup fallido · {run_node}\n\n{error}\n\n— BackupSMC"
    return subject, html, text


def render_backup_warning(run_node: str, error: str) -> tuple[str, str, str]:
    subject = f"[BackupSMC] Backup con advertencias · {run_node}"
    html = f"""
    <div style="font-family:-apple-system,Segoe UI,sans-serif;max-width:560px;margin:auto">
      <div style="background:#fef3c7;border-radius:12px;padding:20px">
        <h2 style="color:#b45309;margin:0 0 8px 0">⚠ Backup con advertencias</h2>
        <p style="color:#78350f;margin:0"><strong>Nodo:</strong> {run_node}</p>
      </div>
      <div style="padding:16px 4px">
        <p style="color:#374151;font-size:14px">El backup completó con algunos errores de lectura:</p>
        <pre style="background:#f3f4f6;border-radius:8px;padding:12px;font-size:12px;white-space:pre-wrap;color:#92400e">{error}</pre>
      </div>
      <hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0"/>
      <p style="color:#9ca3af;font-size:12px">Enviado por BackupSMC</p>
    </div>
    """
    text = f"Backup con advertencias · {run_node}\n\n{error}\n\n— BackupSMC"
    return subject, html, text


def render_backup_completed(run_node: str, files: int, bytes_done: int) -> tuple[str, str, str]:
    gb = bytes_done / (1024 ** 3)
    subject = f"[BackupSMC] Backup completado · {run_node}"
    html = f"""
    <div style="font-family:-apple-system,Segoe UI,sans-serif;max-width:560px;margin:auto">
      <div style="background:#dcfce7;border-radius:12px;padding:20px">
        <h2 style="color:#166534;margin:0 0 8px 0">✓ Backup completado</h2>
        <p style="color:#14532d;margin:0"><strong>Nodo:</strong> {run_node}</p>
      </div>
      <div style="padding:16px 4px;color:#374151;font-size:14px">
        <p><strong>{files:,}</strong> archivos respaldados</p>
        <p><strong>{gb:.2f} GB</strong> transferidos</p>
      </div>
      <hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0"/>
      <p style="color:#9ca3af;font-size:12px">Enviado por BackupSMC</p>
    </div>
    """
    text = f"Backup completado · {run_node}\n\n{files:,} archivos · {gb:.2f} GB\n\n— BackupSMC"
    return subject, html, text


def render_test() -> tuple[str, str, str]:
    subject = "[BackupSMC] Prueba de correo"
    html = """
    <div style="font-family:-apple-system,Segoe UI,sans-serif;max-width:480px;margin:auto">
      <h2 style="color:#4f6bff">BackupSMC — Prueba de correo</h2>
      <p style="color:#374151">Si recibes este mensaje, la configuración SMTP está funcionando correctamente.</p>
      <hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0"/>
      <p style="color:#9ca3af;font-size:12px">Enviado por BackupSMC</p>
    </div>
    """
    text = "BackupSMC — Prueba de correo. Si recibes este mensaje, el SMTP funciona."
    return subject, html, text
