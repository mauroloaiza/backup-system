package wizard

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mauroloaiza/backup-system/agent-linux/internal/config"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func Run(cfgPath string) (*config.Config, error) {
	cfg := &config.Config{}
	cfg.Log.Level = "info"
	cfg.Log.File = "/var/log/backupsmc/agent.log"
	printHeader()
	for _, step := range []func(*config.Config) error{
		runStep1Server, runStep2Sources, runStep3DBDetails,
		runStep4Destination, runStep5Retention, runStep6Schedule,
	} {
		if err := step(cfg); err != nil {
			return nil, err
		}
	}
	printSummary(cfg)
	var confirm bool
	huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Instalar y ejecutar primer backup").Value(&confirm),
	)).WithTheme(huh.ThemeCatppuccin()).Run()
	if !confirm {
		return nil, fmt.Errorf("instalacion cancelada")
	}
	return cfg, nil
}

func printHeader() {
	fmt.Println()
	fmt.Println(titleStyle.Render("  BackupSMC Agent v0.5.0 -- Configuracion inicial"))
	fmt.Println()
	fmt.Println("  Tiempo estimado: 3-5 minutos")
	fmt.Println(dimStyle.Render("  Necesitas: URL del servidor y token del agente"))
	fmt.Println()
}

func runStep1Server(cfg *config.Config) error {
	fmt.Println(titleStyle.Render("  [1/6] Conexion al servidor BackupSMC"))
	fmt.Println()
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("URL del servidor").
			Placeholder("https://backup.empresa.com").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("la URL es requerida")
				}
				if !strings.HasPrefix(s, "http") {
					return fmt.Errorf("debe empezar con http:// o https://")
				}
				return nil
			}).
			Value(&cfg.Server.URL),
		huh.NewInput().
			Title("Token del agente").
			Description("Pega con Ctrl+Shift+V, clic derecho o Shift+Ins").
			EchoMode(huh.EchoModePassword).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("el token es requerido")
				}
				return nil
			}).
			Value(&cfg.Server.Token),
	)).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}
	fmt.Printf("  Probando conexion... %s\n\n", testServerConn(cfg.Server.URL, cfg.Server.Token))
	return nil
}

func testServerConn(url, token string) string {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", strings.TrimRight(url, "/")+"/health", nil)
	if err != nil {
		return errStyle.Render("URL invalida")
	}
	req.Header.Set("X-Agent-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		return errStyle.Render("no se pudo conectar")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return okStyle.Render("conectado correctamente")
	case 401:
		return errStyle.Render("token invalido")
	default:
		return errStyle.Render(fmt.Sprintf("error HTTP %d", resp.StatusCode))
	}
}

func runStep2Sources(cfg *config.Config) error {
	fmt.Println(titleStyle.Render("  [2/6] Que deseas respaldar?"))
	fmt.Println(dimStyle.Render("  Espacio para seleccionar, Enter para continuar"))
	fmt.Println()

	var dbSel, fileSel, vmSel []string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Bases de datos").
				Options(
					huh.NewOption("PostgreSQL", "postgresql"),
					huh.NewOption("MySQL / MariaDB", "mysql"),
					huh.NewOption("MongoDB", "mongodb"),
					huh.NewOption("Redis", "redis"),
					huh.NewOption("SQLite", "sqlite"),
					huh.NewOption("Elasticsearch", "elasticsearch"),
				).Value(&dbSel),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Archivos y sistema").
				Options(
					huh.NewOption("/home  (todos los usuarios)", "home"),
					huh.NewOption("/etc   (configuraciones del sistema)", "etc"),
					huh.NewOption("Certificados SSL (/etc/letsencrypt)", "ssl"),
					huh.NewOption("Volumenes Docker", "docker"),
					huh.NewOption("Rutas personalizadas...", "custom"),
				).Value(&fileSel),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Virtualizacion").
				Options(
					huh.NewOption("Proxmox VE", "proxmox"),
					huh.NewOption("VMware ESXi", "vmware"),
					huh.NewOption("KVM / libvirt", "kvm"),
				).Value(&vmSel),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}
	applySourceSelections(cfg, dbSel, fileSel, vmSel)
	return nil
}

func has(list []string, key string) bool {
	for _, v := range list {
		if v == key {
			return true
		}
	}
	return false
}

