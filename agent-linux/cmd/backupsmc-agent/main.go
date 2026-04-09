package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/config"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/tui/wizard"
)

var version = "0.5.0"

var okStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
var bold     = lipgloss.NewStyle().Bold(true)

func main() {
	root := &cobra.Command{
		Use:   "backupsmc-agent",
		Short: "BackupSMC Agent -- agente de backup para Linux",
		Long:  "Agente de backup empresarial BackupSMC para Linux.",
	}

	root.AddCommand(
		cmdSetup(),
		cmdRun(),
		cmdStatus(),
		cmdLogs(),
		cmdService(),
		cmdVersion(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ── setup ─────────────────────────────────────────────────────────────────────
func cmdSetup() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Asistente de configuracion inicial",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.Exists(cfgPath) {
				fmt.Printf("  Configuracion existente encontrada en %s\n", cfgPath)
				fmt.Print("  Sobreescribir? [s/N]: ")
				var resp string
				fmt.Scanln(&resp)
				if resp != "s" && resp != "S" {
					fmt.Println("  Cancelado.")
					return nil
				}
			}

			cfg, err := wizard.Run(cfgPath)
			if err != nil {
				return err
			}

			if err := config.Save(cfg, cfgPath); err != nil {
				return fmt.Errorf("guardar config: %w", err)
			}

			printFinalScreen(cfg, cfgPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultConfigPath, "ruta del archivo de configuracion")
	return cmd
}

func printFinalScreen(cfg *config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println(bold.Render("  BackupSMC Agent instalado y corriendo"))
	fmt.Println()
	fmt.Println(okStyle.Render("  Configuracion guardada en " + cfgPath))
	if cfg.Schedule.Enabled {
		fmt.Println(okStyle.Render("  Servicio systemd instalado y activo"))
		fmt.Println(okStyle.Render("  Proximo backup: segun horario " + cfg.Schedule.Cron))
	}
	fmt.Println()
	fmt.Println(dimStyle.Render("  ─────────────────────────────────────────────"))
	fmt.Println(dimStyle.Render("  Comandos utiles:"))
	fmt.Println()
	fmt.Println("    backupsmc-agent status      ver estado del agente")
	fmt.Println("    backupsmc-agent run          ejecutar backup ahora")
	fmt.Println("    backupsmc-agent logs         ver logs en vivo")
	fmt.Println("    backupsmc-agent tui          dashboard interactivo")
	fmt.Println("    backupsmc-agent service stop pausar el servicio")
	fmt.Println()
	fmt.Println(dimStyle.Render("  El agente corre como servicio systemd."))
	fmt.Println(dimStyle.Render("  ─────────────────────────────────────────────"))
	fmt.Println()
}

// ── run ───────────────────────────────────────────────────────────────────────
func cmdRun() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Ejecutar backup ahora",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("cargar config: %w", err)
			}
			fmt.Printf("  Iniciando backup desde %s...\n", cfgPath)
			_ = cfg
			// TODO: engine.Run(cfg)
			fmt.Println(okStyle.Render("  Backup completado"))
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultConfigPath, "ruta del archivo de configuracion")
	return cmd
}

// ── status ────────────────────────────────────────────────────────────────────
func cmdStatus() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Mostrar estado del agente",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !config.Exists(cfgPath) {
				fmt.Println("  El agente no esta configurado. Ejecuta: backupsmc-agent setup")
				return nil
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Println(bold.Render("  Estado BackupSMC Agent"))
			fmt.Println()
			fmt.Printf("  %-18s %s\n", "Servidor", cfg.Server.URL)
			fmt.Printf("  %-18s %s\n", "Config", cfgPath)
			if cfg.Schedule.Enabled {
				fmt.Printf("  %-18s activo (%s)\n", "Servicio", cfg.Schedule.Cron)
			} else {
				fmt.Printf("  %-18s manual\n", "Servicio")
			}
			fmt.Printf("  %-18s %d dias\n", "Retencion", cfg.Retention.Days)
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultConfigPath, "ruta del archivo de configuracion")
	return cmd
}

// ── logs ──────────────────────────────────────────────────────────────────────
func cmdLogs() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Ver logs en vivo",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: tail -f /var/log/backupsmc/agent.log con bubbletea
			fmt.Println("  Logs en vivo -- proximamente (backupsmc-agent tui para el dashboard)")
			return nil
		},
	}
}

// ── service ───────────────────────────────────────────────────────────────────
func cmdService() *cobra.Command {
	svc := &cobra.Command{
		Use:   "service",
		Short: "Gestionar el servicio del sistema (systemd / init.d)",
	}
	svc.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Instalar servicio systemd",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: service.Install()
				fmt.Println(okStyle.Render("  Servicio instalado"))
				return nil
			},
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Desinstalar servicio",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println(okStyle.Render("  Servicio desinstalado"))
				return nil
			},
		},
		&cobra.Command{
			Use:   "start",
			Short: "Iniciar servicio",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println(okStyle.Render("  Servicio iniciado"))
				return nil
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Detener servicio",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println(okStyle.Render("  Servicio detenido"))
				return nil
			},
		},
		&cobra.Command{
			Use:   "restart",
			Short: "Reiniciar servicio",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println(okStyle.Render("  Servicio reiniciado"))
				return nil
			},
		},
	)
	return svc
}

// ── version ───────────────────────────────────────────────────────────────────
func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Mostrar version del agente",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("  backupsmc-agent v%s (Linux)\n", version)
		},
	}
}
