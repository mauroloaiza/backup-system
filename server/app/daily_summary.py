"""
Daily Summary — builds and sends the daily backup digest email.

Runs from APScheduler (see app/scheduler.py) at a configurable hour, but the
core function is also exposed via `POST /api/v1/mail-settings/test-daily` so
users can preview it on demand.
"""
from __future__ import annotations
import logging
from datetime import datetime, timezone, timedelta
from collections import defaultdict
from typing import Optional

from sqlalchemy.orm import Session
from sqlalchemy import func

from app.models import Node, JobRun, SystemSetting
from app import email_service

log = logging.getLogger("backupsmc.daily_summary")


# ── Data aggregation ─────────────────────────────────────────────────────────

def _humanize_bytes(n: int) -> str:
    n = float(n or 0)
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if n < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


def _humanize_duration(seconds: float) -> str:
    s = int(seconds or 0)
    if s < 60:
        return f"{s}s"
    m, s = divmod(s, 60)
    if m < 60:
        return f"{m}m {s}s"
    h, m = divmod(m, 60)
    return f"{h}h {m}m"


def _aware_utc(dt):
    if dt is None:
        return None
    return dt if dt.tzinfo is not None else dt.replace(tzinfo=timezone.utc)


def build_summary(db: Session, window_hours: int = 24) -> dict:
    """Aggregate runs from the last `window_hours` into a per-node breakdown."""
    since = datetime.now(timezone.utc) - timedelta(hours=window_hours)
    # SQLite may store naive datetimes; filter loosely then compare in Python with tz-normalization.
    all_runs = db.query(JobRun).all()
    runs = [r for r in all_runs if r.started_at and _aware_utc(r.started_at) >= since]

    per_node: dict[str, dict] = defaultdict(lambda: {
        "runs": 0, "completed": 0, "failed": 0, "warning": 0,
        "running": 0, "bytes": 0, "last_at": None,
    })
    for r in runs:
        key = r.node_id or "—"
        b = per_node[key]
        b["runs"] += 1
        if r.status == "completed": b["completed"] += 1
        elif r.status == "failed":  b["failed"] += 1
        elif r.status == "warning": b["warning"] += 1
        elif r.status == "running": b["running"] += 1
        b["bytes"] += r.bytes_done or 0
        started = _aware_utc(r.started_at)
        if started and (not b["last_at"] or started > b["last_at"]):
            b["last_at"] = started

    # Resolve node names
    node_map = {n.id: n for n in db.query(Node).all()}
    rows = []
    for node_id, s in per_node.items():
        node = node_map.get(node_id)
        rows.append({
            "node_id": node_id,
            "node_name": node.name if node else node_id,
            "hostname": node.hostname if node else "",
            **s,
        })
    rows.sort(key=lambda x: (-x["failed"], -x["runs"], x["node_name"]))

    # Nodes offline: last_seen older than 5 minutes AND not seen in last 24h window counted as offline
    offline_cutoff = datetime.now(timezone.utc) - timedelta(minutes=5)

    def _aware(dt):
        if dt is None:
            return None
        return dt if dt.tzinfo is not None else dt.replace(tzinfo=timezone.utc)

    offline_nodes = [
        {"name": n.name, "hostname": n.hostname, "last_seen": n.last_seen}
        for n in node_map.values()
        if not n.last_seen or _aware(n.last_seen) < offline_cutoff
    ]
    offline_nodes.sort(key=lambda x: x["name"])

    totals = {
        "runs": len(runs),
        "completed": sum(1 for r in runs if r.status == "completed"),
        "failed": sum(1 for r in runs if r.status == "failed"),
        "warning": sum(1 for r in runs if r.status == "warning"),
        "bytes": sum(r.bytes_done or 0 for r in runs),
        "nodes_active": len(per_node),
        "nodes_total": len(node_map),
        "nodes_offline": len(offline_nodes),
    }
    totals["success_rate"] = round(
        (totals["completed"] / totals["runs"] * 100) if totals["runs"] > 0 else 0.0, 1
    )

    return {
        "window_hours": window_hours,
        "generated_at": datetime.now(timezone.utc),
        "totals": totals,
        "per_node": rows,
        "offline_nodes": offline_nodes,
    }


# ── Rendering ────────────────────────────────────────────────────────────────

