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
from sqlalchemy import func
import os

from app.database import engine, get_db, Base
from app.models import Node, JobRun
from app.schemas import (
    NodeRegister, NodeOut,
    ProgressUpdate,
    JobRunOut,
    DashboardStats,
)

# Create tables on startup
Base.metadata.create_all(bind=engine)

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


# ── Health ────────────────────────────────────────────────────────────────────

@app.get("/health", tags=["health"])
def health_check():
    return {"status": "ok", "service": "backupsmc-server", "version": "0.1.0"}


# ── Nodes ─────────────────────────────────────────────────────────────────────

@app.post("/api/v1/nodes/register", response_model=NodeOut, tags=["nodes"])
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
    else:
        node = Node(
            id=body.id,
            name=body.name,
            hostname=body.hostname,
            os=body.os,
            agent_version=body.agent_version,
            status="online",
        )
        db.add(node)
    db.commit()
    db.refresh(node)
    return node


@app.get("/api/v1/nodes", response_model=list[NodeOut], tags=["nodes"])
def list_nodes(db: Session = Depends(get_db)):
    return db.query(Node).order_by(Node.last_seen.desc()).all()


@app.get("/api/v1/nodes/{node_id}", response_model=NodeOut, tags=["nodes"])
def get_node(node_id: str, db: Session = Depends(get_db)):
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    return node


@app.delete("/api/v1/nodes/{node_id}", tags=["nodes"])
def delete_node(node_id: str, db: Session = Depends(get_db)):
    node = db.query(Node).filter(Node.id == node_id).first()
    if not node:
        raise HTTPException(status_code=404, detail="Node not found")
    db.delete(node)
    db.commit()
    return {"ok": True}


# ── Jobs / Progress ───────────────────────────────────────────────────────────

@app.post("/api/v1/jobs/{job_id}/progress", tags=["jobs"])
def update_progress(job_id: str, body: ProgressUpdate, db: Session = Depends(get_db)):
    """Agent reports backup progress. Creates or updates the JobRun record."""
    # Auto-register node if unknown
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
        if run.node_id:
            node = db.query(Node).filter(Node.id == run.node_id).first()
            if node:
                node.status = "online"

    db.commit()
    return {"ok": True, "job_id": job_id}


@app.get("/api/v1/jobs", response_model=list[JobRunOut], tags=["jobs"])
def list_jobs(
    status: Optional[str] = Query(None, description="Filter by status"),
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


@app.get("/api/v1/jobs/{job_id}", response_model=JobRunOut, tags=["jobs"])
def get_job(job_id: str, db: Session = Depends(get_db)):
    run = db.query(JobRun).filter(JobRun.job_id == job_id).first()
    if not run:
        raise HTTPException(status_code=404, detail="Job not found")
    return run


# ── Dashboard ─────────────────────────────────────────────────────────────────

@app.get("/api/v1/dashboard", response_model=DashboardStats, tags=["dashboard"])
def dashboard(db: Session = Depends(get_db)):
    today = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)

    total_nodes = db.query(func.count(Node.id)).scalar() or 0

    # Nodes not seen in last 5 min are offline
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