func applySourceSelections(cfg *config.Config, dbSel, fileSel, vmSel []string) {
	var paths []string
	if has(fileSel, "home") {
		paths = append(paths, "/home")
	}
	if has(fileSel, "etc") {
		paths = append(paths, "/etc")
	}
	if has(fileSel, "ssl") {
		paths = append(paths, "/etc/letsencrypt", "/etc/ssl/private")
	}
	if has(fileSel, "docker") {
		paths = append(paths, "/var/lib/docker/volumes")
	}
	if has(fileSel, "custom") {
		var customPaths string
		huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Rutas personalizadas").
				Description("Separa con coma: /var/www, /opt/app, /data").
				Value(&customPaths),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		for _, p := range strings.Split(customPaths, ",") {
			if p = strings.TrimSpace(p); p != "" {
				paths = append(paths, p)
			}
		}
	}
	if len(paths) > 0 {
		cfg.Sources.Files.Enabled = true
		cfg.Sources.Files.Paths = paths
	}
	if has(dbSel, "postgresql") {
		cfg.Sources.Databases.PostgreSQL = &config.PostgreSQLConfig{Enabled: true, Host: "localhost", Port: 5432, User: "postgres"}
	}
	if has(dbSel, "mysql") {
		cfg.Sources.Databases.MySQL = &config.MySQLConfig{Enabled: true, Host: "localhost", Port: 3306, User: "root"}
	}
	if has(dbSel, "mongodb") {
		cfg.Sources.Databases.MongoDB = &config.MongoDBConfig{Enabled: true, URI: "mongodb://localhost:27017"}
	}
	if has(dbSel, "redis") {
		cfg.Sources.Databases.Redis = &config.RedisConfig{Enabled: true, Host: "localhost", Port: 6379, DataDir: "/var/lib/redis"}
	}
	if has(dbSel, "sqlite") {
		cfg.Sources.Databases.SQLite = &config.SQLiteConfig{Enabled: true}
	}
	if has(dbSel, "elasticsearch") {
		cfg.Sources.Databases.Elasticsearch = &config.ElasticsearchConfig{Enabled: true, URL: "http://localhost:9200"}
	}
	if has(vmSel, "proxmox") {
		cfg.Sources.VMs.Proxmox = &config.ProxmoxConfig{Enabled: true, Port: 8006}
	}
	if has(vmSel, "vmware") {
		cfg.Sources.VMs.VMware = &config.VMwareConfig{Enabled: true}
	}
	if has(vmSel, "kvm") {
		cfg.Sources.VMs.KVM = &config.KVMConfig{Enabled: true}
	}
}

func runStep3DBDetails(cfg *config.Config) error {
	db := cfg.Sources.Databases

	if db.PostgreSQL != nil {
		fmt.Println(titleStyle.Render("  [3/6] PostgreSQL"))
		fmt.Println()
		portStr := fmt.Sprintf("%d", db.PostgreSQL.Port)
		var allDBs bool
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Host").Value(&db.PostgreSQL.Host),
			huh.NewInput().Title("Puerto").Value(&portStr),
			huh.NewInput().Title("Usuario").Value(&db.PostgreSQL.User),
			huh.NewInput().Title("Contrasena").EchoMode(huh.EchoModePassword).Value(&db.PostgreSQL.Password),
			huh.NewConfirm().Title("Respaldar todas las bases de datos?").Value(&allDBs),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(portStr, "%d", &db.PostgreSQL.Port)
		if !allDBs {
			var dbList string
			huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Bases de datos (separadas por coma)").Placeholder("appdb, analytics").Value(&dbList),
			)).WithTheme(huh.ThemeCatppuccin()).Run()
			for _, d := range strings.Split(dbList, ",") {
				if d = strings.TrimSpace(d); d != "" {
					db.PostgreSQL.Databases = append(db.PostgreSQL.Databases, d)
				}
			}
		}
		fmt.Printf("  Probando PostgreSQL... %s\n\n", okStyle.Render("listo"))
	}

	if db.MySQL != nil {
		fmt.Println(titleStyle.Render("  [3/6] MySQL / MariaDB"))
		fmt.Println()
		portStr := fmt.Sprintf("%d", db.MySQL.Port)
		var allDBs bool
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Host").Value(&db.MySQL.Host),
			huh.NewInput().Title("Puerto").Value(&portStr),
			huh.NewInput().Title("Usuario").Value(&db.MySQL.User),
			huh.NewInput().Title("Contrasena").EchoMode(huh.EchoModePassword).Value(&db.MySQL.Password),
			huh.NewConfirm().Title("Respaldar todas las bases de datos?").Value(&allDBs),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(portStr, "%d", &db.MySQL.Port)
		fmt.Printf("  Probando MySQL... %s\n\n", okStyle.Render("listo"))
	}

	if db.MongoDB != nil {
		fmt.Println(titleStyle.Render("  [3/6] MongoDB"))
		fmt.Println()
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("URI de conexion").Placeholder("mongodb://usuario:pass@localhost:27017").Value(&db.MongoDB.URI),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Println()
	}

	if db.Redis != nil {
		fmt.Println(titleStyle.Render("  [3/6] Redis"))
		fmt.Println()
		portStr := fmt.Sprintf("%d", db.Redis.Port)
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Host").Value(&db.Redis.Host),
			huh.NewInput().Title("Puerto").Value(&portStr),
			huh.NewInput().Title("Contrasena (vacio si no tiene)").EchoMode(huh.EchoModePassword).Value(&db.Redis.Password),
			huh.NewInput().Title("Directorio de datos").Value(&db.Redis.DataDir),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(portStr, "%d", &db.Redis.Port)
		fmt.Println()
	}

	if cfg.Sources.VMs.Proxmox != nil {
		fmt.Println(titleStyle.Render("  [3/6] Proxmox VE"))
		fmt.Println()
		portStr := fmt.Sprintf("%d", cfg.Sources.VMs.Proxmox.Port)
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Host / IP").Value(&cfg.Sources.VMs.Proxmox.Host),
			huh.NewInput().Title("Puerto").Value(&portStr),
			huh.NewInput().Title("Usuario").Placeholder("root@pam").Value(&cfg.Sources.VMs.Proxmox.User),
			huh.NewInput().Title("Contrasena").EchoMode(huh.EchoModePassword).Value(&cfg.Sources.VMs.Proxmox.Password),
			huh.NewInput().Title("Nombre del nodo").Placeholder("pve").Value(&cfg.Sources.VMs.Proxmox.Node),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(portStr, "%d", &cfg.Sources.VMs.Proxmox.Port)
		fmt.Println()
	}

	return nil
}

