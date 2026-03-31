module github.com/smcsoluciones/backup-system/agent

go 1.22

require (
	// HTTP client y servidor
	github.com/go-resty/resty/v2 v2.16.0

	// CLI flags
	github.com/spf13/cobra v1.8.1

	// Docker SDK
	github.com/docker/docker v27.3.1+incompatible

	// Logging estructurado
	go.uber.org/zap v1.27.0

	// Config
	github.com/spf13/viper v1.19.0
)
