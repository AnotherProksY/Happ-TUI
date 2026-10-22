package utils

import (
	"os/exec"
	"syscall"
)

// SpawnCore starts the back-end detached from the terminal.
func SpawnCore() error {
	cmdPath, err := GetBinPath(AppName + "-core")
	if err != nil {
		return err
	}

	cmd := exec.Command(cmdPath)

	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	cmd.SysProcAttr = detachedSysProcAttr()

	return cmd.Start()
}

func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