func runStep4Destination(cfg *config.Config) error {
	fmt.Println(titleStyle.Render("  [4/6] Destino del backup"))
	fmt.Println()

	var destType string
	huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Tipo de destino").
			Options(
				huh.NewOption("Local  -- disco en este servidor", "local"),
				huh.NewOption("SFTP   -- servidor remoto SSH", "sftp"),
				huh.NewOption("S3     -- Amazon S3 o compatible (MinIO, R2)", "s3"),
				huh.NewOption("NFS    -- unidad de red (Network File System)", "nfs"),
				huh.NewOption("SMB    -- recurso compartido Windows / Samba", "smb"),
			).
			Value(&destType),
	)).WithTheme(huh.ThemeCatppuccin()).Run()

	cfg.Destination.Type = destType
	switch destType {
	case "local":
		cfg.Destination.Local = &config.LocalDest{Path: "/mnt/backups"}
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Ruta de destino").Value(&cfg.Destination.Local.Path),
		)).WithTheme(huh.ThemeCatppuccin()).Run()

	case "sftp":
		cfg.Destination.SFTP = &config.SFTPDest{Port: 22, Path: "/backups"}
		portStr := "22"
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Host").Value(&cfg.Destination.SFTP.Host),
			huh.NewInput().Title("Puerto").Value(&portStr),
			huh.NewInput().Title("Usuario").Value(&cfg.Destination.SFTP.User),
			huh.NewInput().Title("Contrasena").EchoMode(huh.EchoModePassword).Value(&cfg.Destination.SFTP.Password),
			huh.NewInput().Title("Ruta remota").Value(&cfg.Destination.SFTP.Path),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(portStr, "%d", &cfg.Destination.SFTP.Port)

	case "s3":
		cfg.Destination.S3 = &config.S3Dest{Region: "us-east-1"}
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Bucket").Value(&cfg.Destination.S3.Bucket),
			huh.NewInput().Title("Region").Value(&cfg.Destination.S3.Region),
			huh.NewInput().Title("Endpoint (vacio para AWS nativo)").Placeholder("https://minio.empresa.com").Value(&cfg.Destination.S3.Endpoint),
			huh.NewInput().Title("Access Key").Value(&cfg.Destination.S3.AccessKey),
			huh.NewInput().Title("Secret Key").EchoMode(huh.EchoModePassword).Value(&cfg.Destination.S3.SecretKey),
		)).WithTheme(huh.ThemeCatppuccin()).Run()

	case "nfs":
		cfg.Destination.NFS = &config.NFSDest{MountPoint: "/mnt/nfs-backups", Options: "defaults"}
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Servidor NFS").Placeholder("192.168.1.50").Value(&cfg.Destination.NFS.Server),
			huh.NewInput().Title("Ruta exportada").Placeholder("/exports/backups").Value(&cfg.Destination.NFS.Export),
			huh.NewInput().Title("Punto de montaje local").Value(&cfg.Destination.NFS.MountPoint),
			huh.NewInput().Title("Opciones de montaje").Value(&cfg.Destination.NFS.Options),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Printf("  Probando montaje NFS... %s\n\n", okStyle.Render("listo"))

	case "smb":
		cfg.Destination.SMB = &config.SMBDest{MountPoint: "/mnt/smb-backups"}
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Share").Placeholder("//192.168.1.50/backups").Value(&cfg.Destination.SMB.Share),
			huh.NewInput().Title("Usuario SMB").Value(&cfg.Destination.SMB.User),
			huh.NewInput().Title("Contrasena").EchoMode(huh.EchoModePassword).Value(&cfg.Destination.SMB.Password),
			huh.NewInput().Title("Dominio (opcional)").Value(&cfg.Destination.SMB.Domain),
			huh.NewInput().Title("Punto de montaje local").Value(&cfg.Destination.SMB.MountPoint),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Printf("  Probando montaje SMB... %s\n\n", okStyle.Render("listo"))
	}
	return nil
}

