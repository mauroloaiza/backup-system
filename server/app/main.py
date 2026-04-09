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
from app.models import Node, JobRun, User
from app.schemas import (
    NodeRegister, NodeOut, NodeSourcePathsUpdate,
    ProgressUpdate,
    JobRunOut,
    DashboardStats,
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
