package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
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
	if d.file != nil {
		if err := debug.SetCrashOutput(d.file, debug.CrashOptions{}); err != nil {
			log.Printf("Unable to attach crash diagnostics: %v", err)
		} else {
			d.crashOutput = true
		}
	}
	log.Printf("Shaiya GX startup; platform=%s/%s; diagnostic=%s", runtime.GOOS, runtime.GOARCH, d.path)
	if build, ok := debug.ReadBuildInfo(); ok {
		for _, s := range build.Settings {
			if s.Key == "vcs.revision" || s.Key == "vcs.modified" {
				log.Printf("%s=%s", s.Key, s.Value)
			}
		}
	}
	return d
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
func (d *clientDiagnostics) fail(err error, interactive bool) {
	log.Printf("ERROR: %v\n%s", err, debug.Stack())
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
