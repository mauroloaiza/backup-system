"""
Pydantic schemas for request/response validation.
"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


# ── Auth ──────────────────────────────────────────────────────────────────────

class LoginRequest(BaseModel):
    username: str
    password: str

class Token(BaseModel):
    access_token: str
    token_type: str = "bearer"

class UserOut(BaseModel):
    id: int
    username: str
    is_active: bool

    class Config:
        from_attributes = True


# ── Node ──────────────────────────────────────────────────────────────────────

class NodeRegister(BaseModel):
    id: str = Field(..., description="Unique node ID (hostname or UUID)")
    name: str
    hostname: str
    os: str = "windows"
    agent_version: str = "0.1.0"
    source_paths: list[str] = []

class NodeOut(BaseModel):
    id: str
    name: str
    hostname: str
    os: str
    agent_version: str
    status: str
    source_paths: list[str] = []
    last_seen: datetime
    created_at: datetime

    class Config:
        from_attributes = True

class NodeSourcePathsUpdate(BaseModel):
    source_paths: list[str]


# ── Progress (from agent) ─────────────────────────────────────────────────────

class ProgressUpdate(BaseModel):
    job_id: str
    node_id: str
    status: str  # running | completed | failed | warning
    files_total: int = 0
    files_done: int = 0
    bytes_total: int = 0
    bytes_done: int = 0
    current_file: Optional[str] = None
    error_message: Optional[str] = None
    started_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


# ── Job Run ───────────────────────────────────────────────────────────────────

class JobRunOut(BaseModel):
    id: int
    job_id: str
    node_id: Optional[str]
    status: str
    backup_type: str
    files_total: int
    files_done: int
    bytes_total: int
    bytes_done: int
    current_file: Optional[str]
    error_message: Optional[str]
    manifest_path: Optional[str]
    progress_pct: float
    duration_seconds: Optional[float]
    started_at: datetime
    updated_at: datetime
    finished_at: Optional[datetime]

    class Config:
        from_attributes = True


# ── History / Stats ───────────────────────────────────────────────────────────

class DailyStatPoint(BaseModel):
    date: str           # "2026-04-09"
    total: int
    completed: int
    failed: int
    bytes: int
    duration_avg: float  # seconds

class HistoryStats(BaseModel):
    points: list[DailyStatPoint]
    success_rate: float  # 0-100
    avg_duration: float  # seconds
    total_bytes: int
    total_runs: int


# ── Dashboard ─────────────────────────────────────────────────────────────────

class DashboardStats(BaseModel):
    total_nodes: int
    nodes_online: int
    nodes_offline: int
    total_runs: int
    runs_today: int
    runs_completed: int
    runs_failed: int
    runs_running: int
    bytes_backed_up_today: int
    bytes_backed_up_total: int
    recent_runs: list[JobRunOut]
    running_now: list[JobRunOut]
