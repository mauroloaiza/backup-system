# Arquitectura técnica — BackupSMC

## Visión general

BackupSMC sigue una arquitectura de **microservicios desacoplados** compuesta por tres componentes principales: el **Servidor** (control plane), el **Agente** (data plane) y el **Frontend** (UI). La comunicación entre ellos se realiza mediante REST API y un protocolo interno agente-servidor.

---

## Diagrama de alto nivel

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENTE / BROWSER                    │
└───────────────────────┬─────────────────────────────────────┘
                        │ HTTPS
┌───────────────────────▼─────────────────────────────────────┐
│                    FRONTEND (React)                         │
│  Dashboard · Configuración · Historial · Alertas            │
└───────────────────────┬─────────────────────────────────────┘
                        │ REST API (HTTPS)
┌───────────────────────▼─────────────────────────────────────┐
│                    SERVER (FastAPI)                         │
│  ┌─────────────┐  ┌────────────┐  ┌──────────────────────┐ │
│  │  API REST   │  │  Scheduler │  │  Notification Engine │ │
│  │  (v1)       │  │  Celery    │  │  Email / Webhook      │ │
│  └──────┬──────┘  └─────┬──────┘  └──────────────────────┘ │
│         │               │                                    │
│  ┌──────▼───────────────▼──────────────────────────────┐   │
│  │            PostgreSQL  ·  Redis                      │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────────────┬─────────────────────────────────────┘
                        │ Agent Protocol (HTTPS + token)
          ┌─────────────┼──────────────┐
          │             │              │
┌─────────▼──┐  ┌───────▼────┐  ┌─────▼──────┐
│  Agent     │  │  Agent     │  │  Agent     │
│  Nodo 1    │  │  Nodo 2    │  │  Nodo N    │
│  (Go)      │  │  (Go)      │  │  (Go)      │
└─────┬──────┘  └────────────┘  └────────────┘
      │
      │  Lee fuentes locales:
      │  DB · Archivos · Docker · VM
      │
      ▼
┌─────────────────────────────────────────────────┐
│               DESTINOS DE ALMACENAMIENTO        │
│  Local  ·  S3/MinIO  ·  SFTP  ·  Google Drive  │
└─────────────────────────────────────────────────┘
```

---

## Componentes

### 1. Server (FastAPI)

**Responsabilidades:**
- Autenticación y autorización (JWT + API Keys)
- Gestión de nodos, jobs y políticas de backup
- Scheduler de jobs (Celery Beat)
- Registro de historial y logs
- Motor de notificaciones
- API REST consumida por el frontend y el agente

**Stack:**
```
Python 3.12
FastAPI 0.115+
SQLAlchemy 2.x (async) + Alembic (migraciones)
Celery 5.x + Redis (broker)
Pydantic v2 (validación y serialización)
PostgreSQL 16 (base de datos principal)
```

**Estructura interna:**
```
server/
└── app/
    ├── api/           # Routers FastAPI por recurso
    │   ├── v1/
    │   │   ├── auth.py
    │   │   ├── nodes.py
    │   │   ├── jobs.py
    │   │   ├── backups.py
    │   │   ├── destinations.py
    │   │   └── notifications.py
    ├── models/        # SQLAlchemy ORM models
    ├── schemas/       # Pydantic schemas (request/response)
    ├── services/      # Lógica de negocio
    ├── tasks/         # Celery tasks
    ├── core/          # Config, security, dependencies
    └── main.py
