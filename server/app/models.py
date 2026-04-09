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
