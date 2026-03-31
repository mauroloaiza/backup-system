# Modelo de negocio — BackupSMC

## Propuesta de valor

**BackupSMC** protege los datos críticos de las organizaciones ofreciendo una solución de backup empresarial **confiable, multi-destino y fácil de gestionar**, disponible tanto en la nube gestionada por SMC Soluciones como instalada en la infraestructura del propio cliente.

> "Veeam/Acronis para PyMEs latinoamericanas, con soporte en español y precios accesibles."

---

## Segmento de clientes

### Primario
- **PyMEs** de 10 a 500 empleados en LATAM
- Con infraestructura crítica: bases de datos, servidores de archivos, entornos Docker
- Sin equipo de IT dedicado a backup
- Que buscan alternativas más accesibles que Veeam Enterprise

### Secundario
- **MSPs (Managed Service Providers)** que quieren ofrecer backup como servicio a sus clientes
- **Empresas medianas** con requisitos de cumplimiento (RGPD, ISO 27001, SOC 2) que necesitan evidencia de backups

---

## Modelo de entrega

### SaaS Cloud (gestionado por SMC Soluciones)
- El servidor BackupSMC corre en infraestructura de SMC
- El cliente instala solo el **agente ligero** en sus servidores
- Sin mantenimiento de base de datos ni del servidor de control
- Acceso via `https://app.backupsmc.com`

### On-Premise (self-hosted)
- El cliente despliega toda la stack con Docker Compose en su propia infraestructura
- Ideal para clientes con restricciones de datos (bancario, gobierno, salud)
- Licencia anual por número de nodos o agentes

---

## Tiers / Planes

### SaaS

| Feature | Free | Pro | Business | Enterprise |
|---------|------|-----|----------|------------|
| **Precio** | Gratis | $15/mes | $49/mes | A convenir |
| **Agentes** | 1 | 5 | 25 | Ilimitados |
| **Almacenamiento incluido** | 10 GB | 100 GB | 500 GB | Personalizado |
| **Retención** | 7 días | 30 días | 90 días | Personalizada |
| **Fuentes** | Archivos | Archivos + DB | Archivos + DB + Docker | Todas |
| **Destinos** | Local/S3 | Local/S3/SFTP | Todos | Todos |
| **Frecuencia mínima** | Diario | Cada hora | Cada 15 min | Cada 5 min |
| **Notificaciones** | Email | Email + Webhook | Email + Webhook + Slack | Todas |
| **Soporte** | Community | Email (48h) | Email (8h) | Dedicado (SLA) |
| **SLA uptime** | — | 99.5% | 99.9% | 99.95% |
| **Usuarios** | 1 | 3 | 10 | Ilimitados |

### On-Premise

| Tipo | Precio | Incluye |
|------|--------|---------|
| Starter | $299/año | Hasta 5 agentes, actualizaciones 1 año |
| Business | $799/año | Hasta 25 agentes, actualizaciones + soporte email |
| Enterprise | Desde $2,000/año | Agentes ilimitados, SLA, soporte dedicado, onboarding |

---

## Flujo de ingresos

```
┌─────────────────────────────────────┐
│  INGRESOS RECURRENTES (MRR)         │
│  ├─ Suscripciones SaaS mensuales    │
│  └─ Licencias On-Premise anuales    │
├─────────────────────────────────────┤
│  INGRESOS COMPLEMENTARIOS           │
│  ├─ Almacenamiento adicional (S3)   │
│  ├─ Agentes adicionales             │
│  ├─ Servicios de implementación     │
│  └─ Soporte premium / SLA           │
└─────────────────────────────────────┘
```

---

## Estrategia de adquisición

### Canal principal: SMC Soluciones (interno)
- Ofrecerlo como servicio adicional a clientes existentes de SMC Desk / soporte IT
- Bundling: "SMC Desk + BackupSMC" con descuento

### Canal secundario: MSPs
- Programa de partners MSP: margen del 20-30% para revendedores
- White-label disponible para Enterprise

### Marketing digital
- SEO: "backup para servidores linux", "backup mysql automatico"
- Trials: plan Free actúa como lead magnet
- Contenido técnico (blog, YouTube): tutoriales de backup y recuperación

---

## Métricas clave (KPIs)

| Métrica | Objetivo Año 1 |
|---------|---------------|
| MRR | $3,000 USD |
| Clientes activos (SaaS) | 50 |
| Licencias On-Premise vendidas | 10 |
| Churn mensual | < 3% |
| NPS | > 50 |
| Tiempo medio de setup | < 15 min |

---

## Estructura de costos (SaaS cloud)

| Costo | Tipo | Estimado mensual |
|-------|------|-----------------|
| Infraestructura (VPS/cloud) | Variable | $80-200 USD |
| Almacenamiento S3 | Variable (por cliente) | ~ $0.023/GB |
| Desarrollo (horas SMC) | Fijo | — |
| Soporte | Variable | — |
| Herramientas (CI/CD, monitoring) | Fijo | $30-50 USD |

---

## Ventaja competitiva

| Aspecto | BackupSMC | Veeam | Acronis |
|---------|-----------|-------|---------|
| Precio LATAM | ✅ Accesible | ❌ Caro | ❌ Caro |
| Self-hosted | ✅ Sí | ✅ Sí | ⚠️ Limitado |
| Soporte en español | ✅ Nativo | ❌ No | ⚠️ Limitado |
| Open-source base | ❌ Propietario | ❌ | ❌ |
| Integración SMC Desk | ✅ Futura | ❌ | ❌ |
| Setup < 15 min | ✅ | ❌ Complejo | ⚠️ |

---

## Roadmap comercial

| Fase | Período | Hito |
|------|---------|------|
| MVP interno | Q2 2026 | Backup DB + archivos, 1 destino S3 |
| Beta privada | Q3 2026 | 5 clientes piloto, feedback |
| Lanzamiento SaaS | Q4 2026 | Plan Free + Pro disponibles |
| On-Premise v1 | Q1 2027 | Licencias self-hosted |
| MSP Program | Q2 2027 | Portal para partners |
| Integración SMC Desk | Q3 2027 | Backup automático desde tickets |
