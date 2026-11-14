//go:build !windows

package main

import "os/exec"

func showStartupError(message string) {
	// Diagnostic logging works without either optional desktop helper.
	if command, err := exec.LookPath("zenity"); err == nil {
		_ = exec.Command(command, "--error", "--title=Shaiya GX", "--text="+message).Run()
		return
	}
	if command, err := exec.LookPath("kdialog"); err == nil {
		_ = exec.Command(command, "--error", message, "--title", "Shaiya GX").Run()
	}
}
