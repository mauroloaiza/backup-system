# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es/1.0.0/)
y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [0.10.0] - 2026-04-20

### Changed — Migración SQLite → PostgreSQL

El servidor ahora corre sobre **PostgreSQL 16** (Alpine) en lugar de SQLite. Primer paso del hardening operativo: múltiples conexiones concurrentes, sin lockeos de escritura, booleans tipados, y separación física de la base de datos respecto del container de aplicación.

**Infraestructura**:
- Nuevo servicio `postgres` en `docker-compose.ec2-dev.yml` (imagen `postgres:16-alpine`, volumen dedicado `postgres_data`, `healthcheck` con `pg_isready`). `server` ahora `depends_on: postgres: condition: service_healthy`.
- `DATABASE_URL=postgresql+psycopg2://backupsmc:<pw>@postgres:5432/backupsmc` en `.env` DEV.
- Credenciales de postgres (`POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`) añadidas al `.env` — password generada vía `openssl rand -hex 24`.
- Postgres **NO** expuesto al host — solo accesible desde la red interna del compose project.

**Código**:
- `requirements.txt`: añadido `psycopg2-binary==2.9.10`.
- `app/database.py`: pooling habilitado cuando el driver es Postgres (`pool_size=10`, `max_overflow=20`, `pool_pre_ping=True`, `pool_recycle=3600`). SQLite sigue soportado para desarrollo local.
- `app/main.py` → `_run_migrations()` ahora usa `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` en Postgres (idempotente real) y sigue tragándose errores en SQLite (semántica pre-existente).
- Helper `is_postgres()` expuesto desde `database.py`.

**Migración de datos**:
- Nuevo módulo `app/bootstrap_migrate.py` — corre en startup. Si el target Postgres está vacío y existe un `backupsmc.db` legacy (path via `SQLITE_LEGACY_PATH` env, default `/app/backupsmc.db`), copia todas las tablas **en orden de dependencias FK** (`users → agent_tokens → system_settings → email_settings → nodes → node_configs → job_runs → notifications`).
- Coerción de tipos: SQLite guarda `BOOLEAN` como `0/1` integer; el migrador detecta columnas `SABoolean` en el destino y las convierte con `bool()` antes de insertar (Postgres rechaza el integer literal).
- Bump de secuencias post-import: `setval(pg_get_serial_sequence(table, 'id'), MAX(id)+1, false)` para que los próximos `INSERT` con autoincrement no colisionen con IDs migrados.
- Fully idempotent — re-runs son no-op (el check de `users > 0` corta el flujo).

**Verificación end-to-end post-deploy**:
- ✅ 1 user + 1 agent token + 4 nodes + 74 job runs + 1 node config + 7 notifications migrados intactos
- ✅ Login admin funciona, JWT emitido
- ✅ `/api/v1/dashboard` devuelve `total_runs=74` (preservó todo el historial)
- ✅ `/api/v1/nodes` devuelve los 4 nodos con sus nombres originales (dc-fileserver-01, app-server-prod, finance-nas-02, legacy-win7-archive)
- ✅ Agent pull endpoint funciona igual contra Postgres

---

## [0.9.0] - 2026-04-20

### Added — Agent Config Editor remoto

Editar la configuración de un agente remoto desde la UI web, **sin SSH al host del agente**. Cambios propagados vía un endpoint de polling.

