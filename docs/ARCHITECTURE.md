# Arquitectura técnica — SMC Backup

> Versión: v0.4.1 — 2026-04-02

---

## Visión general

SMC Backup sigue una arquitectura de **tres capas desacopladas**: el **Servidor** (control plane), el **Agente** (data plane en cada nodo Windows) y el **Frontend** (UI web). La comunicación entre capas usa REST sobre HTTP con dos canales de autenticación separados.

---

## Diagrama de alto nivel

```
┌──────────────────────────────────────────────────────────────┐
│                     BROWSER / OPERADOR                       │
└─────────────────────────┬────────────────────────────────────┘
                          │ HTTPS · JWT Bearer
┌─────────────────────────▼────────────────────────────────────┐
│                  FRONTEND (React + Vite)                     │
│  Login · Dashboard Nodos · source_paths · Historial (WIP)    │
└─────────────────────────┬────────────────────────────────────┘
                          │ REST API /api/v1
┌─────────────────────────▼────────────────────────────────────┐
│                  SERVER (FastAPI / Python)                    │
│  ┌───────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │  Auth (JWT)   │  │  Nodes API   │  │   Jobs API        │  │
│  │  /auth/login  │  │  /nodes/*    │  │   /jobs/*/progress│  │
│  └───────────────┘  └──────────────┘  └───────────────────┘  │
│                    SQLite (dev) / PostgreSQL (prod)           │
└──────────────────┬──────────────┬───────────────────────────┘
                   │ X-Agent-Token│
       ┌───────────▼──┐   ┌───────▼───────────┐
       │  AGENTE Go   │   │   AGENTE Go        │
       │  Nodo A      │   │   Nodo B           │
       │  Windows Srv │   │   Windows Srv      │
       └──────────────┘   └────────────────────┘
              │
    ┌─────────▼────────────────────┐
    │       DESTINO DE BACKUP      │
    │  Local · S3 · SFTP           │
    └──────────────────────────────┘
```

---

## Componentes

### 1. Server — `server/`

| Item | Detalle |
|------|---------|
| Framework | FastAPI |
| Lenguaje | Python 3.11+ |
| Base de datos | SQLite (dev) · PostgreSQL (prod) |
| ORM | SQLAlchemy |
| Auth web | JWT (python-jose) + sha256_crypt (passlib) |
| Auth agente | Header `X-Agent-Token` |
| Configuración | Variables de entorno vía `.env` |
| Punto de entrada | `uvicorn app.main:app --port 8000` |

**Endpoints principales:**

| Método | Ruta | Auth | Descripción |
|--------|------|------|-------------|
| POST | `/api/v1/auth/login` | Pública | Login usuario → JWT |
| POST | `/api/v1/nodes/register` | Agent Token | Registro/heartbeat del agente |
| PUT | `/api/v1/nodes/{id}/source-paths` | JWT | Actualizar rutas de origen |
| POST | `/api/v1/jobs/{id}/progress` | Agent Token | Reportar progreso del job |

---

### 2. Agente — `agent/`

| Item | Detalle |
|------|---------|
| Lenguaje | Go 1.22+ |
| Binario | `backupsmc-agent.exe` (20.6 MB, sin dependencias externas) |
| Instalador | Inno Setup 6 (`BackupSMC-Agent-{version}-Setup.exe`) |
| Configuración | `agent.yaml` (Viper) |
| Modo de ejecución | Servicio Windows (SCM) o proceso foreground |

**Pipeline de backup por archivo:**

```
Archivo fuente
    │
    ▼ [Throttle reader — token bucket]
    │
    ├─── TeeReader → SHA-256 hash (plaintext)
    │
    ▼ [Compresión zstd — si la extensión no está en skip list]
    │
    ▼ [Cifrado AES-256-GCM — streaming]
    │
    ▼ Destino (Local / S3 / SFTP)
    │
    └─── [VerifyAfterBackup] → leer de vuelta, descifrar, descomprimir, recomputar hash
```

**Packages internos:**

| Package | Responsabilidad |
|---------|----------------|
| `backup/engine` | Orchestración del job (VSS, scan, backup, manifiesto) |
| `backup/scanner` | Detección de cambios (mtime + size, caché persistente) |
| `backup/retention` | GFS + simple days — limpieza post-job |
| `backup/manifest` | Manifiesto cifrado (JSON + zstd + AES-256-GCM) |
| `backup/restore` | Restauración por job ID con filtro glob |
| `backup/vss` | VSS snapshots (Windows only, stub en Linux) |
| `backup/acl` | SDDL ACL preservación y restauración |
| `destination/local` | Escritura en disco local |
| `destination/s3` | AWS SDK v2 |
| `destination/sftp` | `pkg/sftp` + `golang.org/x/crypto/ssh` |
| `destination/factory` | Selecciona destino por `cfg.Destination.Type` |
| `retry` | Backoff exponencial con jitter y errores permanentes |
| `throttle` | Token bucket reader/writer (stdlib puro) |
| `notify` | Email SMTP + Windows Event Log |
| `noderegister` | Registro + heartbeat al servidor |
| `reporter` | Progreso en tiempo real vía HTTP |
| `config` | Viper YAML + defaults + validación |
| `crypto` | AES-256-GCM streaming (encrypt/decrypt) |
| `compress` | zstd streaming (encode/decode) |
| `winsvc` | Integración con Windows SCM |

