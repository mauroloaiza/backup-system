# Roadmap — BackupSMC

> Estado actual: **Fase 0 — Fundación** (Q1-Q2 2026)

---

## Leyenda de estados

| Símbolo | Estado |
|---------|--------|
| ✅ | Completado |
| 🔄 | En progreso |
| 📋 | Planificado |
| 💡 | Idea / Por evaluar |

---

## Fase 0 — Fundación (Q1-Q2 2026)

> Estructurar el proyecto, definir arquitectura y convenciones de desarrollo.

- ✅ Definir stack tecnológico (FastAPI + Go + React)
- ✅ Estructurar repositorio y documentación base
- ✅ Definir modelo de negocio y roadmap
- ✅ Crear `docker-compose` de desarrollo y producción
- ✅ Crear archivos de gobernanza (LICENSE, CONTRIBUTING, CODE_OF_CONDUCT)
- 📋 Configurar repositorio GitHub privado
- 📋 Configurar CI/CD con GitHub Actions
- 📋 Definir convenciones de base de datos (modelos y migraciones)
- 📋 Scaffold inicial del servidor FastAPI
- 📋 Scaffold inicial del agente Go
- 📋 Scaffold inicial del frontend React

---

## Fase 1 — MVP Core (Q2-Q3 2026)

> Primera versión funcional para uso interno en SMC Soluciones.

### Backend
- 📋 Autenticación: registro, login, JWT (access + refresh)
- 📋 Gestión de nodos/agentes (registro, listado, estado)
- 📋 Gestión de jobs de backup (CRUD, cron config)
- 📋 Ejecución de jobs vía Celery
- 📋 Historial de ejecuciones y logs
- 📋 API de destinos: configurar local y S3
- 📋 Migraciones con Alembic
- 📋 Tests unitarios y de integración (pytest, cobertura > 70%)

### Agente (Go)
- 📋 Instalación con un comando (curl | bash)
- 📋 Registro automático en servidor con token
- 📋 Backup de MySQL (mysqldump + compresión gzip)
- 📋 Backup de PostgreSQL (pg_dump + compresión gzip)
- 📋 Backup de directorios (tar + gzip)
- 📋 Transferencia a destino local
- 📋 Transferencia a S3 (boto3/aws-sdk)
- 📋 Reporte de estado y progreso al servidor

### Frontend
- 📋 Layout base con sidebar y navegación
- 📋 Página de login
- 📋 Dashboard con resumen de estado
- 📋 Listado y detalle de nodos
- 📋 Crear y listar jobs de backup
- 📋 Historial de ejecuciones con logs

### Infraestructura
- 📋 Pipeline CI: lint + tests en cada PR
- 📋 Pipeline CD: build y push de imágenes Docker a registry

---

## Fase 2 — Destinos múltiples (Q3 2026)

> Ampliar soporte de destinos de almacenamiento.

- 📋 Destino SFTP (via rclone)
- 📋 Destino Google Drive (via rclone)
- 📋 Destino MinIO / Backblaze B2
- 📋 Interfaz para gestionar múltiples destinos en UI
- 📋 Prueba de conectividad desde UI antes de guardar destino
- 📋 Rotación automática de backups según política de retención

---

## Fase 3 — Fuentes avanzadas (Q3-Q4 2026)

> Soporte para fuentes más complejas.

- 📋 Backup de volúmenes Docker (pause + tar del volumen)
- 📋 Listado de contenedores activos desde agente
- 📋 Backup completo de servidor (tar del filesystem)
- 📋 Snapshot de VM (integración inicial con libvirt/KVM)
- 📋 Exclusión de rutas/archivos por patrón (.gitignore style)
- 📋 Cifrado de backups (AES-256-GCM, clave por job)

---

## Fase 4 — Notificaciones y monitoreo (Q4 2026)

> Visibilidad completa del estado del sistema.

- 📋 Notificaciones por email (SMTP configurable)
- 📋 Notificaciones por webhook (compatible Slack, Teams, Discord)
- 📋 Alertas por job fallido, espacio bajo, agente desconectado
- 📋 Dashboard con métricas: tamaño acumulado, tasa de éxito, tendencias
- 📋 Gráficas de ejecuciones (Recharts)
- 📋 Exportar historial a CSV/PDF

---

## Fase 5 — Multi-tenant y usuarios (Q1 2027)

> Soporte para múltiples organizaciones y gestión de usuarios.

- 📋 Modelo multi-tenant (organizaciones separadas)
- 📋 Roles y permisos: Admin, Operador, Viewer
- 📋 Invitación de usuarios por email
- 📋 Auditoría de acciones (quién hizo qué y cuándo)
- 📋 2FA (TOTP) para usuarios

---

## Fase 6 — SaaS y On-Premise (Q1-Q2 2027)

> Preparación para lanzamiento comercial.

- 📋 Portal de billing y suscripciones (Stripe)
- 📋 Gestión de planes y límites por tenant
- 📋 Wizard de onboarding para nuevos clientes
- 📋 Documentación pública (docs.backupsmc.com)
- 📋 Installer one-click On-Premise (script + Docker)
- 📋 Portal de licencias On-Premise
- 📋 Página web de marketing (backupsmc.com)

---

## Fase 7 — Integraciones y ecosistema (Q2-Q3 2027)

> Conectar BackupSMC con otras herramientas.

- 📋 Integración con SMC Desk (GLPI): backup automático al crear ticket tipo "recuperación"
- 📋 API pública v2 para integraciones de terceros
- 📋 Plugin/webhook para Grafana (métricas)
- 📋 Integración con Prometheus + Alertmanager
- 💡 CLI (`backupsmc-cli`) para administración desde terminal
- 💡 App móvil (React Native) para monitoreo
- 💡 Marketplace de destinos (plugins de comunidad)

---

## Backlog / Ideas futuras

- 💡 Deduplicación de bloques (block-level dedup)
- 💡 Backup incremental (solo cambios desde último backup)
- 💡 Verificación automática de backups (restore test)
- 💡 Disaster Recovery automatizado
- 💡 Integración con Azure Blob Storage / GCS
- 💡 Soporte para Windows en el agente
- 💡 Interfaz de restauración (restore wizard desde UI)
- 💡 Programar ventanas de mantenimiento (excluir horarios)

---

## Versiones planeadas

| Versión | Fase | Fecha estimada |
|---------|------|----------------|
| v0.1.0 | Fase 0 completa | Q2 2026 |
| v0.2.0 | Fase 1 MVP | Q3 2026 |
| v0.5.0 | Fases 2 + 3 | Q4 2026 |
| v0.8.0 | Fases 4 + 5 | Q1 2027 |
| v1.0.0 | Lanzamiento SaaS | Q2 2027 |
| v1.x.x | Fases 7+ | Q3 2027+ |
