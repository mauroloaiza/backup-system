#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# deploy.sh — Build, push y deploy de BackupSMC al servidor de producción
#
# Uso (desde el servidor dev):
#   bash scripts/deploy.sh
# ─────────────────────────────────────────────────────────────────────────────
set -e

PROD_HOST="admin@ec2-52-1-71-173.compute-1.amazonaws.com"
PROD_KEY="/home/sysadmin/.ssh/prod.pem"
PROD_DIR="/opt/backupsmc"
DOCKER_USER="smcsoluciones"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'; BOLD='\033[1m'
SSH="ssh -i $PROD_KEY -o StrictHostKeyChecking=no"

echo -e "\n${BOLD}BackupSMC — Deploy a producción${RESET}\n"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ── Build imágenes ────────────────────────────────────────────────────────────
echo -e "  ${YELLOW}→${RESET}  Construyendo imágenes..."

docker build -f server/Dockerfile    -t $DOCKER_USER/backupsmc-server:latest   ./server
docker build -f frontend/Dockerfile  -t $DOCKER_USER/backupsmc-frontend:latest ./frontend

echo -e "  ${GREEN}✓${RESET}  Imágenes construidas"

# ── Push a Docker Hub ─────────────────────────────────────────────────────────
echo -e "  ${YELLOW}→${RESET}  Pusheando a Docker Hub..."

docker push $DOCKER_USER/backupsmc-server:latest
docker push $DOCKER_USER/backupsmc-frontend:latest

echo -e "  ${GREEN}✓${RESET}  Push completado"

# ── Preparar servidor ─────────────────────────────────────────────────────────
echo -e "  ${YELLOW}→${RESET}  Preparando servidor de producción..."

$SSH $PROD_HOST "sudo mkdir -p $PROD_DIR && sudo chown admin:admin $PROD_DIR"

# Copiar docker-compose y env
scp -i "$PROD_KEY" -o StrictHostKeyChecking=no \
  docker-compose.yml "$PROD_HOST:$PROD_DIR/docker-compose.yml"

# Copiar .env si existe localmente y no existe en producción
if [ -f .env ]; then
  $SSH $PROD_HOST "[ -f $PROD_DIR/.env ]" || \
    scp -i "$PROD_KEY" -o StrictHostKeyChecking=no .env "$PROD_HOST:$PROD_DIR/.env"
fi

echo -e "  ${GREEN}✓${RESET}  Archivos copiados"

# ── Deploy ────────────────────────────────────────────────────────────────────
echo -e "  ${YELLOW}→${RESET}  Levantando contenedores..."

$SSH $PROD_HOST "cd $PROD_DIR && \
  sudo docker compose pull && \
  sudo docker compose up -d --remove-orphans"

echo -e "  ${GREEN}✓${RESET}  Contenedores activos"

# ── Verify ────────────────────────────────────────────────────────────────────
sleep 3
STATUS=$($SSH $PROD_HOST "sudo docker compose -f $PROD_DIR/docker-compose.yml ps --format json 2>/dev/null | grep -c '\"running\"'" || echo "?")

echo ""
echo -e "  ${GREEN}✓${RESET}  Deploy completado"
echo -e "     Frontend:  http://$PROD_HOST:3100"
echo -e "     (configura Nginx Proxy Manager → backupsmc.smcsoluciones.com → puerto 3100)"
echo ""