def render(summary: dict) -> tuple[str, str, str]:
    t = summary["totals"]
    gen = summary["generated_at"].strftime("%Y-%m-%d %H:%M UTC")
    subject = f"[BackupSMC] Resumen diario · {t['completed']}✓ / {t['failed']}✗ · {_humanize_bytes(t['bytes'])}"

    status_color = "#16a34a" if t["failed"] == 0 else ("#f59e0b" if t["failed"] < 3 else "#dc2626")
    status_label = "Todo en orden" if t["failed"] == 0 else (
        "Algunas fallas" if t["failed"] < 3 else "Múltiples fallas"
    )

    rows_html = ""
    for r in summary["per_node"][:30]:
        last_txt = r["last_at"].strftime("%H:%M") if r["last_at"] else "—"
        fail_pill = (
            f'<span style="background:#fee2e2;color:#b91c1c;padding:2px 8px;border-radius:999px;font-size:11px">{r["failed"]}</span>'
            if r["failed"] else
            '<span style="color:#9ca3af">0</span>'
        )
        rows_html += f"""
        <tr>
          <td style="padding:10px 8px;border-top:1px solid #f1f5f9">
            <div style="font-weight:600;color:#111827">{r['node_name']}</div>
            <div style="font-size:11px;color:#6b7280">{r['hostname']}</div>
          </td>
          <td style="padding:10px 8px;border-top:1px solid #f1f5f9;text-align:center;color:#059669;font-weight:600">{r['completed']}</td>
          <td style="padding:10px 8px;border-top:1px solid #f1f5f9;text-align:center">{fail_pill}</td>
          <td style="padding:10px 8px;border-top:1px solid #f1f5f9;text-align:right;color:#374151">{_humanize_bytes(r['bytes'])}</td>
          <td style="padding:10px 8px;border-top:1px solid #f1f5f9;text-align:right;font-size:12px;color:#6b7280">{last_txt}</td>
        </tr>
        """

    offline_html = ""
    if summary["offline_nodes"]:
        items = "".join(
            f'<li style="color:#78350f;font-size:13px;margin-bottom:4px">'
            f'<strong>{n["name"]}</strong> '
            f'<span style="color:#b45309;font-size:11px">({n["hostname"]})</span></li>'
            for n in summary["offline_nodes"][:10]
        )
        offline_html = f"""
        <div style="background:#fef3c7;border-radius:10px;padding:14px 18px;margin:16px 0">
          <div style="color:#b45309;font-weight:600;margin-bottom:6px">⚠ Nodos sin contacto ({len(summary['offline_nodes'])})</div>
          <ul style="margin:4px 0 0 18px;padding:0">{items}</ul>
        </div>
        """

    html = f"""
    <div style="font-family:-apple-system,Segoe UI,Roboto,sans-serif;max-width:680px;margin:auto;color:#111827">
      <div style="background:linear-gradient(135deg,#4f6bff,#7c3aed);border-radius:14px;padding:24px;color:white">
        <div style="font-size:12px;opacity:0.8;letter-spacing:0.5px;text-transform:uppercase">Resumen diario · BackupSMC</div>
        <h2 style="margin:6px 0 2px 0;font-size:22px">{status_label}</h2>
        <div style="font-size:12px;opacity:0.85">{gen} · últimas {summary['window_hours']}h</div>
      </div>

      <div style="display:flex;gap:10px;margin:18px 0">
        <div style="flex:1;background:#f9fafb;border-radius:10px;padding:14px">
          <div style="font-size:11px;color:#6b7280;text-transform:uppercase">Runs</div>
          <div style="font-size:22px;font-weight:700;color:#111827">{t['runs']}</div>
        </div>
        <div style="flex:1;background:#ecfdf5;border-radius:10px;padding:14px">
          <div style="font-size:11px;color:#065f46;text-transform:uppercase">OK</div>
          <div style="font-size:22px;font-weight:700;color:#059669">{t['completed']}</div>
        </div>
        <div style="flex:1;background:#fef2f2;border-radius:10px;padding:14px">
          <div style="font-size:11px;color:#991b1b;text-transform:uppercase">Fallos</div>
          <div style="font-size:22px;font-weight:700;color:#dc2626">{t['failed']}</div>
        </div>
        <div style="flex:1;background:#eff6ff;border-radius:10px;padding:14px">
          <div style="font-size:11px;color:#1e40af;text-transform:uppercase">Datos</div>
          <div style="font-size:18px;font-weight:700;color:#1d4ed8">{_humanize_bytes(t['bytes'])}</div>
        </div>
      </div>

      <div style="background:white;border:1px solid #e5e7eb;border-radius:12px;overflow:hidden">
        <div style="background:#f9fafb;padding:12px 16px;font-size:12px;color:#6b7280;text-transform:uppercase;letter-spacing:0.5px">
          Actividad por nodo · éxito {t['success_rate']}% · {t['nodes_active']}/{t['nodes_total']} nodos activos
        </div>
        <table style="width:100%;border-collapse:collapse">
          <thead>
            <tr style="background:#fafbfc">
              <th style="padding:8px;text-align:left;font-size:11px;color:#6b7280;text-transform:uppercase">Nodo</th>
              <th style="padding:8px;text-align:center;font-size:11px;color:#6b7280;text-transform:uppercase">OK</th>
              <th style="padding:8px;text-align:center;font-size:11px;color:#6b7280;text-transform:uppercase">Fail</th>
              <th style="padding:8px;text-align:right;font-size:11px;color:#6b7280;text-transform:uppercase">Datos</th>
              <th style="padding:8px;text-align:right;font-size:11px;color:#6b7280;text-transform:uppercase">Últ.</th>
            </tr>
          </thead>
          <tbody>{rows_html or '<tr><td colspan="5" style="padding:20px;text-align:center;color:#9ca3af">Sin actividad en el período</td></tr>'}</tbody>
        </table>
      </div>

      {offline_html}

      <hr style="border:none;border-top:1px solid #e5e7eb;margin:22px 0"/>
      <p style="color:#9ca3af;font-size:11px;text-align:center">
        Resumen generado automáticamente por BackupSMC · Puedes desactivar este correo en Configuración → Notificaciones
      </p>
    </div>
    """

    # Plain text fallback
    lines = [
        f"BackupSMC — Resumen diario ({gen})",
        f"Ventana: últimas {summary['window_hours']}h · {status_label}",
        "",
        f"Runs: {t['runs']}  ·  OK: {t['completed']}  ·  Fallos: {t['failed']}  ·  Datos: {_humanize_bytes(t['bytes'])}",
        f"Éxito: {t['success_rate']}%  ·  Nodos activos: {t['nodes_active']}/{t['nodes_total']}",
        "",
        "Por nodo:",
    ]
    for r in summary["per_node"][:30]:
        lines.append(f"  · {r['node_name']:<28} OK={r['completed']:<3} Fail={r['failed']:<3} {_humanize_bytes(r['bytes'])}")
    if summary["offline_nodes"]:
        lines += ["", f"⚠ Nodos sin contacto ({len(summary['offline_nodes'])}):"]
        for n in summary["offline_nodes"][:10]:
            lines.append(f"  · {n['name']} ({n['hostname']})")
    text = "\n".join(lines)

    return subject, html, text


