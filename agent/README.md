# BackupSMC — Agent (Go)

Agente ligero escrito en Go. Se instala en cada servidor o VM a respaldar.
Compila a un único binario sin dependencias externas.

## Características

- Registro automático en el servidor BackupSMC
- Backup de MySQL y PostgreSQL (mysqldump / pg_dump)
- Backup de archivos y directorios (tar + compresión)
- Backup de volúmenes Docker
- Transferencia a múltiples destinos (Local, S3, SFTP, Google Drive)
- Cifrado AES-256-GCM
- Reporte de estado en tiempo real

## Estructura

```
agent/
├── cmd/agent/       # Entry point
├── internal/
│   ├── backup/      # Lógica de backup por fuente
│   ├── destination/ # Adaptadores de destino
│   └── config/      # Configuración
├── go.mod
├── go.sum
└── Dockerfile
```

## Compilación

```bash
go build -o bin/backupsmc-agent cmd/agent/main.go
```

## Uso

```bash
backupsmc-agent --server https://app.backupsmc.com --token <TOKEN>
```

Ver [DEVELOPMENT.md](../docs/DEVELOPMENT.md) para más detalles.
