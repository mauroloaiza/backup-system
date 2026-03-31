# Guía de desarrollo — BackupSMC

## Requisitos previos

| Herramienta | Versión mínima | Instalación |
|-------------|---------------|-------------|
| Docker | 24+ | https://docs.docker.com/get-docker/ |
| Docker Compose | v2 | Incluido en Docker Desktop |
| Node.js | 20 LTS | https://nodejs.org |
| Python | 3.12+ | https://python.org |
| Go | 1.22+ | https://go.dev/dl |
| Git | 2.40+ | https://git-scm.com |

---

## Configuración inicial

```bash
# 1. Clonar el repositorio
git clone https://github.com/smcsoluciones/backup-system.git
cd backup-system

# 2. Copiar variables de entorno
cp .env.example .env
# Editar .env con tus valores locales

# 3. Levantar servicios de desarrollo
docker compose -f docker-compose.dev.yml up -d

# 4. Verificar que todo corre
docker compose -f docker-compose.dev.yml ps
```

### Accesos locales

| Servicio | URL |
|----------|-----|
| API (Swagger UI) | http://localhost:8000/docs |
| API (ReDoc) | http://localhost:8000/redoc |
| Frontend | http://localhost:5173 |
| Flower (Celery monitor) | http://localhost:5555 |
| PostgreSQL | localhost:5433 |
| Redis | localhost:6380 |

---

## Desarrollo del servidor (FastAPI)

### Sin Docker (recomendado para iterar rápido)

```bash
cd server

# Crear entorno virtual
python -m venv .venv
source .venv/bin/activate       # Linux/Mac
.venv\Scripts\activate          # Windows

# Instalar dependencias
pip install -r requirements.txt
pip install -r requirements-dev.txt

# Ejecutar servidor con hot-reload
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

### Migraciones de base de datos

```bash
cd server

# Crear una nueva migración
alembic revision --autogenerate -m "descripción del cambio"

# Aplicar migraciones
alembic upgrade head

# Revertir última migración
alembic downgrade -1
```

### Tests

```bash
cd server

# Ejecutar todos los tests
pytest

# Con cobertura
pytest --cov=app --cov-report=html

# Solo tests de un módulo
pytest tests/api/test_jobs.py -v
```

### Linting y formato

```bash
cd server

# Formatear código
black .

# Verificar linting
ruff check .

# Type checking
mypy app/
```

---

## Desarrollo del agente (Go)

### Ejecutar en local

```bash
cd agent

# Descargar dependencias
go mod download

# Ejecutar en modo desarrollo
go run cmd/agent/main.go --server http://localhost:8000 --token dev-token

# Compilar binario
go build -o bin/backupsmc-agent cmd/agent/main.go

# Tests
go test ./...

# Linting
golangci-lint run
```

### Cross-compilación

```bash
cd agent

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o bin/backupsmc-agent-linux-amd64 cmd/agent/main.go

# Linux arm64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o bin/backupsmc-agent-linux-arm64 cmd/agent/main.go

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o bin/backupsmc-agent-windows-amd64.exe cmd/agent/main.go
```

---

## Desarrollo del frontend (React)

### Sin Docker

```bash
cd frontend

# Instalar dependencias
npm install

# Iniciar servidor de desarrollo (Vite)
npm run dev

# Build de producción
npm run build

# Preview del build
npm run preview
```

### Linting y formato

```bash
cd frontend

# ESLint
npm run lint

# Prettier
npm run format

# Type checking
npm run type-check
```

### Tests

```bash
cd frontend

# Unit tests (Vitest)
npm run test

# Tests con UI
npm run test:ui

# Cobertura
npm run test:coverage
```

---

## Convenciones del proyecto

### Git Flow

```
main        → producción (solo via PR desde release/ o hotfix/)
develop     → integración (target de todos los PRs)
feature/*   → nuevas funcionalidades (desde develop)
fix/*       → correcciones (desde develop)
hotfix/*    → correcciones urgentes (desde main)
release/*   → preparación de versiones (desde develop)
```

### Commits (Conventional Commits)

```bash
feat(server): add PostgreSQL backup support
fix(agent): handle connection timeout gracefully
docs: update development guide
chore(deps): bump fastapi to 0.115.0
test(api): add integration tests for backup jobs
```

### Nombrado de ramas

```bash
feature/s3-destination-support
feature/docker-volume-backup
fix/agent-registration-timeout
hotfix/critical-auth-bypass
release/v0.2.0
```

---

## Variables de entorno para desarrollo

Las más importantes para desarrollo local en `.env`:

```env
APP_ENV=development
DEBUG=true
DATABASE_URL=postgresql+asyncpg://backupsmc:secret@localhost:5433/backupsmc
REDIS_URL=redis://localhost:6380/0
SECRET_KEY=dev-secret-key-not-for-production
```

---

## Troubleshooting

### Docker no levanta

```bash
# Ver logs de un servicio específico
docker compose -f docker-compose.dev.yml logs server -f

# Reconstruir imagen
docker compose -f docker-compose.dev.yml build server --no-cache

# Reiniciar todo
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d
```

### Puerto ocupado

```bash
# Ver qué usa el puerto 8000
lsof -i :8000       # Linux/Mac
netstat -ano | findstr :8000   # Windows
```

### Base de datos no disponible

```bash
# Verificar estado de PostgreSQL
docker compose -f docker-compose.dev.yml exec db pg_isready

# Conectarse directamente
docker compose -f docker-compose.dev.yml exec db psql -U backupsmc -d backupsmc
```
