//go:build windows

package winsvc

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const ServiceName = "BackupSMC"
const ServiceDisplayName = "BackupSMC Backup Agent"
const ServiceDescription = "Enterprise backup agent — SMC Soluciones"

// Handler is the Windows service handler. Run() blocks until the SCM
// sends a Stop command, at which point it calls the provided stop function.
type Handler struct {
	Run  func() error
	Stop func()
}

func (h *Handler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}

	go func() { _ = h.Run() }()

	s <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for c := range r {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending}
			h.Stop()
			return false, 0
		}
	}
	return false, 0
}

// RunAsService starts the Windows service handler. Call this from the
// "service" subcommand when running under the SCM.
func RunAsService(h *Handler) error {
	return svc.Run(ServiceName, h)
}

// IsRunningAsService returns true when the process was launched by the SCM.
func IsRunningAsService() bool {
	ok, _ := svc.IsWindowsService()
	return ok
}

// Install registers the agent as a Windows service using sc.exe.
// exePath must be the full path to backupsmc-agent.exe.
// configPath must be the full path to agent.yaml.
func Install(exePath, configPath string) error {
	exePath, _ = filepath.Abs(exePath)
	configPath, _ = filepath.Abs(configPath)
	binPath := fmt.Sprintf(`"%s" service -c "%s"`, exePath, configPath)

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("winsvc: connect SCM: %w", err)
	}
	defer m.Disconnect()

	// Check if already installed
	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("winsvc: service %q already installed", ServiceName)
	}

	s, err = m.CreateService(ServiceName, binPath, mgr.Config{
		DisplayName:  ServiceDisplayName,
		Description:  ServiceDescription,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	})
	if err != nil {
		return fmt.Errorf("winsvc: create service: %w", err)
	}
	defer s.Close()

	return nil
}

// Uninstall removes the BackupSMC Windows service.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("winsvc: connect SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("winsvc: service not found: %w", err)
	}
	defer s.Close()

	return s.Delete()
}

// Start sends a start command to the SCM.
func Start() error {
	return runSC("start", ServiceName)
}

// Stop sends a stop command to the SCM.
func Stop() error {
	return runSC("stop", ServiceName)
}

func runSC(args ...string) error {
	out, err := exec.Command("sc.exe", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc.exe %v: %w — %s", args, err, string(out))
	}
	return nil
}