**Server**:
- Nuevo modelo `NodeConfig` (`node_configs`) con `version` monotónica, `payload_json`, `updated_by`, y contador `last_pulled_version` para mostrar "Aplicada / Pendiente".
- `GET  /api/v1/nodes/{id}/config` — UI lee la config editable.
- `PUT  /api/v1/nodes/{id}/config` — UI guarda cambios (incrementa `version`, auditable por usuario).
- `GET  /api/v1/nodes/{id}/config/pull?current_version=N` — endpoint de polling del agente. Devuelve `204 No Content` si está al día, o `{node_id, version, payload}` con la nueva config. También registra `last_pulled_version`/`last_pulled_at` para el indicador de UI.
- Nuevo schema `NodeConfigPayload` con **sólo campos editables y no-secretos**: `source_paths`, `exclude_patterns`, `schedule_interval_minutes`, `use_vss`, `incremental`, `verify_after_backup`, `throttle_mbps`, `pre_script`/`post_script`, `retention_days` + GFS, `retry_*`, `log_level`, toggles de email.
- **Explícitamente excluidos** (seguridad — nunca viajan por la red): `encryption_passphrase`, `sftp_password`, `sftp_key_file`, `smtp_user`/`smtp_pass`, `api_token`. Estos secretos permanecen en el `agent.yaml` local del agente.

**Agent** (v0.3.0):
- Nuevo paquete `internal/configsync` — goroutine que polling cada 60s al server. En versión nueva:
  1. Aplica cambios **en memoria** sobre el `*config.Config` bajo mutex (siguiente tick los ve).
  2. Reescribe el `agent.yaml` local **preservando secretos y claves que no maneja** (parse → patch → marshal atómico con rename).
  3. Bump del archivo sidecar `.backupsmc-config-version` para sobrevivir reinicios sin volver a aplicar snapshots viejos.
- Campos hot-aplicables (siguiente tick de backup): `source_paths`, `exclude_patterns`, VSS, incremental, verify, throttle, scripts, retención, retry, nivel de log, toggles de email.
- Fields que sólo toman efecto tras restart: el `schedule_interval` en curso (el ticker sigue con el valor viejo hasta el próximo tick) y cualquier cambio en `destination.*` (requiere re-abrir el backend).

**Frontend**:
- Nuevo componente `NodeConfigEditor` — drawer derecho de 560px con formularios para todos los campos editables, organizados en secciones (Planificación, Carpetas, Exclusiones, Scripts, Retención, Reintentos, Logging, Email).
- Cards de nodo (`/nodes`): nuevo ícono de engranaje junto al badge de estado para abrir el editor.
- **Badge "Aplicada vN / Pendiente vN"** en el header del drawer — muestra si el agente ya hizo pull de la versión actual (ámbar pulsante si está pendiente). React Query hace refetch cada 10s para que el indicador se actualice automáticamente ~60s después de guardar.
- Manejo inmutable del draft con `structuredClone`, botón "Cancelar" descarta cambios no guardados.

### Technical

- Migración: `node_configs` se crea vía `Base.metadata.create_all()` (FK a `nodes` con `ON DELETE CASCADE`).
- En el PUT, `node.source_paths` se sincroniza al valor nuevo para que la página `/nodes` refleje el cambio sin esperar al siguiente heartbeat.
- Writeback YAML atómico: `tmp file + rename` para evitar corrupción si el agente muere durante la escritura.

---

## [0.8.3] - 2026-04-19

### Added — Gráficas en el drawer de nodo

- **Panel "Volumen por run"** en `NodeRunsDrawer` (abrir un nodo en `/nodes`): BarChart recharts con las últimas 20 ejecuciones, barras ordenadas cronológicamente de izq→der. Color según estado: azul `#4361ee` = completed, rojo `#ef4444` = failed, ámbar `#f59e0b` = warning. Tooltip formatea bytes con `fmtBytes`. Detecta visualmente de un vistazo los runs que fallaron o que procesaron volúmenes anómalos.
- **Panel "Duración (min)"** al lado: LineChart con la duración en minutos por run. Útil para detectar degradación progresiva (runs que tardan cada vez más por crecimiento del dataset o recursos saturados).
- Ambas gráficas se ocultan cuando hay menos de 2 ejecuciones (no aporta valor una gráfica de un solo punto).

### Technical

- Nuevo helper `buildRunChartData(runs)` en `Nodes.tsx` — invierte la lista (la API devuelve newest-first) y mapea a filas `{ label, bytes, minutes, status }`.
- `useMemo` para no recomputar el dataset del gráfico en cada render del drawer (los runs se refrescan cada 10s via React Query).

