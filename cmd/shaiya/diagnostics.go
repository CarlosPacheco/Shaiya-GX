package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/Spelunkern/Shaiya-GX/internal/clientlog"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

type clientDiagnostics struct {
	file        *os.File
	path        string
	previous    io.Writer
	crashOutput bool
}

func startDiagnostics() *clientDiagnostics {
	var directories []string
	if exe, err := os.Executable(); err == nil {
		directories = append(directories, filepath.Join(filepath.Dir(exe), "logs"))
	}
	if cache, err := os.UserCacheDir(); err == nil {
		directories = append(directories, filepath.Join(cache, "Shaiya GX", "logs"))
	}
	directories = append(directories, filepath.Join(os.TempDir(), "shaiya-gx-logs"))
	d := openDiagnostics(directories)
	d.logStartup()
	if d.file != nil {
		if err := debug.SetCrashOutput(d.file, debug.CrashOptions{}); err != nil {
			log.Printf("Unable to attach crash diagnostics: %v", err)
		} else {
			d.crashOutput = true
		}
	}
	return d
}

func (d *clientDiagnostics) logStartup() {
	revision, modified := "unknown", "unknown"
	if build, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value
			}
		}
	}
	log.Printf("Shaiya GX startup platform=%s/%s revision=%s modified=%s", runtime.GOOS, runtime.GOARCH, revision, modified)
	ram := "unknown"
	if bytes, err := platform.TotalMemoryBytes(); err == nil {
		ram = fmt.Sprintf("%.0f MiB", float64(bytes)/(1<<20))
	}
	log.Printf("System OS=%q CPU=%q logical_cpus=%d physical_ram=%q",
		platform.OSVersion(), platform.CPUModel(), runtime.NumCPU(), ram)
}

func openDiagnostics(directories []string) *clientDiagnostics {
	d := &clientDiagnostics{previous: log.Writer()}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0700); err != nil {
			continue
		}
		path := filepath.Join(directory, "client.log")
		// Keep the preceding session while bounding history to two logs.
		if _, err := os.Stat(path); err == nil {
			previous := filepath.Join(directory, "client.previous.log")
			if err := os.Remove(previous); err != nil && !os.IsNotExist(err) {
				continue
			}
			if err := os.Rename(path, previous); err != nil {
				continue
			}
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			continue
		}
		d.file, d.path = f, path
		log.SetOutput(io.MultiWriter(f, d.previous))
		break
	}
	return d
}
func (d *clientDiagnostics) Close() {
	clientlog.FlushWarnings()
	log.SetOutput(d.previous)
	if d.crashOutput {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
	}
	if d.file != nil {
		_ = d.file.Close()
	}
}
func (d *clientDiagnostics) data(path string) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	log.Printf("Data directory: %s", absolute)
}
func (d *clientDiagnostics) fail(err error, interactive bool, stack []byte) {
	log.Printf("ERROR: %v", err)
	if len(stack) > 0 {
		log.Print(string(stack))
	}
	message := fmt.Sprintf("Shaiya GX could not start or continue.\n\n%v", err)
	if d.file != nil {
		_ = d.file.Sync()
		message += "\n\nDiagnostic log: " + d.path
	} else {
		message += "\n\nUnable to write a diagnostic log. Start from a terminal to see the error."
	}
	if interactive {
		showStartupError(message)
	}
}
