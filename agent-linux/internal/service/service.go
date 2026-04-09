package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	systemdUnit = "/etc/systemd/system/backupsmc-agent.service"
	initScript  = "/etc/init.d/backupsmc-agent"
)

// Install installs the agent as a system service (systemd or SysV).
func Install(binaryPath, configPath string) error {
	if isSystemd() {
		return installSystemd(binaryPath, configPath)
	}
	return installSysV(binaryPath, configPath)
}

// Uninstall removes the service.
func Uninstall() error {
	if isSystemd() {
		exec.Command("systemctl", "stop", "backupsmc-agent").Run()
		exec.Command("systemctl", "disable", "backupsmc-agent").Run()
		os.Remove(systemdUnit)
		exec.Command("systemctl", "daemon-reload").Run()
		return nil
	}
	exec.Command("service", "backupsmc-agent", "stop").Run()
	if _, err := exec.LookPath("update-rc.d"); err == nil {
		exec.Command("update-rc.d", "-f", "backupsmc-agent", "remove").Run()
	} else {
		exec.Command("chkconfig", "--del", "backupsmc-agent").Run()
	}
	return os.Remove(initScript)
}

// Start starts the service.
func Start() error { return runService("start") }

// Stop stops the service.
func Stop() error { return runService("stop") }

// Restart restarts the service.
func Restart() error { return runService("restart") }

// Status prints the service status.
func Status() error {
	var cmd *exec.Cmd
	if isSystemd() {
		cmd = exec.Command("systemctl", "status", "backupsmc-agent")
	} else {
		cmd = exec.Command("service", "backupsmc-agent", "status")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── systemd ───────────────────────────────────────────────────────────────────

func installSystemd(binaryPath, configPath string) error {
	unit := fmt.Sprintf(`[Unit]
Description=BackupSMC Linux Agent
After=network.target

[Service]
Type=simple
ExecStart=%s run --config %s
Restart=on-failure
RestartSec=30
User=root

[Install]
WantedBy=multi-user.target
`, binaryPath, configPath)

	if err := os.WriteFile(systemdUnit, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("escribir unit: %w", err)
	}
	if b, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %w\n%s", err, b)
	}
	if b, err := exec.Command("systemctl", "enable", "--now", "backupsmc-agent").CombinedOutput(); err != nil {
		return fmt.Errorf("enable: %w\n%s", err, b)
	}
	return nil
}

// ── SysV init.d ───────────────────────────────────────────────────────────────

func installSysV(binaryPath, configPath string) error {
	script := fmt.Sprintf(`#!/bin/sh
### BEGIN INIT INFO
# Provides:          backupsmc-agent
# Required-Start:    $remote_fs $syslog $network
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: BackupSMC Linux Agent
### END INIT INFO

DAEMON=%s
ARGS="run --config %s"
PIDFILE=/var/run/backupsmc-agent.pid
NAME=backupsmc-agent

case "$1" in
  start)
    echo "Starting $NAME"
    start-stop-daemon --start --background --make-pidfile --pidfile $PIDFILE \
      --exec $DAEMON -- $ARGS
    ;;
  stop)
    echo "Stopping $NAME"
    start-stop-daemon --stop --pidfile $PIDFILE
    ;;
  restart)
    $0 stop
    $0 start
    ;;
  status)
    if [ -f $PIDFILE ] && kill -0 $(cat $PIDFILE) 2>/dev/null; then
      echo "$NAME is running"
    else
      echo "$NAME is not running"
    fi
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|status}" >&2
    exit 1
    ;;
esac
exit 0
`, binaryPath, configPath)

	if err := os.WriteFile(initScript, []byte(script), 0o755); err != nil {
		return fmt.Errorf("escribir init script: %w", err)
	}
	if _, err := exec.LookPath("update-rc.d"); err == nil {
		exec.Command("update-rc.d", "backupsmc-agent", "defaults").Run()
	} else {
		exec.Command("chkconfig", "--add", "backupsmc-agent").Run()
	}
	if b, err := exec.Command("service", "backupsmc-agent", "start").CombinedOutput(); err != nil {
		return fmt.Errorf("start: %w\n%s", err, b)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	out, err := exec.Command("systemctl", "is-system-running").Output()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	return s == "running" || s == "degraded"
}

func runService(action string) error {
	var cmd *exec.Cmd
	if isSystemd() {
		cmd = exec.Command("systemctl", action, "backupsmc-agent")
	} else {
		cmd = exec.Command("service", "backupsmc-agent", action)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