---

## [0.8.2] - 2026-04-19

### Added — Mascota: Tita la hormiga 🐜

- **Identidad visual definida** — BackupSMC ahora tiene mascota oficial: una hormiga trabajadora que carga un disco de respaldo en su abdomen. Justificación: las hormigas son trabajadoras incansables, cargan cargas pesadas, operan en colonias coordinadas (igual que la arquitectura multi-agente del sistema) y culturalmente representan "guardar para el invierno". Contraparte del búho de SMC Desk (sabio/observador vs trabajador/constructor).
- **3 variantes SVG** en `frontend/src/components/AntMascot.tsx`:
  - `<AntIcon>` — silueta monocromática (usa `currentColor`), para badges sobre fondo oscuro. Reemplaza el ícono de cubos-red en Sidebar y Login header.
  - `<AntMascot>` — a color completo, carga un disco de respaldo con luz verde intermitente en el abdomen. En uso en la card de Login.
  - `<AntSpot>` — versión miniatura (24×24) para acentos futuros en headers de sección.
- **Favicon real** (`frontend/public/icon.svg`) — antes el `index.html` apuntaba a `/icon.svg` pero el archivo no existía (404 silencioso en cada pestaña). Ahora muestra la hormiga sobre fondo azul primario. Funciona legible a 16×16.
- **Anatomía correcta**: tres segmentos corporales (cabeza, tórax, abdomen con "cintura" estrecha) — la característica definitoria de una hormiga vs. abeja/escarabajo/araña. 6 patas, 2 antenas.
- **Detalle narrativo**: el disco que carga tiene líneas de datos y un LED verde pulsante (`<animate>`) que sugiere "respaldo activo".

### Changed

- Subtítulo del Sidebar: "Soporte IT" → "Respaldo empresarial" (el anterior venía copiado de otro proyecto).
- Copy del Login card: ahora dice "Tita está lista para cargar tus respaldos" (reemplaza el genérico "Ingresa tus credenciales para continuar").

### Fixed

- Favicon 404 — `index.html` referenciaba `/icon.svg` que nunca existió en el repo.

### Changed — Versiones

- Frontend `frontend/package.json`: 0.8.1 → **0.8.2** (patch — solo assets + branding).

---

## [0.8.1] - 2026-04-19

### Fixed — Limpieza de datos mock en Dashboard

- **Gráfica "Tendencia de backups — últimos 14 días"** usaba `data.recent_runs` como fuente, pero el endpoint `/dashboard` limita `recent_runs` a 10 filas. Resultado: la gráfica de 14 días mostraba casi siempre solo los últimos 1-2 días con datos, el resto en cero — dando la falsa impresión de poca actividad. Ahora consume `/stats/history?days=14` (agregación server-side real) con polling cada 60 s.

### Removed

- Botón **"Personalizar"** del Dashboard — placeholder no funcional desde el scaffolding inicial. Eliminado.
- Comentario engañoso `// Generate placeholder chart data from actual run history` reemplazado por documentación correcta del flujo de datos.

### Audit — Otras páginas

Revisadas y confirmadas sin datos mock: `Nodes`, `Jobs`, `History`, `Settings`, `Login`, `Layout`, `Sidebar`. Todas consumen API real vía React Query. Los arrays `const STATUS_OPTIONS`, `RANGE_OPTIONS`, `PROVIDERS`, `nav` son configuración estructural legítima (no datos fake).

### Changed — Versiones

- Frontend `frontend/package.json`: 0.8.0 → **0.8.1** (patch — solo frontend).

---

## [0.8.0] - 2026-04-19

### Added — Página Destinos real

