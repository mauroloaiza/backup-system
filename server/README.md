# BackupSMC — Server (FastAPI)

Backend del sistema BackupSMC. Provee la API REST, scheduler de jobs y motor de notificaciones.

## Stack

- **Python 3.12** + **FastAPI 0.115+**
- **SQLAlchemy 2.x** (async) + **Alembic** (migraciones)
- **Celery 5** + **Redis** (jobs asincrónicos)
- **PostgreSQL 16** (base de datos)
- **Pydantic v2** (validación)

## Estructura

```
server/
├── app/
│   ├── api/v1/          # Routers FastAPI
│   ├── models/          # SQLAlchemy ORM models
│   ├── schemas/         # Pydantic schemas
│   ├── services/        # Lógica de negocio
│   ├── tasks/           # Celery tasks
│   ├── core/            # Config, seguridad, deps
│   └── main.py          # Entry point FastAPI
├── tests/               # Tests pytest
├── alembic/             # Migraciones DB
├── requirements.txt
├── requirements-dev.txt
├── Dockerfile
└── Dockerfile.dev
```

## Desarrollo

Ver [DEVELOPMENT.md](../docs/DEVELOPMENT.md)

## API Docs

Con el servidor corriendo: http://localhost:8000/docs
