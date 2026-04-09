#!/bin/sh
# BackupSMC Linux Agent Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/mauroloaiza/backup-system/main/agent-linux/install.sh | sh
set -e

VERSION="0.5.0"
BINARY_NAME="backupsmc-agent"
INSTALL_DIR="/usr/local/bin"
RELEASES_URL="https://github.com/mauroloaiza/backup-system/releases/download/v${VERSION}"

# ── colors ────────────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; RESET='\033[0m'; BOLD='\033[1m'
else
  GREEN=''; YELLOW=''; RED=''; RESET=''; BOLD=''
fi

ok()   { printf "${GREEN}  ✓${RESET}  %s\n" "$1"; }
warn() { printf "${YELLOW}  !${RESET}  %s\n" "$1"; }
err()  { printf "${RED}  ✗${RESET}  %s\n" "$1"; exit 1; }
step() { printf "${BOLD}  →${RESET}  %s\n" "$1"; }

# ── root check ────────────────────────────────────────────────────────────────
if [ "$(id -u)" -ne 0 ]; then
  err "Ejecuta como root: sudo sh install.sh"
fi

printf "\n${BOLD}  BackupSMC Agent v${VERSION} — Instalador${RESET}\n\n"

# ── OS detection ──────────────────────────────────────────────────────────────
OS_ID=""
VERSION_ID=""
if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS_ID="$ID"
  VERSION_ID="${VERSION_ID:-0}"
fi

detect_os() {
  case "$OS_ID" in
    debian)
      VERSION_MAJOR="${VERSION_ID%%.*}"
      if [ "$VERSION_MAJOR" -ge 9 ]; then
        ok "Detectado: Debian $VERSION_ID"
      else
        err "Debian $VERSION_ID no soportado. Minimo: Debian 9"
      fi
      ;;
    ubuntu)
      case "$VERSION_ID" in
        18.04|20.04|22.04|24.04) ok "Detectado: Ubuntu $VERSION_ID" ;;
        *) err "Ubuntu $VERSION_ID no soportado. Soportados: 18.04, 20.04, 22.04, 24.04" ;;
      esac
      ;;
    rhel|centos|almalinux|rocky)
      VERSION_MAJOR="${VERSION_ID%%.*}"
      if [ "$VERSION_MAJOR" -ge 6 ]; then
        ok "Detectado: $ID $VERSION_ID"
      else
        err "$ID $VERSION_ID no soportado. Minimo: RHEL/CentOS 6"
      fi
      ;;
    amzn)
      if [ "$VERSION_ID" = "2" ] || [ "$VERSION_ID" = "2023" ]; then
        ok "Detectado: Amazon Linux $VERSION_ID"
      else
        err "Amazon Linux $VERSION_ID no soportado. Soportados: 2, 2023"
      fi
      ;;
    sles|opensuse-leap|opensuse-tumbleweed)
      VERSION_MAJOR="${VERSION_ID%%.*}"
      if [ "$VERSION_MAJOR" -ge 12 ]; then
        ok "Detectado: $ID $VERSION_ID"
      else
        err "$ID $VERSION_ID no soportado. Minimo: SUSE 12"
      fi
      ;;
    fedora)
      VERSION_MAJOR="${VERSION_ID%%.*}"
      if [ "$VERSION_MAJOR" -ge 32 ]; then
        ok "Detectado: Fedora $VERSION_ID"
      else
        err "Fedora $VERSION_ID no soportado. Minimo: Fedora 32"
      fi
      ;;
    *)
      warn "Sistema no reconocido ($OS_ID). Intentando instalacion generica."
      ;;
  esac
}

detect_os

# ── architecture ──────────────────────────────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH_SUFFIX="amd64" ;;
  aarch64) ARCH_SUFFIX="arm64" ;;
  armv7l)  ARCH_SUFFIX="arm"   ;;
  *)       err "Arquitectura $ARCH no soportada" ;;
esac

BINARY_FILE="${BINARY_NAME}-linux-${ARCH_SUFFIX}"
DOWNLOAD_URL="${RELEASES_URL}/${BINARY_FILE}"

# ── download ──────────────────────────────────────────────────────────────────
step "Descargando ${BINARY_FILE}..."

TMP_FILE=$(mktemp /tmp/backupsmc_XXXXXX)
trap 'rm -f "$TMP_FILE"' EXIT

if command -v curl > /dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
elif command -v wget > /dev/null 2>&1; then
  wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
else
  err "Se requiere curl o wget"
fi

ok "Descargado"

# ── install ───────────────────────────────────────────────────────────────────
step "Instalando en ${INSTALL_DIR}/${BINARY_NAME}..."
chmod +x "$TMP_FILE"
mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
ok "Instalado"

# ── verify ────────────────────────────────────────────────────────────────────
if command -v backupsmc-agent > /dev/null 2>&1; then
  ok "Verificado: $(backupsmc-agent version 2>/dev/null || echo 'ok')"
fi

# ── done ──────────────────────────────────────────────────────────────────────
printf "\n${BOLD}  Listo!${RESET}\n\n"
printf "  Para configurar el agente ejecuta:\n\n"
printf "    ${BOLD}backupsmc-agent setup${RESET}\n\n"
printf "  Sistemas soportados:\n"
printf "    Debian 9+, Ubuntu 18.04+, RHEL/CentOS 6+, AlmaLinux 8+,\n"
printf "    Rocky Linux 8+, Amazon Linux 2/2023, SUSE 12+, Fedora 32+\n\n"