---

### 3. Frontend — `frontend/`

| Item | Detalle |
|------|---------|
| Framework | React 18 + Vite |
| Estilos | Tailwind CSS + shadcn/ui |
| Estado | Zustand (auth persistente en localStorage) |
| HTTP | Axios con interceptor 401 |
| Rutas | React Router v6 con `RequireAuth` wrapper |
| Puerto dev | `npm run dev` → 5173 (o siguiente disponible) |

---

### 4. GUI Desktop — `gui/`

| Item | Detalle |
|------|---------|
| Framework | Wails v2 |
| Backend | Go |
| Frontend | React (misma base que el web) |
| Binario | `BackupSMC.exe` |
| Nota | `CREATE_NO_WINDOW` en todos los `exec.Command` para evitar parpadeo de CMD |

---

## Autenticación — dos canales

```
Browser ──── POST /auth/login {username, password}
                    │
                    └─→ JWT Bearer token (header Authorization)
                         Expira en 24h · sha256_crypt

Agente ────── X-Agent-Token: <api_token>
                    │
                    └─→ Validado contra AGENT_TOKEN en .env
                         Sin expiración · rotación manual
```

---

## Modelo de datos (SQLite/PostgreSQL)

```sql
users
  id, username, hashed_password, is_active

nodes
  id, name, hostname, os, agent_version,
  status, last_seen, source_paths_json

jobs (pendiente de implementar en BD)
  id, node_id, status, started_at, finished_at,
  changed_files, changed_bytes, errors, manifest_path
```

---

## Configuración del agente (`agent.yaml`)

```yaml
server:
  url: "http://servidor:8000"
  api_token: "token-secreto"

backup:
  source_paths: ["C:\\Users", "D:\\Data"]
  encryption_passphrase: "passphrase-minimo-16-chars"
  use_vss: true
  incremental: true
  schedule_interval: 24h
  verify_after_backup: false
  throttle_mbps: 0
  pre_script: ""
  post_script: ""

destination:
  type: local          # local | s3 | sftp
  local_path: "D:\\Backups\\BackupSMC"

retention:
  days: 30
  gfs:
    enabled: false
    keep_daily: 7
    keep_weekly: 4
    keep_monthly: 12

retry:
  max_attempts: 3
  initial_delay: 1s

notify:
  email:
    enabled: false
    smtp_host: "smtp.empresa.com"
    smtp_port: 587
    on_failure: true
    on_success: false
```

---

## Flujo de un backup completo

```
1. Scheduler ticker dispara (o `backupsmc-agent run`)
2. Pre-script (si configurado) — aborta si exit != 0
3. VSS snapshot por volumen (Windows)
4. Para cada source_path:
   a. Carga caché incremental de %ProgramData%\BackupSMC\state\
   b. Scanner detecta archivos modificados (mtime + size)
   c. Para cada archivo modificado:
      - retry.Do() envuelve backupFile()
      - Throttle → TeeReader (SHA-256) → zstd → AES-256-GCM → destino
      - Si verify_after_backup: leer de vuelta y comparar hash
   d. Guarda nueva caché incremental
5. Escribe manifiesto cifrado (con retry)
6. Reporta resultado al servidor
7. Post-script (siempre, incluso en fallo)
8. runRetention() — aplica GFS o days
9. Notificación email + Windows Event Log
```

---

## Estructura de objetos en el destino

```
jobs/
  {job-id}/
    {timestamp}/
      manifest.bsmc          ← manifiesto cifrado
      data/
        {src-tag}/
          {rel-path}.bsmc    ← archivo cifrado
          {rel-path}.part0000.bsmc  ← chunk (archivos >512 MB)
```

---

## Dependencias clave (Go)

| Paquete | Uso |
|---------|-----|
| `github.com/spf13/cobra` | CLI |
| `github.com/spf13/viper` | Configuración YAML |
| `go.uber.org/zap` | Logging estructurado |
| `github.com/google/uuid` | Job IDs |
| `github.com/aws/aws-sdk-go-v2` | Destino S3 |
| `github.com/pkg/sftp` | Destino SFTP |
| `golang.org/x/crypto/ssh` | SSH para SFTP |
| `golang.org/x/sys/windows/svc` | Servicio Windows |
| `github.com/klauspost/compress/zstd` | Compresión |
| `wails.io/v2` | GUI desktop |

---

## Dependencias clave (Python)

| Paquete | Uso |
|---------|-----|
| `fastapi` | Framework REST |
| `uvicorn` | ASGI server |
| `sqlalchemy` | ORM |
| `python-jose[cryptography]` | JWT |
| `passlib[sha256_crypt]` | Hash de contraseñas |
| `python-multipart` | Form data |
| `python-dotenv` | Variables de entorno |