# ── Sending ──────────────────────────────────────────────────────────────────

def _get_setting(db: Session, key: str, default: str = "") -> str:
    s = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    return s.value if s else default


def send_daily_summary(db: Session, to_override: Optional[str] = None) -> dict:
    """
    Build and send the daily digest. Returns a status dict.

    Honors these settings:
      - notify_daily_summary (bool)       — must be "true" to auto-send
      - notify_email_to       (string)    — destination (can be overridden)
    """
    # Always build so callers can preview even when the cron is off.
    summary = build_summary(db, window_hours=24)
    subject, html, text = render(summary)

    to = (to_override or "").strip() or _get_setting(db, "notify_email_to", "").strip()
    if not to:
        return {"ok": False, "sent": False, "reason": "Sin destinatario (notify_email_to vacío)", "summary": summary["totals"]}

    enabled = _get_setting(db, "notify_daily_summary", "false") == "true"
    if not enabled and not to_override:
        return {"ok": False, "sent": False, "reason": "notify_daily_summary=false", "summary": summary["totals"]}

    try:
        email_service.send_mail(db, to, subject, html, text)
    except Exception as e:
        log.error("daily summary send failed: %s", e)
        return {"ok": False, "sent": False, "reason": str(e), "summary": summary["totals"]}

    log.info("daily summary sent to=%s totals=%s", to, summary["totals"])
    return {"ok": True, "sent": True, "sent_to": to, "summary": summary["totals"]}
