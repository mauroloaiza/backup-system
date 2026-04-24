"""
BackupSMC — Server Entry Point
"""
from datetime import datetime, timezone, timedelta
from typing import Optional
from fastapi import FastAPI, Depends, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
from sqlalchemy.orm import Session
from sqlalchemy import func, text
import os

from app.database import engine, get_db, Base, is_postgres
from app import bootstrap_migrate
from app.models import Node, JobRun, User, AgentToken, SystemSetting, Notification, EmailSettings, NodeConfig, RestoreRequest
from app.schemas import (
    NodeRegister, NodeOut, NodeSourcePathsUpdate, DestinationAggregate,
    NodeConfigOut, NodeConfigUpdate, NodeConfigPayload, NodeConfigPull,
    ProgressUpdate,
    JobRunOut,
    DashboardStats,
    HistoryStats, DailyStatPoint,
    AgentTokenOut, AgentTokenCreate,
    SettingsOut, SettingsUpdate,
    LoginRequest, Token, UserOut,
    NotificationOut, NotificationList,
    EmailSettingsOut, EmailSettingsUpdate, EmailTestRequest,
    RestoreRequestCreate, RestoreRequestOut, RestorePendingOut, RestoreProgressUpdate,
)
from app.auth import (
    verify_password, hash_password, create_access_token,
    get_current_user, require_agent_token,
)
from app import notifications_service, email_service, daily_summary, scheduler

# ── DB init ───────────────────────────────────────────────────────────────────

Base.metadata.create_all(bind=engine)

# Add missing columns for existing DBs (idempotent).
#
# For SQLite we swallow errors (ALTER TABLE fails if the column already exists).
# For Postgres we use ADD COLUMN IF NOT EXISTS which is a proper no-op.
def _run_migrations():
    with engine.connect() as conn:
        additions = [
            ("nodes", "source_paths_json", "TEXT"),
            ("nodes", "destinations_json", "TEXT"),
        ]
        for table, col, ctype in additions:
            if is_postgres():
                stmt = f'ALTER TABLE {table} ADD COLUMN IF NOT EXISTS {col} {ctype}'
            else:
                stmt = f'ALTER TABLE {table} ADD COLUMN {col} {ctype}'
            try:
                conn.execute(text(stmt))
                conn.commit()
            except Exception:
                # SQLite: column already exists. Safe to ignore.
                pass

# One-shot data migration from /app/backupsmc.db → Postgres on first boot.
# No-op if already migrated or SQLite file absent or SQLite engine still active.
bootstrap_migrate.run_if_needed()


def _seed_agent_token():
    """If no agent tokens exist, create one from env or generate a default."""
    import secrets
    env_token = os.getenv("AGENT_TOKEN", "")
    db = next(get_db())
    try:
        if db.query(AgentToken).count() == 0:
            token_val = env_token if env_token else secrets.token_urlsafe(32)
            db.add(AgentToken(name="default", token=token_val, is_active=True))
            db.commit()
    finally:
        db.close()

_run_migrations()

# Seed default admin user from env
def _seed_admin():
    admin_user = os.getenv("ADMIN_USER", "admin")
    admin_pass = os.getenv("ADMIN_PASSWORD", "backupsmc2024")
    db = next(get_db())
    try:
        if not db.query(User).filter(User.username == admin_user).first():
            db.add(User(username=admin_user, hashed_password=hash_password(admin_pass)))
            db.commit()
    finally:
        db.close()

_seed_admin()
_seed_agent_token()

# ── App ───────────────────────────────────────────────────────────────────────