- **Destinos reportados por los agentes** reemplazan los datos mock que llevaban desde el scaffolding inicial:
  - Columna `destinations_json` nueva en la tabla `nodes` (migración idempotente).
  - Campo `destinations: list[DestinationEntry]` aceptado en `POST /api/v1/nodes/register` y expuesto en `NodeOut`.
  - Endpoint nuevo `GET /api/v1/destinations` — devuelve una fila por `(nodo × destino)` con stats agregadas de runs (`bytes_backed_up`, `runs_count`, `last_backup_at`, `last_status`) y `node_status` calculado (online/stale/offline).
  - Página `/destinations`: cards de resumen por tipo (S3/SFTP/Local) + lista de destinos únicos agrupados por `type+target`, con sub-filas de nodos que comparten ese destino. Ordenada por bytes respaldados, polling cada 15 s.
  - Empty state honesto cuando no hay datos ("Requiere agent v0.2.0+").
- **Agente Go v0.2.0**:
  - Nueva función `noderegister.BuildDestinations(cfg.Destination)` que convierte `DestinationConfig` en el payload estructural (nunca incluye passwords ni key-files).
  - `Register` / `StartHeartbeat` aceptan ahora `[]Destination` y lo envían en cada heartbeat (5 min).
  - `agentVersion` bumpeado 0.1.0 → **0.2.0**.

### Removed

- Dato hardcodeado de destinos ("S3 · backupsmc 187/500 GB" y similares) en `frontend/src/pages/Destinations.tsx` — ya no existe, todo viene del backend.
- Mensaje "La configuración de destinos desde la UI estará disponible en v0.2.0" reemplazado por empty state real y leyenda sobre `agent.yaml`.

### Changed — Versiones

- Backend `server/app/main.py`: 0.7.0 → **0.8.0**.
- Frontend `frontend/package.json`: 0.7.0 → **0.8.0**.
- Go agent `agentVersion`: 0.1.0 → **0.2.0**.

---

## [0.7.0] - 2026-04-19

### Added — Resumen diario por correo

- **Módulo `app/daily_summary.py`**: agrega runs de las últimas 24 h por nodo (runs, completados, fallidos, advertencias, bytes, último timestamp) + totales globales (tasa de éxito, nodos activos, nodos sin contacto) y renderiza un correo HTML con gradiente, tabla por nodo, pills de fallos y sección de nodos offline. Fallback de texto plano incluido.
- **Scheduler APScheduler** (`app/scheduler.py`): `BackgroundScheduler` arranca con FastAPI, cron diario configurable vía env `DAILY_SUMMARY_HOUR` (default 7) y `DAILY_SUMMARY_MINUTE` (default 0), zona horaria `SCHEDULER_TZ` (default UTC). Respeta el flag `notify_daily_summary` y `notify_email_to` en cada ejecución.
- **Endpoints nuevos**:
  - `POST /api/v1/mail-settings/test-daily` — envía el resumen ahora mismo (ignora el flag `notify_daily_summary`) para previsualizar.
  - `GET /api/v1/mail-settings/daily-preview` — devuelve el JSON agregado sin enviar correo (útil para integraciones).
- **UI**: botón "Resumen diario" en la card SMTP de `/settings` — envía el digest al correo del campo de prueba (o al `notify_email_to` si está vacío), con feedback inline mostrando totales. Leyenda explicativa del cron 07:00 UTC.
- **Dependencia**: `apscheduler==3.10.4` añadida a `server/requirements.txt`.

### Changed — Versiones

- Backend `server/app/main.py`: título/health/server_version 0.6.0 → **0.7.0**.
- Frontend `frontend/package.json`: 0.6.0 → **0.7.0**.

---

## [0.6.0] - 2026-04-19

### Added — Notificaciones y correo

