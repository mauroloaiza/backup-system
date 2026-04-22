"""
One-shot SQLite → Postgres data migration.

Runs on server startup. If:
  1. The configured DATABASE_URL points to Postgres
  2. The Postgres target is fresh (no users registered yet)
  3. A legacy SQLite file is reachable (default /app/backupsmc.db, or SQLITE_LEGACY_PATH env)

…then this copies every row of every application table into Postgres,
preserving primary keys. Sequences are bumped afterwards so new inserts
don't collide with migrated IDs.

Fully idempotent: re-running after the first successful migration is a no-op
(the users-count check short-circuits).
"""
from __future__ import annotations

import logging
import os
from pathlib import Path

from sqlalchemy import create_engine, inspect, text
from sqlalchemy.orm import sessionmaker
from sqlalchemy.types import Boolean as SABoolean

from app.database import DATABASE_URL, engine as target_engine, is_postgres

log = logging.getLogger("backupsmc.bootstrap")

# Order matters — parents before children (FKs).
MIGRATION_ORDER = [
    "users",
    "agent_tokens",
    "system_settings",
    "email_settings",
    "nodes",
    "node_configs",
    "job_runs",
    "notifications",
]

# Tables with an auto-increment integer PK whose sequence needs bumping.
SEQUENCES = {
    "users": "id",
    "agent_tokens": "id",
    "job_runs": "id",
    "notifications": "id",
    "email_settings": "id",
}


def _sqlite_path() -> Path | None:
    override = os.getenv("SQLITE_LEGACY_PATH")
    candidates = [override] if override else ["/app/backupsmc.db", "./backupsmc.db"]
    for c in candidates:
        if c and Path(c).exists():
            return Path(c)
    return None


def _postgres_is_fresh() -> bool:
    """True if the Postgres target has no users (→ nothing to lose)."""
    insp = inspect(target_engine)
    if "users" not in insp.get_table_names():
        return True  # schema not even created yet — treat as fresh
    with target_engine.connect() as conn:
        count = conn.execute(text("SELECT COUNT(*) FROM users")).scalar()
    return (count or 0) == 0


def _copy_table(src_conn, dst_conn, table: str) -> int:
    """Copy all rows from src to dst, column-by-column. Returns row count."""
    # Inspect columns that actually exist on BOTH sides (some migrations may
    # have added columns only on one engine).
    src_insp = inspect(src_conn)
    dst_insp = inspect(dst_conn)
    if table not in src_insp.get_table_names():
        return 0
    if table not in dst_insp.get_table_names():
        log.warning("bootstrap: table %s missing in Postgres, skipping", table)
        return 0

    src_cols = {c["name"] for c in src_insp.get_columns(table)}
    dst_cols_info = {c["name"]: c for c in dst_insp.get_columns(table)}
    cols = sorted(src_cols & dst_cols_info.keys())
    if not cols:
        return 0

    # Which destination columns are booleans? SQLite stores them as 0/1 ints;
    # Postgres wants real bools and will reject integer values.
    bool_cols = {
        name for name, info in dst_cols_info.items()
        if isinstance(info.get("type"), SABoolean)
    }

    col_list = ", ".join(f'"{c}"' for c in cols)
    placeholders = ", ".join(f":{c}" for c in cols)

    rows = src_conn.execute(text(f'SELECT {col_list} FROM "{table}"')).mappings().all()
    if not rows:
        return 0

    coerced: list[dict] = []
    for r in rows:
        d = dict(r)
        for bc in bool_cols:
            if bc in d and d[bc] is not None:
                d[bc] = bool(d[bc])
        coerced.append(d)

    dst_conn.execute(
        text(f'INSERT INTO "{table}" ({col_list}) VALUES ({placeholders})'),
        coerced,
    )
    return len(rows)


def _bump_sequences(dst_conn) -> None:
    """After COPY, make each sequence start above the max existing id."""
    for table, pk in SEQUENCES.items():
        # Only Postgres has sequences. SQLite uses ROWID.
        try:
            dst_conn.execute(text(
                f"SELECT setval(pg_get_serial_sequence('{table}', '{pk}'), "
                f"COALESCE((SELECT MAX({pk}) FROM {table}), 0) + 1, false)"
            ))
        except Exception as e:
            log.debug("bootstrap: sequence bump for %s skipped: %s", table, e)


def run_if_needed() -> None:
    """Entry point called from main.py on startup."""
    if not is_postgres():
        return  # SQLite target — nothing to do
    if not _postgres_is_fresh():
        log.info("bootstrap: Postgres already populated, skipping legacy import")
        return

    legacy_path = _sqlite_path()
    if not legacy_path:
        log.info("bootstrap: no legacy SQLite found, skipping import")
        return

    log.warning("bootstrap: migrating legacy SQLite → Postgres (%s)", legacy_path)
    src_engine = create_engine(f"sqlite:///{legacy_path}",
                               connect_args={"check_same_thread": False})
    src_sess = sessionmaker(bind=src_engine)

    total = 0
    with src_sess() as src, target_engine.begin() as dst:
        for table in MIGRATION_ORDER:
            try:
                copied = _copy_table(src.connection(), dst, table)
                if copied:
                    log.info("bootstrap: %s: %d rows", table, copied)
                total += copied
            except Exception as e:
                log.error("bootstrap: failed copying %s: %s", table, e)
                raise
        _bump_sequences(dst)

    log.warning("bootstrap: migration complete — %d rows imported", total)
