"""
Database setup — PostgreSQL in production, SQLite fallback via DATABASE_URL.
"""
import os
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, DeclarativeBase

DATABASE_URL = os.getenv("DATABASE_URL", "sqlite:///./backupsmc.db")

# SQLite needs check_same_thread=False for FastAPI (single-writer assumption).
# Postgres doesn't need this and errors on unknown args.
connect_args = {"check_same_thread": False} if DATABASE_URL.startswith("sqlite") else {}

# Pool sizing for Postgres — SQLite ignores these.
_engine_kwargs: dict = {"connect_args": connect_args}
if DATABASE_URL.startswith("postgresql"):
    _engine_kwargs.update({
        "pool_size": 10,
        "max_overflow": 20,
        "pool_pre_ping": True,   # auto-reconnect after idle/broken connections
        "pool_recycle": 3600,    # recycle connections hourly
    })

engine = create_engine(DATABASE_URL, **_engine_kwargs)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


class Base(DeclarativeBase):
    pass


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def is_postgres() -> bool:
    return DATABASE_URL.startswith("postgresql")
