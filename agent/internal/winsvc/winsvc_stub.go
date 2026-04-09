//go:build !windows

package winsvc

import "errors"

const ServiceName = "BackupSMC"
const ServiceDisplayName = "BackupSMC Backup Agent"
const ServiceDescription = "Enterprise backup agent — SMC Soluciones"

var errNotSupported = errors.New("winsvc: Windows services are only supported on Windows")

type Handler struct {
	Run  func() error
	Stop func()
}

func RunAsService(_ *Handler) error     { return errNotSupported }
func IsRunningAsService() bool          { return false }
func Install(_, _ string) error         { return errNotSupported }
func Uninstall() error                  { return errNotSupported }
func Start() error                      { return errNotSupported }
func Stop() error                       { return errNotSupported }
