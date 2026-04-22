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

class DestinationEntry(BaseModel):
    """One destination configured on a node (reported by the agent)."""
    type: str                     # "local" | "s3" | "sftp"
    target: str                   # human-readable (path, bucket, host:/path)
    details: Optional[dict] = None  # provider-specific extras (region, port, user, prefix)

class NodeRegister(BaseModel):
    id: str = Field(..., description="Unique node ID (hostname or UUID)")
    name: str
    hostname: str
    os: str = "windows"
    agent_version: str = "0.1.0"
    source_paths: list[str] = []
    destinations: Optional[list[DestinationEntry]] = None

class NodeOut(BaseModel):
    id: str
    name: str
    hostname: str
    os: str
    agent_version: str
    status: str
    source_paths: list[str] = []
    destinations: list[DestinationEntry] = []
    last_seen: datetime
    created_at: datetime

    class Config:
        from_attributes = True

class NodeSourcePathsUpdate(BaseModel):
    source_paths: list[str]


# ── Remote agent config editor ────────────────────────────────────────────────

class EmailNotifyPayload(BaseModel):
    """Agent-side SMTP notifications (separate from server-side alerting)."""
    enabled: bool = False
    on_failure: bool = True
    on_success: bool = False
    to: list[str] = []

class NodeConfigPayload(BaseModel):
    """
    The *editable* subset of the agent config that we mirror server-side.

    Intentionally excluded (stay in agent.yaml, never over the wire):
      - backup.encryption_passphrase
      - destination.sftp_password / sftp_key_file
      - server.api_token
      - notify.email.username / password (SMTP creds)
    """
    # Backup behavior
    source_paths: list[str] = []
    exclude_patterns: list[str] = []
    schedule_interval_minutes: int = Field(1440, ge=5, le=10080)  # 5 min → 7 days
    use_vss: bool = True
    incremental: bool = True
    verify_after_backup: bool = False
    throttle_mbps: float = Field(0, ge=0, le=10000)
    pre_script: Optional[str] = None
    post_script: Optional[str] = None

    # Retention
    retention_days: int = Field(30, ge=0, le=3650)
    gfs_enabled: bool = False
    gfs_keep_daily: int = Field(7, ge=0, le=365)
    gfs_keep_weekly: int = Field(4, ge=0, le=520)
    gfs_keep_monthly: int = Field(12, ge=0, le=120)

    # Retry
    retry_max_attempts: int = Field(3, ge=1, le=20)
    retry_initial_delay_seconds: int = Field(1, ge=1, le=300)

    # Log
    log_level: str = Field("info", pattern="^(debug|info|warn|error)$")

    # Email notifications (toggles only — SMTP creds never leave the agent)
    email: EmailNotifyPayload = EmailNotifyPayload()

class NodeConfigOut(BaseModel):
    """What the UI receives."""
    node_id: str
    version: int
    payload: NodeConfigPayload
    updated_by: Optional[str] = None
    updated_at: Optional[datetime] = None
    last_pulled_version: int = 0
    last_pulled_at: Optional[datetime] = None
    # Derived — true if the agent has pulled the latest version.
    in_sync: bool = False

class NodeConfigUpdate(BaseModel):
    payload: NodeConfigPayload

class NodeConfigPull(BaseModel):
    """What the agent receives from /config/pull when a new version exists."""
    node_id: str
    version: int
    payload: NodeConfigPayload


class DestinationAggregate(BaseModel):
    """One row on the /destinations page: destination + owning node + usage stats."""
    node_id: str
    node_name: str
    hostname: str
    node_status: str              # "online" | "stale" | "offline"
    type: str                     # local | s3 | sftp
    target: str
    details: Optional[dict] = None
    bytes_backed_up: int          # sum across completed runs for this node
    runs_count: int               # completed runs count for this node
    last_backup_at: Optional[datetime] = None
    last_status: Optional[str] = None  # status of the last run


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


# ── Agent Tokens ──────────────────────────────────────────────────────────────

class AgentTokenOut(BaseModel):
    id: int
    name: str
    token: str
    is_active: bool
    last_used_at: Optional[datetime]
    created_at: datetime

    class Config:
        from_attributes = True

class AgentTokenCreate(BaseModel):
    name: str


# ── Settings ──────────────────────────────────────────────────────────────────

class SettingsOut(BaseModel):
    server_version: str
    agent_token: Optional[str]  # first active token (backward compat)
    notify_email_enabled: bool
    notify_email_to: str
    notify_on_failure: bool
    notify_on_success: bool
    notify_daily_summary: bool

class SettingsUpdate(BaseModel):
    notify_email_enabled: Optional[bool] = None
    notify_email_to: Optional[str] = None
    notify_on_failure: Optional[bool] = None
    notify_on_success: Optional[bool] = None
    notify_daily_summary: Optional[bool] = None


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


# ── Email / SMTP Settings ─────────────────────────────────────────────────────

class EmailSettingsOut(BaseModel):
    outbound_provider: str = "smtp"
    from_name: str = "BackupSMC"
    from_email: Optional[str] = None
    smtp_host: Optional[str] = None
    smtp_port: int = 587
    smtp_secure: bool = False
    smtp_user: Optional[str] = None
    smtp_pass: Optional[str] = None  # always masked on response
    configured: bool = False

class EmailSettingsUpdate(BaseModel):
    outbound_provider: Optional[str] = None
    from_name: Optional[str] = None
    from_email: Optional[str] = None
    smtp_host: Optional[str] = None
    smtp_port: Optional[int] = None
    smtp_secure: Optional[bool] = None
    smtp_user: Optional[str] = None
    smtp_pass: Optional[str] = None  # if "••••••••" or empty → keep existing

class EmailTestRequest(BaseModel):
    to: Optional[str] = None  # fallback: notify_email_to


# ── Notifications ─────────────────────────────────────────────────────────────

class NotificationOut(BaseModel):
    id: int
    type: str
    title: str
    body: Optional[str]
    entity_type: Optional[str]
    entity_id: Optional[str]
    read: bool
    created_at: datetime

    class Config:
        from_attributes = True

class NotificationList(BaseModel):
    notifications: list[NotificationOut]
    unread: int


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
