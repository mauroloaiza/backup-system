package main

import (
	"os/exec"
	"syscall"
)

// hiddenCmd returns an *exec.Cmd that runs without a visible console window.
// On Windows, exec.Command shows a brief CMD flash unless CREATE_NO_WINDOW is set.
func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		HideWindow:    true,
	}
	return cmd
}