- **Sistema de notificaciones in-app** (replicado de SMC Desk):
  - Modelo `notifications` (id, user_id, type, title, body, entity_type, entity_id, read, created_at).
  - Service `app/notifications_service.py` con `fanout_to_all_users`, `list_for_user`, `mark_read`, `mark_all_read`, `unread_count`.
  - Endpoints: `GET /api/v1/notifications`, `PATCH /api/v1/notifications/{id}/read`, `POST /api/v1/notifications/mark-all-read`.
  - Hook en `update_progress`: al cambiar estado a `failed`/`warning`/`completed` dispara `on_job_status_change()`, respeta flags `notify_on_failure`/`notify_on_success`.
  - Tipos: `backup_completed`, `backup_failed`, `backup_warning`, `node_offline`, `general`.
  - Store Zustand `useNotifications` con polling cada 30s (sin WS por simplicidad).
  - Campana en `Layout.tsx` con badge de no-leídos, dropdown con iconos por tipo, acción "Marcar todo", navegación a `/jobs` al hacer clic en notifs con `entity_type=job_run`.

- **Configuración SMTP saliente** (replicado de SMC Desk `mail-settings`):
  - Modelo `email_settings` (single-row): provider (smtp/office365), from_name, from_email, host, port, secure, user, pass.
  - Service `app/email_service.py` usando `smtplib` stdlib (sin dependencias nuevas), SSL/STARTTLS, templates HTML para `backup_failed`/`backup_warning`/`backup_completed`/`test`.
  - Endpoints: `GET /api/v1/mail-settings` (contraseña enmascarada), `PUT /api/v1/mail-settings` (no sobreescribe password si llega como mask), `POST /api/v1/mail-settings/test`.
  - Envío automático desde `on_job_status_change` cuando `notify_email_enabled=true` + SMTP configurado.
  - Card "Servidor SMTP" en `/settings` con formulario de proveedor/from/host/puerto/SSL/usuario/contraseña y botón de prueba.

### Added — UX nodos

- **Drawer de runs por nodo** en `/nodes`: clic en el header de una Node card abre panel lateral con tasa de éxito, # runs, total GB y últimas 20 ejecuciones (con error en rojo si fallaron). Auto-refresh cada 10 s.
- **Estado computado de nodos** en el frontend desde `last_seen`: `online` (<10 min), `stale/Intermitente` (10–30 min), `offline` (>30 min).
- **Badge `stale`** nuevo en `components/Badge.tsx` (ámbar).

### Added — Deploy EC2 DEV

- Compose `docker-compose.ec2-dev.yml` + `.env.ec2-dev` (SQLite, SECRET_KEY + AGENT_TOKEN generados), contenedores prefijados `backupsmc-dev-*`, red `backupsmc-dev`, puerto 3100.
- `frontend/.env.production` con `VITE_INSTANCE_TAG=DEV` → badge rojo "DEV" en el sidebar para distinguir la instancia.
- `nginx.conf` apunta el proxy `/api/` a `backupsmc-dev-server:8000`.

### Added — Demo data

- `seed_demo.py`: 4 nodos de ejemplo (DC-FILESERVER-01, app-server-prod, finance-nas-02, LEGACY-WIN7-ARCHIVE) con 74 runs distribuidos en 30 días y perfiles de fallo distintos.
- `seed_notifs.py`: 7 notificaciones demo (4 sin leer) para mostrar la campana con contenido realista.

### Changed

- Versión de servidor y frontend bumpeada a **0.6.0** (FastAPI title, `/health`, `/api/v1/settings.server_version`, `frontend/package.json`).

---

## [0.5.0] - 2026-04-09

### Added — Agente Linux (`agent-linux/`)

- **Backup de archivos** (`internal/backup/files`): tar+gzip con exclusiones por prefijo/glob y manejo de symlinks. Binario estático `CGO_ENABLED=0` compatible con kernel ≥ 2.6.32.
- **Backup de bases de datos** (`internal/backup/databases`):
  - PostgreSQL: `pg_dump -Fc` por base, `PGPASSWORD` via env
  - MySQL/MariaDB: `mysqldump | gzip` pipeline, `--single-transaction --routines --triggers`
  - MongoDB: `mongodump --gzip`
  - Redis: `BGSAVE` + copia de `dump.rdb`
  - SQLite: `VACUUM INTO` con fallback a copia directa
  - Elasticsearch: snapshot via REST API
