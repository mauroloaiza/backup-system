# Guía de despliegue — BackupSMC

## Requisitos del servidor

| Recurso | Mínimo | Recomendado |
|---------|--------|-------------|
| CPU | 2 vCPU | 4 vCPU |
| RAM | 2 GB | 4 GB |
| Disco | 20 GB | 50 GB + almacenamiento backups |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |
| Docker | 24+ | 24+ |
| Docker Compose | v2 | v2 |
| Dominio | Opcional | Recomendado (HTTPS) |

---

## Instalación en producción

### 1. Preparar el servidor

```bash
# Actualizar sistema
sudo apt update && sudo apt upgrade -y

# Instalar Docker
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER
newgrp docker

# Verificar
docker --version
docker compose version
```

### 2. Clonar y configurar

```bash
# Clonar repositorio (o transferir los archivos)
git clone https://github.com/smcsoluciones/backup-system.git
cd backup-system

# Configurar variables de entorno
cp .env.example .env
nano .env
```

**Variables críticas a cambiar en producción:**

```env
APP_ENV=production
DEBUG=false
SECRET_KEY=<genera con: openssl rand -hex 32>
POSTGRES_PASSWORD=<contraseña segura>
AGENT_SECRET=<genera con: openssl rand -hex 32>
CORS_ORIGINS=https://tu-dominio.com
```

### 3. Levantar servicios

```bash
# Construir y levantar
docker compose up -d --build

# Verificar estado
docker compose ps

# Ver logs
docker compose logs -f
```

### 4. Verificar

```bash
# Health check del servidor
curl http://localhost:8000/health

# Acceder al frontend
curl -I http://localhost:3000
```

---

## Configurar HTTPS con Nginx + Let's Encrypt

### Instalar Nginx y Certbot

```bash
sudo apt install nginx certbot python3-certbot-nginx -y
```

### Configurar Nginx como reverse proxy

```nginx
# /etc/nginx/sites-available/backupsmc
server {
    listen 80;
    server_name app.backupsmc.smcsoluciones.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    location /api/ {
        proxy_pass http://localhost:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# Activar configuración
sudo ln -s /etc/nginx/sites-available/backupsmc /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Obtener certificado SSL
sudo certbot --nginx -d app.backupsmc.smcsoluciones.com
```

---

## Actualizar a una nueva versión

```bash
cd /path/to/backup-system

# Obtener cambios
git pull origin main

# Reconstruir y reiniciar
docker compose up -d --build

# Aplicar migraciones si hay cambios de base de datos
docker compose exec server alembic upgrade head

# Verificar
docker compose ps
```

---

## Instalar el agente en un nodo

El agente Go se instala en cada servidor que se desea respaldar.

### Linux (método rápido)

```bash
# Descargar e instalar
curl -fsSL https://app.backupsmc.com/install-agent.sh | bash

# O manualmente
wget https://github.com/smcsoluciones/backup-system/releases/latest/download/backupsmc-agent-linux-amd64
chmod +x backupsmc-agent-linux-amd64
sudo mv backupsmc-agent-linux-amd64 /usr/local/bin/backupsmc-agent
```

### Configurar como servicio systemd

```ini
# /etc/systemd/system/backupsmc-agent.service
[Unit]
Description=BackupSMC Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/backupsmc-agent \
  --server https://app.backupsmc.smcsoluciones.com \
  --token TU_TOKEN_DE_REGISTRO
Restart=always
RestartSec=10
User=root

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable backupsmc-agent
sudo systemctl start backupsmc-agent
sudo systemctl status backupsmc-agent
```

---

## Backups de la propia base de datos de BackupSMC

```bash
# Backup manual de PostgreSQL
docker compose exec db pg_dump -U backupsmc backupsmc > backup_$(date +%Y%m%d).sql

# Restaurar
cat backup_20260101.sql | docker compose exec -T db psql -U backupsmc -d backupsmc
```

---

## Monitoreo

### Logs

```bash
# Ver logs en tiempo real
docker compose logs -f

# Logs de un servicio específico
docker compose logs server -f --tail=100
docker compose logs worker -f --tail=100
```

### Estado de servicios

```bash
docker compose ps
docker stats
```

### Health checks

```bash
# API
curl http://localhost:8000/health

# Base de datos
docker compose exec db pg_isready -U backupsmc

# Redis
docker compose exec redis redis-cli ping
```

---

## Troubleshooting en producción

### Worker Celery no procesa jobs

```bash
# Verificar que Redis responde
docker compose exec redis redis-cli ping

# Revisar logs del worker
docker compose logs worker -f

# Reiniciar worker
docker compose restart worker
```

### Base de datos llena o corrupta

```bash
# Verificar espacio
df -h
docker system df

# Vacuum PostgreSQL
docker compose exec db psql -U backupsmc -c "VACUUM ANALYZE;"
```

### Agente no se conecta al servidor

```bash
# Verificar que el servidor es accesible desde el nodo
curl https://app.backupsmc.smcsoluciones.com/health

# Ver logs del agente
sudo journalctl -u backupsmc-agent -f

# Verificar token
backupsmc-agent --verify-token
```