app = FastAPI(
    title="BackupSMC API",
    description="API REST del sistema de backup empresarial BackupSMC",
    version="0.11.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

@app.on_event("startup")
def _on_startup():
    try:
        scheduler.start()
    except Exception as e:
        import logging
        logging.getLogger("backupsmc").error("scheduler start failed: %s", e)


@app.on_event("shutdown")
def _on_shutdown():
    try:
        scheduler.shutdown()
    except Exception:
        pass


app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Serve the dashboard SPA if the dist folder exists
_dashboard_dist = os.path.join(os.path.dirname(__file__), "..", "dashboard")
if os.path.isdir(_dashboard_dist):
    app.mount("/dashboard", StaticFiles(directory=_dashboard_dist, html=True), name="dashboard")


# ── Health (public) ───────────────────────────────────────────────────────────

@app.get("/health", tags=["health"])
def health_check():
    return {"status": "ok", "service": "backupsmc-server", "version": "0.11.0"}


# ── Auth ──────────────────────────────────────────────────────────────────────

@app.post("/api/v1/auth/login", response_model=Token, tags=["auth"])
def login(body: LoginRequest, db: Session = Depends(get_db)):
    user = db.query(User).filter(User.username == body.username).first()
    if not user or not verify_password(body.password, user.hashed_password):
        raise HTTPException(status_code=401, detail="Usuario o contraseña incorrectos")
    token = create_access_token({"sub": user.username})
    return Token(access_token=token)

@app.get("/api/v1/auth/me", response_model=UserOut, tags=["auth"])
def me(current_user: User = Depends(get_current_user)):
    return current_user


# ── Nodes (agent → register, protected by agent token) ───────────────────────

@app.post("/api/v1/nodes/register", response_model=NodeOut, tags=["nodes"],
          dependencies=[Depends(require_agent_token)])
def register_node(body: NodeRegister, db: Session = Depends(get_db)):
    """Agent calls this on startup to register / heartbeat."""
    node = db.query(Node).filter(Node.id == body.id).first()
    if node:
        node.name = body.name
        node.hostname = body.hostname
        node.os = body.os
        node.agent_version = body.agent_version
        node.status = "online"
        node.last_seen = datetime.now(timezone.utc)
        if body.source_paths:
            node.source_paths = body.source_paths
        if body.destinations is not None:
            node.destinations = [d.model_dump() for d in body.destinations]
    else:
        node = Node(
            id=body.id,
            name=body.name,
            hostname=body.hostname,
            os=body.os,
            agent_version=body.agent_version,
            status="online",
        )
        node.source_paths = body.source_paths
        if body.destinations is not None:
            node.destinations = [d.model_dump() for d in body.destinations]
        db.add(node)
    db.commit()
    db.refresh(node)
    return node


@app.get("/api/v1/nodes", response_model=list[NodeOut], tags=["nodes"],
         dependencies=[Depends(get_current_user)])
def list_nodes(db: Session = Depends(get_db)):
    return db.query(Node).order_by(Node.last_seen.desc()).all()


@app.get("/api/v1/nodes/{node_id}", response_model=NodeOut, tags=["nodes"],
         dependencies=[Depends(get_current_user)])
def get_node(node_id: str, db: Session = Depends(get_db)):
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    return node


@app.put("/api/v1/nodes/{node_id}/source-paths", response_model=NodeOut, tags=["nodes"],
         dependencies=[Depends(get_current_user)])
def update_node_source_paths(
    node_id: str,
    body: NodeSourcePathsUpdate,
    db: Session = Depends(get_db),
):
    """Web UI calls this to add/update source paths for a node."""
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    node.source_paths = body.source_paths
    db.commit()
    db.refresh(node)
    return node


@app.delete("/api/v1/nodes/{node_id}", tags=["nodes"],
            dependencies=[Depends(get_current_user)])
def delete_node(node_id: str, db: Session = Depends(get_db)):
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    db.delete(node)
    db.commit()
    return {"ok": True}


# ── Remote agent config (UI editor + agent pull) ─────────────────────────────

def _node_config_row(db: Session, node_id: str) -> NodeConfig:
    """Get or lazily create the NodeConfig row for a node."""
    cfg = db.query(NodeConfig).filter(NodeConfig.node_id == node_id).first()
    if not cfg:
        cfg = NodeConfig(
            node_id=node_id,
            version=1,
            payload_json=NodeConfigPayload().model_dump_json(),
        )
        db.add(cfg)
        db.commit()
        db.refresh(cfg)
    return cfg


def _config_out(cfg: NodeConfig) -> NodeConfigOut:
    payload = NodeConfigPayload.model_validate(cfg.payload or {})
    return NodeConfigOut(
        node_id=cfg.node_id,
        version=cfg.version,
        payload=payload,
        updated_by=cfg.updated_by,
        updated_at=cfg.updated_at,
        last_pulled_version=cfg.last_pulled_version or 0,
        last_pulled_at=cfg.last_pulled_at,
        in_sync=(cfg.last_pulled_version or 0) >= cfg.version,
    )


@app.get("/api/v1/nodes/{node_id}/config", response_model=NodeConfigOut, tags=["nodes"],
         dependencies=[Depends(get_current_user)])
def get_node_config(node_id: str, db: Session = Depends(get_db)):
    """UI: read the stored remote config for a node."""
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    cfg = _node_config_row(db, node_id)
    return _config_out(cfg)


@app.put("/api/v1/nodes/{node_id}/config", response_model=NodeConfigOut, tags=["nodes"])
def update_node_config(
    node_id: str,
    body: NodeConfigUpdate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """
    UI: write new config for a node. Increments `version`, which causes the
    agent to pick it up on its next poll (and write it to its local agent.yaml).

    Also syncs `source_paths` to the Node row so the existing /nodes UI stays
    accurate without waiting for the next register heartbeat.
    """
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")

    cfg = _node_config_row(db, node_id)
    cfg.payload = body.payload.model_dump()
    cfg.version = (cfg.version or 0) + 1
    cfg.updated_by = current_user.username
    cfg.updated_at = datetime.now(timezone.utc)

    # Keep node.source_paths aligned so the Nodes page reflects the change
    # immediately (not waiting for the agent to re-register).
    node.source_paths = body.payload.source_paths

    db.commit()
    db.refresh(cfg)
    return _config_out(cfg)


@app.get("/api/v1/nodes/{node_id}/config/pull", tags=["nodes"],
         dependencies=[Depends(require_agent_token)])
def pull_node_config(
    node_id: str,
    current_version: int = Query(0, ge=0),
    db: Session = Depends(get_db),
):
    """
    Agent endpoint: called periodically with the agent's currently-applied
    version. If the server has a newer version, returns the full payload
    ({node_id, version, payload}). Otherwise returns 204 No Content so the
    agent can skip parsing.

    Also marks `last_pulled_*` for UI "Applied / Pending" indicators.
    """
    cfg = _node_config_row(db, node_id)

    # Record the poll regardless — useful to know the agent is alive for config.
    cfg.last_pulled_version = current_version
    cfg.last_pulled_at = datetime.now(timezone.utc)
    db.commit()

    if current_version >= cfg.version:
        from fastapi import Response
        return Response(status_code=204)

    payload = NodeConfigPayload.model_validate(cfg.payload or {})
    return NodeConfigPull(node_id=node_id, version=cfg.version, payload=payload)


# ── Destinations (aggregated from node-reported configs + run history) ───────

@app.get("/api/v1/destinations", response_model=list[DestinationAggregate], tags=["destinations"],
         dependencies=[Depends(get_current_user)])
def list_destinations(db: Session = Depends(get_db)):
    """
    Return a flat list of (node × destination) rows with usage stats.

    Destinations come from what each agent reports in `/nodes/register`
    (nothing is stored server-side — this is a read-only view of agent truth).
    """
    now = datetime.now(timezone.utc)

    def _node_status(n: Node) -> str:
        if not n.last_seen:
            return "offline"
        # Normalize to tz-aware UTC (SQLite can return naive)
        last = n.last_seen if n.last_seen.tzinfo else n.last_seen.replace(tzinfo=timezone.utc)
        age = (now - last).total_seconds()
        if age < 600:        # <10 min
            return "online"
        if age < 1800:       # 10-30 min
            return "stale"
        return "offline"

    # Pre-compute per-node aggregates from completed/warning runs
    rows = (
        db.query(
            JobRun.node_id,
            func.sum(JobRun.bytes_done).label("bytes"),
            func.count(JobRun.id).label("runs"),
            func.max(JobRun.started_at).label("last_at"),
        )
        .filter(JobRun.node_id.isnot(None))
        .group_by(JobRun.node_id)
        .all()
    )
    agg_map: dict[str, dict] = {r.node_id: {
        "bytes": int(r.bytes or 0),
        "runs": int(r.runs or 0),
        "last_at": r.last_at,
    } for r in rows}

    # Last-run status per node (separate query — cheap, < 100 nodes typical)
    last_status_map: dict[str, str] = {}
    for nid in agg_map.keys():
        last_run = (
            db.query(JobRun.status)
              .filter(JobRun.node_id == nid)
              .order_by(JobRun.started_at.desc())
              .first()
        )
        if last_run:
            last_status_map[nid] = last_run[0]

    out: list[DestinationAggregate] = []
    for node in db.query(Node).all():
        dests = node.destinations
        if not dests:
            continue
        ns = _node_status(node)
        ag = agg_map.get(node.id, {"bytes": 0, "runs": 0, "last_at": None})
        for d in dests:
            out.append(DestinationAggregate(
                node_id=node.id,
                node_name=node.name,
                hostname=node.hostname,
                node_status=ns,
                type=d.get("type", "local"),
                target=d.get("target", ""),
                details=d.get("details"),
                bytes_backed_up=ag["bytes"],
                runs_count=ag["runs"],
                last_backup_at=ag["last_at"],
                last_status=last_status_map.get(node.id),
            ))
    # Sort: online first, then by bytes desc
    status_order = {"online": 0, "stale": 1, "offline": 2}
    out.sort(key=lambda x: (status_order.get(x.node_status, 3), -x.bytes_backed_up))
    return out


# ── Jobs (agent → progress, protected by agent token) ────────────────────────

@app.post("/api/v1/jobs/{job_id}/progress", tags=["jobs"],
          dependencies=[Depends(require_agent_token)])
def update_progress(job_id: str, body: ProgressUpdate, db: Session = Depends(get_db)):
    """Agent reports backup progress."""
    if body.node_id:
        node = db.query(Node).filter(Node.id == body.node_id).first()
        if not node:
            node = Node(id=body.node_id, name=body.node_id, hostname=body.node_id)
            db.add(node)
        else:
            node.last_seen = datetime.now(timezone.utc)
            node.status = "online"

    run = db.query(JobRun).filter(JobRun.job_id == job_id).first()
    previous_status = run.status if run else None
    if not run:
        run = JobRun(
            job_id=job_id,
            node_id=body.node_id,
            started_at=body.started_at or datetime.now(timezone.utc),
        )
        db.add(run)

    run.status = body.status
    run.files_total = body.files_total
    run.files_done = body.files_done
    run.bytes_total = body.bytes_total
    run.bytes_done = body.bytes_done
    run.current_file = body.current_file
    run.error_message = body.error_message
    run.updated_at = datetime.now(timezone.utc)

    if body.status in ("completed", "failed", "warning"):
        run.finished_at = datetime.now(timezone.utc)

    db.commit()
    db.refresh(run)

    # Fire in-app notifications on status transitions.
    try:
        notifications_service.on_job_status_change(db, run, previous_status)
    except Exception:
        # Never let notification failures break the agent heartbeat.
        pass

    return {"ok": True, "job_id": job_id}


@app.get("/api/v1/jobs", response_model=list[JobRunOut], tags=["jobs"],
         dependencies=[Depends(get_current_user)])
def list_jobs(
    status: Optional[str] = Query(None),
    node_id: Optional[str] = Query(None),
    limit: int = Query(50, le=200),
    db: Session = Depends(get_db),
):
    q = db.query(JobRun)
    if status:
        q = q.filter(JobRun.status == status)
    if node_id:
        q = q.filter(JobRun.node_id == node_id)
    return q.order_by(JobRun.started_at.desc()).limit(limit).all()


@app.get("/api/v1/jobs/{job_id}", response_model=JobRunOut, tags=["jobs"],
         dependencies=[Depends(get_current_user)])
def get_job(job_id: str, db: Session = Depends(get_db)):
    run = db.query(JobRun).filter(JobRun.job_id == job_id).first()
    if not run:
        raise HTTPException(status_code=404, detail="Job not found")
    return run


# ── Dashboard (protected) ─────────────────────────────────────────────────────

@app.get("/api/v1/dashboard", response_model=DashboardStats, tags=["dashboard"],
         dependencies=[Depends(get_current_user)])
def dashboard(db: Session = Depends(get_db)):
    today = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)

    total_nodes = db.query(func.count(Node.id)).scalar() or 0
    cutoff = datetime.now(timezone.utc) - timedelta(minutes=5)
    nodes_online = db.query(func.count(Node.id)).filter(Node.last_seen >= cutoff).scalar() or 0
    nodes_offline = total_nodes - nodes_online

    total_runs = db.query(func.count(JobRun.id)).scalar() or 0
    runs_today = db.query(func.count(JobRun.id)).filter(JobRun.started_at >= today).scalar() or 0
    runs_completed = db.query(func.count(JobRun.id)).filter(JobRun.status == "completed").scalar() or 0
    runs_failed = db.query(func.count(JobRun.id)).filter(JobRun.status == "failed").scalar() or 0
    runs_running = db.query(func.count(JobRun.id)).filter(JobRun.status == "running").scalar() or 0

    bytes_today = db.query(func.sum(JobRun.bytes_done)).filter(JobRun.started_at >= today).scalar() or 0
    bytes_total = db.query(func.sum(JobRun.bytes_done)).scalar() or 0

    recent = db.query(JobRun).order_by(JobRun.started_at.desc()).limit(10).all()
    running = db.query(JobRun).filter(JobRun.status == "running").order_by(JobRun.started_at.desc()).all()

    return DashboardStats(
        total_nodes=total_nodes,
        nodes_online=nodes_online,
        nodes_offline=nodes_offline,
        total_runs=total_runs,
        runs_today=runs_today,
        runs_completed=runs_completed,
        runs_failed=runs_failed,
        runs_running=runs_running,
        bytes_backed_up_today=bytes_today,
        bytes_backed_up_total=bytes_total,
        recent_runs=recent,
        running_now=running,
    )


# ── History stats ─────────────────────────────────────────────────────────────

@app.get("/api/v1/stats/history", response_model=HistoryStats, tags=["stats"],
         dependencies=[Depends(get_current_user)])
def history_stats(days: int = Query(30, ge=7, le=365), db: Session = Depends(get_db)):
    """Return per-day aggregated stats for the last N days."""
    from collections import defaultdict

    since = datetime.now(timezone.utc) - timedelta(days=days)
    runs = db.query(JobRun).filter(JobRun.started_at >= since).all()

    # aggregate by date
    by_day: dict[str, dict] = defaultdict(lambda: {
        "total": 0, "completed": 0, "failed": 0, "bytes": 0, "dur_sum": 0.0, "dur_count": 0
    })
    for r in runs:
        d = r.started_at.strftime("%Y-%m-%d")
        by_day[d]["total"] += 1
        if r.status == "completed":
            by_day[d]["completed"] += 1
        elif r.status == "failed":
            by_day[d]["failed"] += 1
        by_day[d]["bytes"] += r.bytes_done or 0
        if r.duration_seconds:
            by_day[d]["dur_sum"] += r.duration_seconds
            by_day[d]["dur_count"] += 1

    # fill all days in range (including zeros)
    points = []
    for i in range(days - 1, -1, -1):
        d = (datetime.now(timezone.utc) - timedelta(days=i)).strftime("%Y-%m-%d")
        s = by_day.get(d, {"total": 0, "completed": 0, "failed": 0, "bytes": 0, "dur_sum": 0.0, "dur_count": 0})
        dur_avg = (s["dur_sum"] / s["dur_count"]) if s["dur_count"] > 0 else 0.0
        points.append(DailyStatPoint(
            date=d,
            total=s["total"],
            completed=s["completed"],
            failed=s["failed"],
            bytes=s["bytes"],
            duration_avg=round(dur_avg, 1),
        ))

    total_runs = len(runs)
    completed_runs = sum(1 for r in runs if r.status == "completed")
    success_rate = round((completed_runs / total_runs * 100) if total_runs > 0 else 0.0, 1)
    total_bytes = sum(r.bytes_done or 0 for r in runs)
    all_durations = [r.duration_seconds for r in runs if r.duration_seconds]
    avg_duration = round(sum(all_durations) / len(all_durations), 1) if all_durations else 0.0

    return HistoryStats(
        points=points,
        success_rate=success_rate,
        avg_duration=avg_duration,
        total_bytes=total_bytes,
        total_runs=total_runs,
    )


# ── Agent Tokens ──────────────────────────────────────────────────────────────

@app.get("/api/v1/tokens", response_model=list[AgentTokenOut], tags=["tokens"],
         dependencies=[Depends(get_current_user)])
def list_tokens(db: Session = Depends(get_db)):
    return db.query(AgentToken).order_by(AgentToken.created_at.desc()).all()


@app.post("/api/v1/tokens", response_model=AgentTokenOut, tags=["tokens"],
          dependencies=[Depends(get_current_user)])
def create_token(body: AgentTokenCreate, db: Session = Depends(get_db)):
    import secrets
    token = AgentToken(name=body.name, token=secrets.token_urlsafe(32), is_active=True)
    db.add(token)
    db.commit()
    db.refresh(token)
    return token


@app.delete("/api/v1/tokens/{token_id}", tags=["tokens"],
            dependencies=[Depends(get_current_user)])
def revoke_token(token_id: int, db: Session = Depends(get_db)):
    token = db.query(AgentToken).filter(AgentToken.id == token_id).first()
    if not token:
        raise HTTPException(status_code=404, detail="Token not found")
    token.is_active = False
    db.commit()
    return {"ok": True}


# ── Settings ──────────────────────────────────────────────────────────────────

def _get_setting(db: Session, key: str, default: str = "") -> str:
    s = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    return s.value if s else default


def _set_setting(db: Session, key: str, value: str):
    s = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    if s:
        s.value = value
    else:
        db.add(SystemSetting(key=key, value=value))
    db.commit()


@app.get("/api/v1/settings", response_model=SettingsOut, tags=["settings"],
         dependencies=[Depends(get_current_user)])
def get_settings(db: Session = Depends(get_db)):
    first_token = db.query(AgentToken).filter(AgentToken.is_active == True).first()  # noqa: E712
    return SettingsOut(
        server_version="0.8.0",
        agent_token=first_token.token if first_token else None,
        notify_email_enabled=_get_setting(db, "notify_email_enabled", "false") == "true",
        notify_email_to=_get_setting(db, "notify_email_to", ""),
        notify_on_failure=_get_setting(db, "notify_on_failure", "true") == "true",
        notify_on_success=_get_setting(db, "notify_on_success", "false") == "true",
        notify_daily_summary=_get_setting(db, "notify_daily_summary", "false") == "true",
    )


# ── Mail (SMTP) settings ──────────────────────────────────────────────────────

MASK = "••••••••"


def _mail_cfg_to_out(cfg: EmailSettings | None) -> EmailSettingsOut:
    if not cfg:
        return EmailSettingsOut(configured=False)
    return EmailSettingsOut(
        outbound_provider=cfg.outbound_provider or "smtp",
        from_name=cfg.from_name or "BackupSMC",
        from_email=cfg.from_email,
        smtp_host=cfg.smtp_host,
        smtp_port=cfg.smtp_port or 587,
        smtp_secure=bool(cfg.smtp_secure),
        smtp_user=cfg.smtp_user,
        smtp_pass=MASK if cfg.smtp_pass else None,
        configured=email_service.is_configured(cfg),
    )


@app.get("/api/v1/mail-settings", response_model=EmailSettingsOut, tags=["mail-settings"],
         dependencies=[Depends(get_current_user)])
def get_mail_settings(db: Session = Depends(get_db)):
    return _mail_cfg_to_out(email_service.get_config(db))


@app.put("/api/v1/mail-settings", response_model=EmailSettingsOut, tags=["mail-settings"],
         dependencies=[Depends(get_current_user)])
def update_mail_settings(body: EmailSettingsUpdate, db: Session = Depends(get_db)):
    cfg = email_service.get_config(db)
    if not cfg:
        cfg = EmailSettings()
        db.add(cfg)

    if body.outbound_provider is not None: cfg.outbound_provider = body.outbound_provider
    if body.from_name is not None:         cfg.from_name = body.from_name
    if body.from_email is not None:        cfg.from_email = body.from_email
    if body.smtp_host is not None:         cfg.smtp_host = body.smtp_host
    if body.smtp_port is not None:         cfg.smtp_port = body.smtp_port
    if body.smtp_secure is not None:       cfg.smtp_secure = body.smtp_secure
    if body.smtp_user is not None:         cfg.smtp_user = body.smtp_user
    # Only replace password if a real value was sent (not mask, not empty).
    if body.smtp_pass and body.smtp_pass != MASK:
        cfg.smtp_pass = body.smtp_pass

    db.commit()
    db.refresh(cfg)
    return _mail_cfg_to_out(cfg)


@app.post("/api/v1/mail-settings/test", tags=["mail-settings"],
          dependencies=[Depends(get_current_user)])
def test_mail_settings(body: EmailTestRequest, db: Session = Depends(get_db)):
    # Resolve recipient: explicit > notify_email_to setting
    to = (body.to or "").strip()
    if not to:
        to = _get_setting(db, "notify_email_to", "").strip()
    if not to:
        raise HTTPException(status_code=400, detail="Falta la dirección de destino.")

    subject, html, text = email_service.render_test()
    try:
        email_service.send_mail(db, to, subject, html, text)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    return {"ok": True, "sent_to": to}


@app.post("/api/v1/mail-settings/test-daily", tags=["mail-settings"],
          dependencies=[Depends(get_current_user)])
def test_daily_summary(body: EmailTestRequest, db: Session = Depends(get_db)):
    """
    Build the daily summary and send it now — ignores the `notify_daily_summary`
    flag so users can preview the digest on demand.
    """
    to = (body.to or "").strip() or None
    result = daily_summary.send_daily_summary(db, to_override=to or _get_setting(db, "notify_email_to", "") or None)
    if not result.get("sent"):
        raise HTTPException(status_code=400, detail=result.get("reason", "No se pudo enviar el resumen"))
    return result


@app.get("/api/v1/mail-settings/daily-preview", tags=["mail-settings"],
         dependencies=[Depends(get_current_user)])
def preview_daily_summary(db: Session = Depends(get_db)):
    """Return the aggregated summary JSON without sending email."""
    return daily_summary.build_summary(db, window_hours=24)


# ── Notifications (in-app bell) ───────────────────────────────────────────────

@app.get("/api/v1/notifications", response_model=NotificationList, tags=["notifications"])
def list_notifications(
    unread: bool = Query(False),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    items = notifications_service.list_for_user(db, current_user.id, only_unread=unread)
    return NotificationList(
        notifications=[NotificationOut.model_validate(i) for i in items],
        unread=notifications_service.unread_count(db, current_user.id),
    )


@app.patch("/api/v1/notifications/{notif_id}/read", tags=["notifications"])
def mark_notification_read(
    notif_id: int,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    ok = notifications_service.mark_read(db, current_user.id, notif_id)
    if not ok:
        raise HTTPException(status_code=404, detail="Notification not found")
    return {"ok": True}


@app.post("/api/v1/notifications/mark-all-read", tags=["notifications"])
def mark_all_notifications_read(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    count = notifications_service.mark_all_read(db, current_user.id)
    return {"ok": True, "updated": count}


@app.put("/api/v1/settings", response_model=SettingsOut, tags=["settings"],
         dependencies=[Depends(get_current_user)])
def update_settings(body: SettingsUpdate, db: Session = Depends(get_db)):
    if body.notify_email_enabled is not None:
        _set_setting(db, "notify_email_enabled", str(body.notify_email_enabled).lower())
    if body.notify_email_to is not None:
        _set_setting(db, "notify_email_to", body.notify_email_to)
    if body.notify_on_failure is not None:
        _set_setting(db, "notify_on_failure", str(body.notify_on_failure).lower())
    if body.notify_on_success is not None:
        _set_setting(db, "notify_on_success", str(body.notify_on_success).lower())
    if body.notify_daily_summary is not None:
        _set_setting(db, "notify_daily_summary", str(body.notify_daily_summary).lower())
    return get_settings(db)
# Appended to server/app/main.py — RESTORE endpoints
# Placed after the notifications block, before the settings PUT.


# ── Restore Requests ──────────────────────────────────────────────────────────

@app.post("/api/v1/nodes/{node_id}/restore", response_model=RestoreRequestOut, tags=["restore"],
          dependencies=[Depends(get_current_user)])
def create_restore_request(
    node_id: str,
    body: RestoreRequestCreate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """Queue a restore request for a node. Agent picks it up via /restore/pending."""
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")

    # Validate the source job belongs to this node.
    src = db.query(JobRun).filter(JobRun.job_id == body.source_job_id).first()
    if not src:
        raise HTTPException(status_code=404, detail="Job de origen no encontrado")
    if src.node_id and src.node_id != node_id:
        raise HTTPException(status_code=400, detail="El job de origen pertenece a otro nodo")
    if src.status != "completed":
        raise HTTPException(status_code=400, detail="Sólo se pueden restaurar jobs completados")

    rr = RestoreRequest(
        node_id=node_id,
        source_job_id=body.source_job_id,
        target_path=body.target_path,
        filter_pattern=body.filter_pattern,
        dry_run=body.dry_run,
        status="queued",
        requested_by=current_user.username,
    )
    db.add(rr)
    db.commit()
    db.refresh(rr)
    return rr


@app.get("/api/v1/restore", response_model=list[RestoreRequestOut], tags=["restore"],
         dependencies=[Depends(get_current_user)])
def list_restore_requests(
    node_id: Optional[str] = Query(None),
    status: Optional[str] = Query(None),
    limit: int = Query(100, le=500),
    db: Session = Depends(get_db),
):
    q = db.query(RestoreRequest)
    if node_id:
        q = q.filter(RestoreRequest.node_id == node_id)
    if status:
        q = q.filter(RestoreRequest.status == status)
    return q.order_by(RestoreRequest.created_at.desc()).limit(limit).all()


@app.get("/api/v1/restore/{restore_id}", response_model=RestoreRequestOut, tags=["restore"],
         dependencies=[Depends(get_current_user)])
def get_restore_request(restore_id: int, db: Session = Depends(get_db)):
    rr = db.query(RestoreRequest).filter(RestoreRequest.id == restore_id).first()
    if not rr:
        raise HTTPException(status_code=404, detail="Restore request not found")
    return rr


@app.post("/api/v1/restore/{restore_id}/cancel", response_model=RestoreRequestOut, tags=["restore"],
          dependencies=[Depends(get_current_user)])
def cancel_restore_request(restore_id: int, db: Session = Depends(get_db)):
    rr = db.query(RestoreRequest).filter(RestoreRequest.id == restore_id).first()
    if not rr:
        raise HTTPException(status_code=404, detail="Restore request not found")
    if rr.status not in ("queued",):
        raise HTTPException(status_code=400, detail="Sólo se pueden cancelar restauraciones en cola")
    rr.status = "cancelled"
    rr.finished_at = datetime.now(timezone.utc)
    db.commit()
    db.refresh(rr)
    return rr


@app.get("/api/v1/nodes/{node_id}/restore/pending", tags=["restore"],
         dependencies=[Depends(require_agent_token)])
def pull_pending_restore(node_id: str, db: Session = Depends(get_db)):
    """
    Agent endpoint. Returns the oldest queued RestoreRequest for this node and
    atomically flips its status to "running". 204 No Content if no work.
    """
    from fastapi import Response

    rr = (db.query(RestoreRequest)
          .filter(RestoreRequest.node_id == node_id, RestoreRequest.status == "queued")
          .order_by(RestoreRequest.created_at.asc())
          .first())
    if not rr:
        return Response(status_code=204)

    rr.status = "running"
    rr.started_at = datetime.now(timezone.utc)
    db.commit()
    db.refresh(rr)

    return RestorePendingOut(
        id=rr.id,
        source_job_id=rr.source_job_id,
        target_path=rr.target_path,
        filter_pattern=rr.filter_pattern,
        dry_run=rr.dry_run,
    )


@app.post("/api/v1/restore/{restore_id}/progress", response_model=RestoreRequestOut, tags=["restore"],
          dependencies=[Depends(require_agent_token)])
def report_restore_progress(
    restore_id: int,
    body: RestoreProgressUpdate,
    db: Session = Depends(get_db),
):
    """Agent reports status updates for a running restore."""
    rr = db.query(RestoreRequest).filter(RestoreRequest.id == restore_id).first()
    if not rr:
        raise HTTPException(status_code=404, detail="Restore request not found")
    if body.status not in ("running", "completed", "failed"):
        raise HTTPException(status_code=400, detail="Estado inválido")

    rr.status = body.status
    if body.message is not None:
        rr.message = body.message
    if body.files_restored is not None:
        rr.files_restored = body.files_restored
    if body.bytes_restored is not None:
        rr.bytes_restored = body.bytes_restored
    if body.status in ("completed", "failed") and not rr.finished_at:
        rr.finished_at = datetime.now(timezone.utc)

    db.commit()
    db.refresh(rr)

    # Create an in-app notification on terminal status so admins see it in the bell.
    if body.status in ("completed", "failed"):
        try:
            admin = db.query(User).filter(User.username == (rr.requested_by or "admin")).first()
            if admin:
                notifications_service.create_for_user(
                    db,
                    user_id=admin.id,
                    type="backup_completed" if body.status == "completed" else "backup_failed",
                    title=f"Restore {body.status}: {rr.source_job_id[:8]}…",
                    body=(rr.message or f"Target: {rr.target_path}"),
                    entity_type="restore_request",
                    entity_id=str(rr.id),
                )
        except Exception:
            pass

    return rr
