# BackupSMC

> Enterprise backup solution — self-hosted & cloud managed. Inspired by Veeam/Acronis.

[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)](./LICENSE)
[![Status: In Development](https://img.shields.io/badge/Status-In%20Development-yellow.svg)]()

---

## ¿Qué es BackupSMC?

**BackupSMC** es una plataforma de backup empresarial desarrollada por **SMC Soluciones**, diseñada para proteger datos críticos de organizaciones con soporte para múltiples fuentes y destinos de almacenamiento.

Disponible en modalidad **SaaS gestionado** (cloud) y **On-Premise** (self-hosted).

---

## Características principales

- **Multi-fuente:** Bases de datos MySQL/PostgreSQL, archivos, volúmenes Docker, VMs y servidores completos
- **Multi-destino:** Almacenamiento local, S3/MinIO/Backblaze, SFTP y Google Drive
- **Agente ligero (Go):** Se instala en cualquier servidor o VM, sin dependencias externas
- **Dashboard web:** Panel de control React para gestión de jobs, historial y alertas
- **Scheduler flexible:** Cron configurable por job, con reintentos automáticos
- **Compresión y cifrado:** Backups comprimidos y cifrados en tránsito y en reposo
- **Retención configurable:** Políticas de retención por job (diaria, semanal, mensual)
- **Notificaciones:** Alertas por email y webhook al completar o fallar un backup

---

## Arquitectura

```
┌──────────────┐     REST API      ┌─────────────────┐
│  Frontend    │◄─────────────────►│   Server        │
│  React/Vite  │                   │   FastAPI       │
└──────────────┘                   │   + Celery      │
                                   │   + PostgreSQL  │
                                   └────────┬────────┘
                                            │ Agent Protocol
                              ┌─────────────▼──────────────┐
                              │   Agent (Go)               │
                              │   Instalado en cada nodo   │
                              └────────────────────────────┘
```

| Componente | Tecnología              |
|------------|-------------------------|
| Backend    | Python 3.12 + FastAPI   |
| Scheduler  | Celery + Redis          |
| Base datos | PostgreSQL 16           |
| Agente     | Go 1.22                 |
| Frontend   | React 18 + Vite + Tailwind CSS |
| Transferencia | rclone + boto3       |

---

## Inicio rápido (Docker)

### Pre-requisitos
- Docker 24+
- Docker Compose v2

### Levantar en desarrollo

```bash
git clone https://github.com/smcsoluciones/backup-system.git
cd backup-system
cp .env.example .env
# Editar .env con tus configuraciones
docker compose -f docker-compose.dev.yml up -d
```

Accede al dashboard en: `http://localhost:3000`
API docs (Swagger): `http://localhost:8000/docs`

### Levantar en producción

```bash
docker compose up -d
```

---

## Estructura del proyecto

```
backup-system/
├── server/          # Backend FastAPI
├── agent/           # Agente Go
├── frontend/        # Dashboard React
├── docs/            # Documentación
├── .github/         # CI/CD y templates
├── docker-compose.yml
├── docker-compose.dev.yml
└── .env.example
```

---

## Documentación

| Documento | Descripción |
|-----------|-------------|
| [ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Arquitectura técnica detallada |
| [BUSINESS_MODEL.md](./docs/BUSINESS_MODEL.md) | Modelo de negocio y tiers |
| [ROADMAP.md](./docs/ROADMAP.md) | Fases y funcionalidades planeadas |
| [DEVELOPMENT.md](./docs/DEVELOPMENT.md) | Guía para desarrolladores |
| [DEPLOYMENT.md](./docs/DEPLOYMENT.md) | Guía de despliegue en producción |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Cómo contribuir al proyecto |
| [CHANGELOG.md](./CHANGELOG.md) | Historial de cambios |

---

## Tiers / Planes

| Feature | Free | Pro | Enterprise |
|---------|------|-----|------------|
| Agentes | 1 | 5 | Ilimitados |
| Retención | 7 días | 90 días | Personalizada |
| Destinos | Local | Local + S3 + SFTP | Todos |
| Soporte | Community | Email | Dedicado |
| SLA | — | 99.5% | 99.9% |

---

## Seguridad

Para reportar vulnerabilidades de seguridad, por favor contacta directamente a:
📧 **security@smcsoluciones.com**

No crear issues públicos para reportar vulnerabilidades.

---

## Licencia

Copyright © 2026 SMC Soluciones. Todos los derechos reservados.
Ver [LICENSE](./LICENSE) para más información.

---

Desarrollado con ❤️ por [SMC Soluciones](https://smcsoluciones.com)
