# Roadmap — SMC Backup

> Última actualización: **v0.10.0** — 2026-04-20

---

## Leyenda

| Símbolo | Estado |
|---------|--------|
| ✅ | Completado |
| 🔄 | En progreso |
| 📋 | Planificado |
| 💡 | Idea / Por evaluar |

---

## Fase 0 — Fundación ✅ CERRADA (v0.1.0)

- ✅ Definir stack tecnológico (FastAPI + Go + React + Wails)
- ✅ Estructurar repositorio y documentación base
- ✅ Definir modelo de negocio y roadmap
- ✅ Crear `docker-compose` de desarrollo y producción
- ✅ Crear archivos de gobernanza (LICENSE, CONTRIBUTING, CODE_OF_CONDUCT)

---

## Fase 1 — MVP Core ✅ CERRADA (v0.3.0)

### Backend
- ✅ Autenticación JWT (access token) + X-Agent-Token para agentes
- ✅ Gestión de nodos/agentes (registro, heartbeat, listado, estado)
- ✅ Gestión de jobs de backup (progreso, historial)
- ✅ Seed de usuario admin desde variables de entorno
- ✅ Migración idempotente de columnas (ALTER TABLE sin Alembic)
- ✅ API REST v1 documentada

### Agente (Go)
- ✅ Instalador Inno Setup con wizard de configuración (URL, token, passphrase, rutas, destino, retención)
- ✅ Registro automático en servidor con token (`X-Agent-Token`)
- ✅ Heartbeat periódico cada 5 minutos con `source_paths`
- ✅ Backup de directorios (compresión zstd + cifrado AES-256-GCM)
- ✅ Backup incremental con caché persistente (`%ProgramData%\BackupSMC\state\`)
- ✅ Soporte VSS (Volume Shadow Copy) para archivos abiertos
- ✅ Destino local
- ✅ Destino S3 / compatible
- ✅ Reporte de estado y progreso al servidor en tiempo real

### Frontend
- ✅ Layout base con sidebar y navegación
- ✅ Página de login (dark theme)
- ✅ Autenticación persistente (Zustand + localStorage + interceptor 401)
- ✅ Dashboard con listado de nodos y estado
- ✅ Gestión de `source_paths` por nodo (agregar / eliminar inline)
- ✅ Logout con limpieza de token

### Infraestructura
- ✅ Docker Compose (dev + producción)
- ✅ GUI desktop Wails v2 (`BackupSMC.exe`) sin parpadeo de ventana CMD

---

## Fase 2 — Enterprise Agent ✅ CERRADA (v0.4.1)

- ✅ Destino SFTP (clave PEM o contraseña)
- ✅ SHA-256 por archivo — calculado durante backup, almacenado en manifiesto
- ✅ Verificación post-escritura (`verify_after_backup`) — descifra y recomputa hash
- ✅ Throttle de ancho de banda — token bucket puro stdlib (`throttle_mbps`)
- ✅ Pre/post scripts — `cmd.exe /C` en Windows, `sh -c` en Linux
- ✅ Retención simple por días (`retention.days`)
- ✅ Retención GFS — Grandfather-Father-Son diario/semanal/mensual
- ✅ Retry con backoff exponencial — `InitialDelay × 2^intento` ±20% jitter, cap 5 min
- ✅ Notificaciones email SMTP (STARTTLS/TLS) con plantillas HTML
- ✅ Windows Event Log — auto-registro de fuente, tipos info/warning/error
- ✅ Archivos grandes (>512 MB) divididos en chunks cifrados individualmente
- ✅ ACL de Windows (SDDL) preservadas y restaurables
- ✅ CLI de restauración (`restore --job-id --target --filter --dry-run`)

---

## Fase 3 — Dashboard & UX 🔄 En progreso (mayoría cerrada v0.5.0 → v0.9.0)

### Frontend
- ✅ Página de historial de jobs (tabla con paginación, filtros, estado) — v0.5.0
- ✅ Detalle de job — archivos respaldados, errores, duración, tamaño (slideover) — v0.5.0
- 📋 Wizard de restauración desde UI (seleccionar job → ruta destino → ejecutar)
- ✅ Gráfica de tendencias (tamaño acumulado, tasa de éxito) con Recharts — v0.5.0
- ✅ Página de configuración del agente desde UI (editar `source_paths`, destino, retención) — v0.9.0 (Agent Config Editor remoto vía polling `/config/pull`)
- ✅ Indicador de estado del agente (online / stale / offline / última vez visto) — v0.6.0

### Backend
- ✅ Endpoint de historial de jobs con paginación — v0.5.0 (`/api/v1/stats/history`)
- ✅ Endpoint de detalle de job (leer manifiesto cifrado) — v0.5.0
- 📋 Endpoint para lanzar restauración remota
- 📋 WebSocket o SSE para progreso en tiempo real en UI

### Agente
- 📋 SFTP: soporte `known_hosts` (actualmente `InsecureIgnoreHostKey`)
- ✅ Retry en heartbeat si el servidor está caído al arrancar
- ✅ Polling de configuración remota (`/config/pull`) — agente v0.2.0 (v0.8.0)

---

## Fase 4 — Multi-destino y Fuentes Avanzadas (Q3 2026)

- 📋 Destino Google Drive (via rclone)
- 📋 Destino MinIO / Backblaze B2
- 📋 Destino Azure Blob Storage
- ✅ Backup de base de datos MySQL (mysqldump + cifrado) — agent-linux v0.5.0
- ✅ Backup de base de datos PostgreSQL (pg_dump + cifrado) — agent-linux v0.5.0
- 📋 Backup de volúmenes Docker (pause + tar del volumen)
- 📋 Prueba de conectividad desde UI antes de guardar destino

---

## Fase 5 — Notificaciones Avanzadas (Q3 2026)

- ✅ Notificaciones in-app (centro de notificaciones + badge) — v0.6.0
- ✅ Resumen diario por email (APScheduler BackgroundScheduler) — v0.7.0
- ✅ Alertas por agente desconectado > N horas (stale badge) — v0.6.0
- 📋 Notificaciones por webhook (Slack, Teams, Discord)
- 📋 Alertas por espacio en destino bajo umbral
- 📋 Exportar historial a CSV/PDF
- 📋 Notificaciones WhatsApp (integración SMC Desk)

---

## Fase 6 — Multi-tenant y Usuarios (Q4 2026)

- 📋 Roles: Admin, Operador, Viewer
- 📋 Múltiples organizaciones (modelo multi-tenant)
- 📋 Invitación de usuarios por email
- 📋 Auditoría de acciones
- 📋 2FA (TOTP)

---

## Fase 7 — SaaS y On-Premise (Q1 2027)

- 📋 Portal de billing y suscripciones (Stripe)
- 📋 Gestión de planes y límites por tenant
- 📋 Documentación pública
- 📋 Página web de marketing (backupsmc.com)
- 📋 Integración con SMC Desk — backup automático desde ticket de recuperación

---

## Infraestructura & Operación (cerrado v0.10.0)

- ✅ Deploy EC2 DEV con docker-compose (`/opt/backupsmc-dev`) — v0.6.0
- ✅ Migración de SQLite a PostgreSQL con `bootstrap_migrate.py` — v0.10.0
- ✅ Destinos persistidos como `destinations_json` en nodos — v0.8.0

---

## Versiones

| Versión | Fase | Fecha |
|---------|------|-------|
| v0.1.0 | Fundación | ✅ 2026-03 |
| v0.3.0 | MVP Core | ✅ 2026-03 |
| v0.4.1 | Enterprise Agent | ✅ 2026-04-02 |
| v0.5.0 | agent-linux + History/Charts + Job detail | ✅ 2026-04-09 |
| v0.6.0 | Notificaciones in-app + estado stale + EC2 DEV | ✅ 2026-04-19 |
| v0.7.0 | Resumen diario por email (APScheduler) | ✅ 2026-04-19 |
| v0.8.0 | Destinos reales + `destinations_json` + agente 0.2.0 | ✅ 2026-04-19 |
| v0.9.0 | Agent Config Editor remoto (polling) | ✅ 2026-04-20 |
| v0.10.0 | Migración PostgreSQL | ✅ 2026-04-20 |
| v0.11.0 | Restore wizard UI + endpoint remoto | 📋 Q2 2026 |
| v0.12.0 | WebSocket/SSE de progreso + SFTP `known_hosts` | 📋 Q2 2026 |
| v1.0.0 | Lanzamiento SaaS | 📋 Q2 2027 |