- **Destinos** (`internal/destination`): local, SFTP (scp/sshpass), S3 (aws cli + endpoint custom), NFS (mount -t nfs), SMB (mount -t cifs)
- **Reporter** (`internal/reporter`): heartbeat + progreso de jobs al servidor BackupSMC via HTTP (`X-Agent-Token`)
- **Engine** (`internal/backup/engine`): orquesta todas las fuentes → staging dir en `/tmp` → upload → reporte final (Option B: exits cleanly, muestra comandos útiles)
- **Service manager** (`internal/service`): instala como systemd unit con fallback automático a SysV init.d; `Install/Uninstall/Start/Stop/Restart/Status`
- **CLI Cobra** (`cmd/backupsmc-agent`): comandos `setup`, `run`, `status`, `logs`, `service`, `version` — todos con implementación real
- **Wizard TUI** (`internal/tui/wizard`): asistente de 6 pasos con charmbracelet/huh; copia/pega en token field, prueba de conexión al servidor, selección de destino NFS/SMB/S3/SFTP/local
- **`install.sh`**: soporta Debian 9+, Ubuntu 18.04+, RHEL/CentOS 6+, AlmaLinux 8+, Rocky 8+, Amazon Linux 2/2023, SUSE 12+, Fedora 32+ — amd64/arm64/arm

### Added — Frontend

- **Página History** (`frontend/src/pages/History.tsx`): estadísticas históricas reales con selector 7/14/30/90 días, gráficos de área (ejecuciones exitosas/fallidas), barras (volumen), tendencia de duración. Para 90 días agrega por semana automáticamente.
- **Job detail slide-over** (`Jobs.tsx`): clic en cualquier fila abre un panel lateral con detalle completo del job: estado, progreso, nodo, tiempos, volumen (archivos/bytes), mensaje de error si aplica.
- **StatCards de resumen** en History: tasa de éxito con indicador de color (verde ≥ 95%, ámbar ≥ 80%, rojo < 80%), total respaldado, duración promedio.

### Added — Servidor

- **`GET /api/v1/stats/history?days=N`** (default 30, rango 7–365): agrega `JobRun` por día — `total`, `completed`, `failed`, `bytes`, `duration_avg`. Rellena días sin ejecuciones con ceros.

---

## [0.4.1] - 2026-04-02

### Added

- **Retry con backoff exponencial** (`internal/retry/retry.go`): política configurable via `retry.max_attempts` + `retry.initial_delay`. Backoff: `initialDelay × 2^intento` con ±20% jitter, cap de 5 minutos. Errores permanentes (archivo inexistente, sin permiso) se marcan con `retry.Permanent{}` y no se reintenten.
- El motor de backup envuelve `backupFile` y la escritura del manifiesto con `retry.Do`; errores de red o disco transitoria se reintenten automáticamente.

### Changed

- **Inno Setup v0.4.1**: wizard de configuración con dos páginas nuevas:
  - *Server & Credentials*: URL del servidor, API token, passphrase, rutas de origen (comma-separated).
  - *Destination & Retention*: ruta local de destino, días de retención, throttle MB/s.
  - En instalaciones nuevas genera `agent.yaml` con los valores ingresados; en actualizaciones preserva el `agent.yaml` existente y salta ambas páginas.
  - Se elimina la advertencia de passphrase placeholder (ahora se valida inline durante el wizard).

## [0.4.0] - 2026-04-02

### Added — Agent (enterprise hardening)

