"""
SQLAlchemy ORM models.
"""
from datetime import datetime, timezone
from sqlalchemy import String, Integer, BigInteger, Boolean, DateTime, Text, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship
from app.database import Base
import json


def utcnow():
    return datetime.now(timezone.utc)


class Node(Base):
    """A registered backup agent node."""
    __tablename__ = "nodes"

    id: Mapped[str] = mapped_column(String(64), primary_key=True)  # hostname or uuid
    name: Mapped[str] = mapped_column(String(255))
    hostname: Mapped[str] = mapped_column(String(255))
    os: Mapped[str] = mapped_column(String(64), default="windows")
    agent_version: Mapped[str] = mapped_column(String(32), default="0.1.0")
    status: Mapped[str] = mapped_column(String(16), default="online")  # online|offline|error
    api_token: Mapped[str] = mapped_column(String(128), nullable=True)
    # JSON-encoded list of source paths reported by the agent
    source_paths_json: Mapped[str] = mapped_column(Text, nullable=True)
    # JSON-encoded list of destinations reported by the agent
    # [{type:"local"|"s3"|"sftp", target:"<human>", details:{...}}]
    destinations_json: Mapped[str] = mapped_column(Text, nullable=True)
    last_seen: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)

    runs: Mapped[list["JobRun"]] = relationship("JobRun", back_populates="node")

    @property
    def source_paths(self) -> list[str]:
        if not self.source_paths_json:
            return []
        try:
            return json.loads(self.source_paths_json)
        except Exception:
            return []

    @source_paths.setter
    def source_paths(self, paths: list[str]):
        self.source_paths_json = json.dumps(paths)

    @property
    def destinations(self) -> list[dict]:
        if not self.destinations_json:
            return []
        try:
            data = json.loads(self.destinations_json)
            return data if isinstance(data, list) else []
        except Exception:
            return []

    @destinations.setter
    def destinations(self, items: list[dict]):
        self.destinations_json = json.dumps(items or [])


class NodeConfig(Base):
    """
    Remote editable configuration for an agent. One row per node.

    Only the *safe* subset of the agent config is mirrored here — no
    encryption passphrases, no SFTP passwords, no API tokens. Secrets
    stay in the local agent.yaml and are never sent over the wire.

    The agent polls /api/v1/nodes/{id}/config/pull; if `version` is
    higher than what it has cached, it applies the JSON and writes
    back to its local agent.yaml.
    """
    __tablename__ = "node_configs"

    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.id", ondelete="CASCADE"),
                                          primary_key=True)
    # Monotonically increasing. Incremented on every PUT so agents detect changes.
    version: Mapped[int] = mapped_column(Integer, default=1, nullable=False)
    # JSON blob — validated by Pydantic NodeConfigPayload on write.
    payload_json: Mapped[str] = mapped_column(Text, nullable=False, default="{}")
    # Username of the last editor (auditable trail).
    updated_by: Mapped[str] = mapped_column(String(64), nullable=True)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)
    # Last time the agent pulled this config (for UI "applied yet?" indicator).
    last_pulled_version: Mapped[int] = mapped_column(Integer, default=0, nullable=False)
    last_pulled_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=True)

    @property
    def payload(self) -> dict:
        if not self.payload_json:
            return {}
        try:
            data = json.loads(self.payload_json)
            return data if isinstance(data, dict) else {}
        except Exception:
            return {}

    @payload.setter
    def payload(self, data: dict):
        self.payload_json = json.dumps(data or {})


class AgentToken(Base):
    """API tokens that agents use to authenticate (X-Agent-Token)."""
    __tablename__ = "agent_tokens"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    name: Mapped[str] = mapped_column(String(128))           # human label
    token: Mapped[str] = mapped_column(String(128), unique=True, index=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    last_used_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)


class SystemSetting(Base):
    """Key/value store for server-side settings."""
    __tablename__ = "system_settings"

    key: Mapped[str] = mapped_column(String(64), primary_key=True)
    value: Mapped[str] = mapped_column(Text, nullable=True)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)


class User(Base):
    """Web dashboard user."""
    __tablename__ = "users"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    hashed_password: Mapped[str] = mapped_column(String(255))
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)


class JobRun(Base):
    """A single backup or restore execution."""
    __tablename__ = "job_runs"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    job_id: Mapped[str] = mapped_column(String(64), index=True)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.id"), nullable=True)
    status: Mapped[str] = mapped_column(String(16), default="running")  # running|completed|failed|warning
    backup_type: Mapped[str] = mapped_column(String(16), default="full")  # full|incremental

    files_total: Mapped[int] = mapped_column(BigInteger, default=0)
    files_done: Mapped[int] = mapped_column(BigInteger, default=0)
    bytes_total: Mapped[int] = mapped_column(BigInteger, default=0)
    bytes_done: Mapped[int] = mapped_column(BigInteger, default=0)
    current_file: Mapped[str] = mapped_column(Text, nullable=True)
    error_message: Mapped[str] = mapped_column(Text, nullable=True)
    manifest_path: Mapped[str] = mapped_column(Text, nullable=True)

    started_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)
    finished_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=True)

    node: Mapped["Node"] = relationship("Node", back_populates="runs")

    @property
    def duration_seconds(self) -> float | None:
        if self.finished_at and self.started_at:
            return (self.finished_at - self.started_at).total_seconds()
        return None

    @property
    def progress_pct(self) -> float:
        if self.files_total and self.files_total > 0:
            return round(self.files_done / self.files_total * 100, 1)
        return 0.0


class EmailSettings(Base):
    """Single-row table with SMTP outbound configuration."""
    __tablename__ = "email_settings"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    # "smtp" | "office365"
    outbound_provider: Mapped[str] = mapped_column(String(32), default="smtp")
    from_name: Mapped[str] = mapped_column(String(128), default="BackupSMC")
    from_email: Mapped[str] = mapped_column(String(255), nullable=True)
    smtp_host: Mapped[str] = mapped_column(String(255), nullable=True)
    smtp_port: Mapped[int] = mapped_column(Integer, default=587)
    smtp_secure: Mapped[bool] = mapped_column(Boolean, default=False)  # true=SSL, false=STARTTLS
    smtp_user: Mapped[str] = mapped_column(String(255), nullable=True)
    smtp_pass: Mapped[str] = mapped_column(String(255), nullable=True)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)


class Notification(Base):
    """In-app notifications shown in the top-bar bell."""
    __tablename__ = "notifications"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    user_id: Mapped[int] = mapped_column(Integer, ForeignKey("users.id"), index=True)
    # Semantic type used by the UI to pick icon/color:
    # backup_completed | backup_failed | backup_warning | node_offline | general
    type: Mapped[str] = mapped_column(String(32), default="general")
    title: Mapped[str] = mapped_column(String(255))
    body: Mapped[str] = mapped_column(Text, nullable=True)
    # Optional link payload so the UI can navigate (e.g. entity_type="job_run", entity_id="42")
    entity_type: Mapped[str] = mapped_column(String(32), nullable=True)
    entity_id: Mapped[str] = mapped_column(String(64), nullable=True)
    read: Mapped[bool] = mapped_column(Boolean, default=False, index=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, index=True)