```

---

### 2. Agent (Go)

**Responsabilidades:**
- Registrarse en el servidor con token único
- Ejecutar jobs de backup asignados por el servidor
- Recopilar datos de fuentes locales (DB, archivos, Docker, VM)
- Comprimir y cifrar los datos antes de transferirlos
- Transferir al destino configurado vía rclone / SDK nativo
- Reportar estado y progreso al servidor en tiempo real

**Stack:**
```
Go 1.22+
HTTP client (net/http estándar)
rclone (librería o CLI integrado)
mysqldump / pg_dump (syscall)
Docker SDK (github.com/docker/docker)
AES-256-GCM (cifrado)
```

**Flujo del agente:**
```
Inicio
  └─► Registro en servidor (token)
        └─► Poll de jobs asignados (cada N seg)
              └─► Ejecutar job
                    ├─► Conectar fuente
                    ├─► Dump / tar / snapshot
                    ├─► Comprimir (gzip/zstd)
                    ├─► Cifrar (AES-256)
                    ├─► Transferir a destino
                    └─► Reportar resultado al servidor
```

---

### 3. Frontend (React)

**Stack:**
```
React 18
Vite 5 (bundler)
TypeScript
Tailwind CSS v3
shadcn/ui (componentes)
Recharts (gráficas)
React Query (estado servidor)
React Router v6
Zustand (estado global)
```

**Páginas principales:**
- `/dashboard` — Resumen general, estado de jobs recientes
- `/nodes` — Gestión de nodos/agentes
- `/jobs` — Configuración y listado de jobs de backup
- `/history` — Historial de ejecuciones con logs
- `/destinations` — Configurar destinos de almacenamiento
- `/settings` — Configuración general, notificaciones, usuarios

---

## Flujo de un backup completo

```
1. Usuario configura un Job (fuente + destino + cron)
2. Celery Beat dispara el job según el cron
3. Servidor encola la tarea en Redis
4. Worker Celery notifica al Agente vía API
5. Agente ejecuta el backup:
   a. Conecta a la fuente (DB / archivos / Docker / VM)
   b. Genera el dump / tar / snapshot
   c. Comprime con zstd
   d. Cifra con AES-256-GCM (clave derivada por job)
   e. Transfiere al destino (Local/S3/SFTP/GDrive)
6. Agente reporta resultado (exitoso / error + tamaño + duración)
7. Servidor actualiza historial en PostgreSQL
8. Servidor envía notificación (email / webhook) si corresponde
9. Servidor aplica política de retención (elimina backups antiguos)
```

---

## Seguridad

| Aspecto | Implementación |
|---------|----------------|
| Autenticación usuarios | JWT (access + refresh token) |
| Autenticación agentes | API Key por nodo (HMAC-SHA256) |
| Transporte | HTTPS/TLS en todos los endpoints |
| Cifrado de backups | AES-256-GCM, clave por job |
| Clave de cifrado | Derivada de `SECRET_KEY` + `job_id` (PBKDF2) |
| Contraseñas DB | bcrypt (usuarios) |
| Secrets | Variables de entorno, nunca en código |

---

## Escalabilidad

- Los **workers Celery** pueden escalarse horizontalmente añadiendo réplicas
- El **agente Go** puede instalarse en N nodos sin límite (según plan)
- **PostgreSQL** soporta read replicas para escalar lecturas
- **Redis Cluster** para alta disponibilidad del broker
- El servidor puede desplegarse detrás de un **load balancer** (nginx/traefik)

---

## Decisiones de diseño

| Decisión | Alternativas consideradas | Razón de elección |
|----------|--------------------------|-------------------|
| Python + FastAPI (backend) | Django, Node.js, Go | Ecosistema maduro para ops, async nativo, OpenAPI auto |
| Go (agente) | Python agent, Bash | Binario único sin runtime, bajo consumo, fácil distribución |
| React + shadcn/ui | Vue, Angular, Next.js | Velocidad de desarrollo, componentes headless, Tailwind |
| Celery + Redis | APScheduler, cron nativo | Cola distribuida, reintentos, visibilidad con Flower |
| PostgreSQL | SQLite, MySQL | JSONB para metadata flexible, robustez, extensiones |
| rclone (transferencias) | Implementar propio | Soporte nativo de 40+ providers, battle-tested |