- **Retención GFS** (`internal/backup/retention`): política Grandfather-Father-Son con ventanas diaria / semanal / mensual configurables (`keep_daily`, `keep_weekly`, `keep_monthly`). Coexiste con la política simple de días; se activa con `retention.gfs.enabled: true`.
- **SFTP destination** (`internal/destination/sftp`): destino de backup sobre SSH/SFTP con autenticación por contraseña o clave privada PEM (con passphrase opcional). Registrado en `factory.go` como `type: sftp`.
- **SHA-256 por archivo**: la firma se calcula sobre el plaintext durante el backup (TeeReader antes de compresión) y se almacena en el manifesto (`file_entry.sha256`).
- **Verificación post-escritura** (`verify_after_backup: true`): después de escribir cada objeto, se descifra y descomprime, y se recomputa el hash para detectar corrupción silenciosa.
- **Throttle de ancho de banda** (`throttle_mbps`): token-bucket puro (stdlib) aplicado sobre el lector del archivo fuente, evitando saturar el enlace durante backups. Funciona tanto en archivos normales como en chunks de archivos grandes.
- **Pre/post scripts** (`pre_script` / `post_script`): comandos shell que se ejecutan antes y después de cada job. El pre-script aborta el backup si termina con código != 0. El post-script siempre se ejecuta (incluso en fallo).
- **Caché incremental persistente**: movida de `%TEMP%` a `%ProgramData%\BackupSMC\state\` via `config.CachePath()`. Sobrevive reinicios del sistema.
- **Windows Event Log** (`internal/notify/eventlog_windows.go`): escribe eventos de éxito/advertencia/error en el Event Viewer de Windows con auto-registro de la fuente. Stub no-op para Linux.
- **Notificaciones email** (`internal/notify/email.go`): SMTP con STARTTLS (587) y TLS (465), plantillas HTML para éxito / advertencia / fallo. Configuradas desde `notify.email.*` en `agent.yaml`.
- **`agent.example.yaml` completo**: documenta todas las opciones nuevas — GFS, SFTP, throttle, pre/post scripts, verify, notify — con comentarios explicativos.

### Changed — Agent

- `engine.go`: `backupFile()` ahora devuelve el hash SHA-256 y acepta `doCompress bool` y `throttleMbps float64`. Se elimina el parámetro `skipCompressExts` de la firma interna (se resuelve antes de la llamada).
- `main.go`: `runRetention()` usa `cfg.Retention` (tipo `config.RetentionConfig`) en lugar del campo depreciado `cfg.Backup.RetentionDays`. `runOnce()` integra notificaciones email y Event Log al terminar cada job. `validate` muestra las opciones nuevas.
- `retention.go` reescrito con soporte GFS completo; la API pasa de `retention.Config{Days}` a `config.RetentionConfig`.

### Added — Server / Frontend (sesión anterior)

- **Autenticación JWT** (`server/app/auth.py`): login con usuario/contraseña (`sha256_crypt`), token Bearer para el frontend, header `X-Agent-Token` para los agentes.
- **Registro de nodos + heartbeat** (`agent/internal/noderegister`): el agente se registra al arrancar y envía heartbeat cada 5 minutos con las rutas de origen actuales.
- **Gestión de rutas en el frontend** (`frontend/src/pages/Nodes.tsx`): lista de `source_paths` por nodo con botón para agregar/eliminar rutas.
- **Login page** (`frontend/src/pages/Login.tsx`): formulario oscuro con Zustand persist + interceptor 401 que redirige a `/login`.

### Fixed

- `reporter.go`: cabecera de autenticación corregida a `X-Agent-Token` (antes enviaba `Authorization: Bearer`).
- `gui/exec_windows.go`: todos los `exec.Command` del GUI Wails usan `CREATE_NO_WINDOW` para eliminar el destello de ventana CMD.
- Servidor: `passlib[bcrypt]` incompatible con Python 3.14 reemplazado por `sha256_crypt`.

---

## [0.3.0] - 2026-03-XX

### Added
- Estructura inicial del proyecto
- Documentación base: README, ARCHITECTURE, BUSINESS_MODEL, ROADMAP
- Configuración Docker (dev y producción)
- GitHub Actions: CI pipeline y release workflow
- Templates de issues y pull requests

---

[Unreleased]: https://github.com/smcsoluciones/backup-system/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/smcsoluciones/backup-system/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/smcsoluciones/backup-system/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/smcsoluciones/backup-system/releases/tag/v0.3.0
