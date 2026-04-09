# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es/1.0.0/)
y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
