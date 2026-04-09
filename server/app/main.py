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

from app.database import engine, get_db, Base
from app.models import Node, JobRun, User, AgentToken, SystemSetting
from app.schemas import (
    NodeRegister, NodeOut, NodeSourcePathsUpdate,
    ProgressUpdate,
    JobRunOut,
    DashboardStats,
    HistoryStats, DailyStatPoint,
    AgentTokenOut, AgentTokenCreate,
    SettingsOut, SettingsUpdate,
    LoginRequest, Token, UserOut,
)
from app.auth import (
    verify_password, hash_password, create_access_token,
    get_current_user, require_agent_token,
)

# ── DB init ───────────────────────────────────────────────────────────────────

Base.metadata.create_all(bind=engine)

# Add missing columns for existing DBs (idempotent)
def _run_migrations():
    with engine.connect() as conn:
        for col_def in [
            "ALTER TABLE nodes ADD COLUMN source_paths_json TEXT",
        ]:
            try:
                conn.execute(text(col_def))
                conn.commit()
            except Exception:
                pass


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
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

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
    return {"status": "ok", "service": "backupsmc-server", "version": "0.1.0"}


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
        server_version="0.5.0",
        agent_token=first_token.token if first_token else None,
        notify_email_enabled=_get_setting(db, "notify_email_enabled", "false") == "true",
        notify_email_to=_get_setting(db, "notify_email_to", ""),
        notify_on_failure=_get_setting(db, "notify_on_failure", "true") == "true",
        notify_on_success=_get_setting(db, "notify_on_success", "false") == "true",
        notify_daily_summary=_get_setting(db, "notify_daily_summary", "false") == "true",
    )


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