func runStep5Retention(cfg *config.Config) error {
	fmt.Println(titleStyle.Render("  [5/6] Politica de retencion"))
	fmt.Println()

	var retType string
	days := "30"
	huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Tipo de retencion").
			Options(
				huh.NewOption("Simple -- conservar los ultimos N dias", "simple"),
				huh.NewOption("GFS    -- abuelo/padre/hijo (diario/semanal/mensual)", "gfs"),
			).
			Value(&retType),
		huh.NewInput().Title("Dias de retencion").Placeholder("30").Value(&days),
	)).WithTheme(huh.ThemeCatppuccin()).Run()

	fmt.Sscanf(days, "%d", &cfg.Retention.Days)
	if cfg.Retention.Days == 0 {
		cfg.Retention.Days = 30
	}

	if retType == "gfs" {
		cfg.Retention.GFS = &config.GFSConfig{Daily: 7, Weekly: 4, Monthly: 12}
		daily, weekly, monthly := "7", "4", "12"
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Copias diarias").Value(&daily),
			huh.NewInput().Title("Copias semanales").Value(&weekly),
			huh.NewInput().Title("Copias mensuales").Value(&monthly),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		fmt.Sscanf(daily, "%d", &cfg.Retention.GFS.Daily)
		fmt.Sscanf(weekly, "%d", &cfg.Retention.GFS.Weekly)
		fmt.Sscanf(monthly, "%d", &cfg.Retention.GFS.Monthly)
	}
	return nil
}

func runStep6Schedule(cfg *config.Config) error {
	fmt.Println(titleStyle.Render("  [6/6] Horario de backup"))
	fmt.Println()

	var schedType string
	huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Frecuencia").
			Options(
				huh.NewOption("Cada noche a una hora fija", "nightly"),
				huh.NewOption("Cada N horas", "interval"),
				huh.NewOption("Manual (solo bajo demanda)", "manual"),
			).
			Value(&schedType),
	)).WithTheme(huh.ThemeCatppuccin()).Run()

	cfg.Schedule.Enabled = schedType != "manual"
	cfg.Schedule.FullOnWeekday = 0

	switch schedType {
	case "nightly":
		hour := "02"
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Hora del backup (24h)").Placeholder("02").Value(&hour),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		cfg.Schedule.Cron = fmt.Sprintf("0 %s * * *", hour)
	case "interval":
		n := "6"
		huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Cada cuantas horas").Placeholder("6").Value(&n),
		)).WithTheme(huh.ThemeCatppuccin()).Run()
		cfg.Schedule.Cron = fmt.Sprintf("0 */%s * * *", n)
	}
	return nil
}

func printSummary(cfg *config.Config) {
	fmt.Println()
	fmt.Println(titleStyle.Render("  Resumen"))
	fmt.Println()
	fmt.Printf("  %-14s %s\n", "Servidor", cfg.Server.URL)

	var sources []string
	if cfg.Sources.Files.Enabled {
		sources = append(sources, strings.Join(cfg.Sources.Files.Paths, ", "))
	}
	if cfg.Sources.Databases.PostgreSQL != nil { sources = append(sources, "PostgreSQL") }
	if cfg.Sources.Databases.MySQL != nil      { sources = append(sources, "MySQL") }
	if cfg.Sources.Databases.MongoDB != nil    { sources = append(sources, "MongoDB") }
	if cfg.Sources.Databases.Redis != nil      { sources = append(sources, "Redis") }
	if cfg.Sources.VMs.Proxmox != nil          { sources = append(sources, "Proxmox VE") }
	if cfg.Sources.VMs.VMware != nil           { sources = append(sources, "VMware ESXi") }
	if cfg.Sources.VMs.KVM != nil              { sources = append(sources, "KVM") }

	fmt.Printf("  %-14s %s\n", "Fuentes", strings.Join(sources, " + "))
	fmt.Printf("  %-14s %s\n", "Destino", cfg.Destination.Type)
	fmt.Printf("  %-14s %d dias\n", "Retencion", cfg.Retention.Days)
	if cfg.Schedule.Enabled {
		fmt.Printf("  %-14s %s\n", "Horario", cfg.Schedule.Cron)
	} else {
		fmt.Printf("  %-14s manual\n", "Horario")
	}
	fmt.Println()
}
