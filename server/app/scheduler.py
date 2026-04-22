"""
APScheduler bootstrap — fires the daily summary email.

Runs as a BackgroundScheduler inside the FastAPI process. The daily cron is
triggered at `DAILY_SUMMARY_HOUR` (env, default "07") every day and simply
calls `daily_summary.send_daily_summary`, which internally checks the
`notify_daily_summary` flag and the SMTP configuration — so this scheduler is
safe to start unconditionally.
"""
from __future__ import annotations
import logging
import os
from typing import Optional

from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger

from app.database import get_db
from app import daily_summary

log = logging.getLogger("backupsmc.scheduler")

_scheduler: Optional[BackgroundScheduler] = None


def _daily_summary_job() -> None:
    db = next(get_db())
    try:
        result = daily_summary.send_daily_summary(db)
        log.info("daily_summary job result: %s", result)
    except Exception as e:
        log.exception("daily_summary job crashed: %s", e)
    finally:
        db.close()


def start() -> BackgroundScheduler:
    global _scheduler
    if _scheduler is not None:
        return _scheduler

    hour = int(os.getenv("DAILY_SUMMARY_HOUR", "7"))
    minute = int(os.getenv("DAILY_SUMMARY_MINUTE", "0"))
    tz = os.getenv("SCHEDULER_TZ", "UTC")

    sch = BackgroundScheduler(timezone=tz)
    sch.add_job(
        _daily_summary_job,
        trigger=CronTrigger(hour=hour, minute=minute),
        id="daily_summary",
        replace_existing=True,
        max_instances=1,
        coalesce=True,
    )
    sch.start()
    log.info("scheduler started: daily_summary at %02d:%02d %s", hour, minute, tz)
    _scheduler = sch
    return sch


def shutdown() -> None:
    global _scheduler
    if _scheduler is not None:
        _scheduler.shutdown(wait=False)
        _scheduler = None
