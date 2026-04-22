"""
In-app notification service — mirrors SMC Desk's NotificationService.

Notifications are fan-out to every active dashboard user (BackupSMC has
no per-ticket ownership model, so alerts are global).
"""
from typing import Iterable, Optional
from sqlalchemy.orm import Session
from sqlalchemy import func, and_

from app.models import Notification, User, JobRun, SystemSetting
from app import email_service


# ── Preference helpers ────────────────────────────────────────────────────────

def _get_flag(db: Session, key: str, default: bool) -> bool:
    row = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    if not row:
        return default
    return (row.value or "").lower() == "true"


def _get_setting(db: Session, key: str, default: str = "") -> str:
    row = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    return row.value if row and row.value else default


def _send_email_safe(db: Session, to: str, subject: str, html: str, text: str) -> None:
    """Try to send one email. Never raises — failures just get logged."""
    import logging
    try:
        email_service.send_mail(db, to, subject, html, text)
    except Exception as e:
        logging.getLogger("backupsmc.notifications").warning(
            "email notify failed to=%s: %s", to, e
        )


# ── Public API ────────────────────────────────────────────────────────────────

def create_for_user(
    db: Session,
    user_id: int,
    type: str,
    title: str,
    body: Optional[str] = None,
    entity_type: Optional[str] = None,
    entity_id: Optional[str] = None,
) -> Notification:
    n = Notification(
        user_id=user_id,
        type=type,
        title=title,
        body=body,
        entity_type=entity_type,
        entity_id=entity_id,
    )
    db.add(n)
    db.commit()
    db.refresh(n)
    return n


def fanout_to_all_users(
    db: Session,
    type: str,
    title: str,
    body: Optional[str] = None,
    entity_type: Optional[str] = None,
    entity_id: Optional[str] = None,
) -> int:
    """Insert one notification per active user. Returns count inserted."""
    users: Iterable[User] = db.query(User).filter(User.is_active == True).all()  # noqa: E712
    count = 0
    for u in users:
        db.add(Notification(
            user_id=u.id,
            type=type,
            title=title,
            body=body,
            entity_type=entity_type,
            entity_id=entity_id,
        ))
        count += 1
    if count:
        db.commit()
    return count


def list_for_user(db: Session, user_id: int, only_unread: bool = False, limit: int = 50):
    q = db.query(Notification).filter(Notification.user_id == user_id)
    if only_unread:
        q = q.filter(Notification.read == False)  # noqa: E712
    return q.order_by(Notification.created_at.desc()).limit(limit).all()


def unread_count(db: Session, user_id: int) -> int:
    return db.query(func.count(Notification.id)).filter(
        and_(Notification.user_id == user_id, Notification.read == False)  # noqa: E712
    ).scalar() or 0


def mark_read(db: Session, user_id: int, notification_id: int) -> bool:
    n = db.query(Notification).filter(
        Notification.id == notification_id, Notification.user_id == user_id
    ).first()
    if not n:
        return False
    n.read = True
    db.commit()
    return True


def mark_all_read(db: Session, user_id: int) -> int:
    updated = db.query(Notification).filter(
        Notification.user_id == user_id, Notification.read == False  # noqa: E712
    ).update({"read": True})
    db.commit()
    return updated


# ── Domain-specific helpers (called from update_progress hook) ────────────────

def on_job_status_change(db: Session, run: JobRun, previous_status: Optional[str]) -> None:
    """
    Called when a JobRun's status transitions. Respects user-facing flags:
      - notify_on_failure (default true)
      - notify_on_success (default false)

    Warnings always notify (partial-error case is actionable).
    """
    if previous_status == run.status:
        return

    new = run.status
    node_label = run.node_id or "nodo desconocido"
    entity_id = str(run.id)

    email_enabled = _get_flag(db, "notify_email_enabled", False)
    email_to      = _get_setting(db, "notify_email_to", "").strip()
    can_email     = email_enabled and email_to and email_service.is_configured(email_service.get_config(db))

    if new == "failed":
        if not _get_flag(db, "notify_on_failure", True):
            return
        fanout_to_all_users(
            db,
            type="backup_failed",
            title=f"Backup fallido · {node_label}",
            body=(run.error_message or "El backup terminó con error.")[:500],
            entity_type="job_run",
            entity_id=entity_id,
        )
        if can_email:
            subject, html, text = email_service.render_backup_failed(
                node_label, run.error_message or "Error sin detalle."
            )
            _send_email_safe(db, email_to, subject, html, text)

    elif new == "warning":
        # Warnings are always useful; not gated by flags.
        fanout_to_all_users(
            db,
            type="backup_warning",
            title=f"Backup con advertencias · {node_label}",
            body=(run.error_message or "Algunos archivos no pudieron copiarse.")[:500],
            entity_type="job_run",
            entity_id=entity_id,
        )
        if can_email:
            subject, html, text = email_service.render_backup_warning(
                node_label, run.error_message or "Errores de lectura parciales."
            )
            _send_email_safe(db, email_to, subject, html, text)

    elif new == "completed":
        if not _get_flag(db, "notify_on_success", False):
            return
        fanout_to_all_users(
            db,
            type="backup_completed",
            title=f"Backup completado · {node_label}",
            body=f"{run.files_done:,} archivos respaldados.",
            entity_type="job_run",
            entity_id=entity_id,
        )
        if can_email:
            subject, html, text = email_service.render_backup_completed(
                node_label, run.files_done or 0, run.bytes_done or 0
            )
            _send_email_safe(db, email_to, subject, html, text)
