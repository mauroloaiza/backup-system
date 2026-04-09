#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# publish-agent.sh — Compila el agente Linux y lo publica en el servidor
#
# Uso (desde el servidor dev):
#   bash scripts/publish-agent.sh
#
# Requisitos:
#   - Go instalado en el servidor dev
#   - SSH access al servidor de producción
#   - docker volume backupsmc_downloads_data montado en producción
# ─────────────────────────────────────────────────────────────────────────────
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AGENT_DIR="$REPO_ROOT/agent-linux"
VERSION=$(grep '^var version' "$AGENT_DIR/cmd/backupsmc-agent/main.go" | grep -oP '"[^"]+"' | tr -d '"')

# Producción
PROD_HOST="admin@ec2-52-1-71-173.compute-1.amazonaws.com"
PROD_KEY="/home/sysadmin/.ssh/prod.pem"
PROD_DOWNLOADS_PATH="/var/lib/docker/volumes/backupsmc_downloads_data/_data"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'; BOLD='\033[1m'

echo -e "\n${BOLD}BackupSMC Agent — Publicación v${VERSION}${RESET}\n"

# ── Build ─────────────────────────────────────────────────────────────────────
build() {
  local GOARCH=$1
  echo -e "  ${YELLOW}→${RESET}  Compilando linux/${GOARCH}..."
  CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH \
    go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o "$AGENT_DIR/bin/backupsmc-agent-linux-${GOARCH}" \
    "$AGENT_DIR/cmd/backupsmc-agent/"
  echo -e "  ${GREEN}✓${RESET}  linux/${GOARCH} → bin/backupsmc-agent-linux-${GOARCH}"
}

cd "$AGENT_DIR"
mkdir -p bin

build amd64
build arm64
# build arm   # descomentar si necesitas ARMv7

# ── Upload ────────────────────────────────────────────────────────────────────
echo ""
echo -e "  ${YELLOW}→${RESET}  Subiendo a producción..."

SSH_CMD="ssh -i $PROD_KEY -o StrictHostKeyChecking=no"

# Crear directorio si no existe
$SSH_CMD $PROD_HOST "sudo mkdir -p $PROD_DOWNLOADS_PATH && sudo chmod 755 $PROD_DOWNLOADS_PATH"

for ARCH in amd64 arm64; do
  BIN="backupsmc-agent-linux-${ARCH}"
  scp -i "$PROD_KEY" -o StrictHostKeyChecking=no \
    "$AGENT_DIR/bin/$BIN" \
    "$PROD_HOST:/tmp/$BIN"
  $SSH_CMD $PROD_HOST "sudo mv /tmp/$BIN $PROD_DOWNLOADS_PATH/$BIN && sudo chmod 644 $PROD_DOWNLOADS_PATH/$BIN"
  echo -e "  ${GREEN}✓${RESET}  $BIN publicado"
done

# ── Verify ────────────────────────────────────────────────────────────────────
echo ""
echo -e "  ${GREEN}✓${RESET}  Disponible en:"
echo -e "     https://backupsmc.smcsoluciones.com/downloads/backupsmc-agent-linux-amd64"
echo -e "     https://backupsmc.smcsoluciones.com/downloads/backupsmc-agent-linux-arm64"
echo ""
