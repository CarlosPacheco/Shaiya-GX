package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/app"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Allow one pending GPU transfer and one PNG writer to bound screenshot memory.
type gameScreenshots struct {
	readback   render.FramebufferReadback
	capturedAt time.Time
	writing    <-chan screenshotResult
}

type screenshotResult struct {
	path string
	err  error
}

func (r screenshotResult) report(a *app.App) {
	if r.err != nil {
		log.Printf("Screenshot: %v", r.err)
		if a != nil {
			a.ReportScreenshot("")
		}
	} else {
		log.Printf("Screenshot saved to %s", r.path)
		if a != nil {
			a.ReportScreenshot(filepath.Base(r.path))
		}
	}
}

func (s *gameScreenshots) poll(a *app.App) {
	if s.writing != nil {
		select {
		case result := <-s.writing:
			s.writing = nil
			result.report(a)
		default:
			return
		}
	}
	frame, err := s.readback.Poll()
	if err != nil {
		screenshotResult{err: err}.report(a)
		return
	}
	if frame != nil {
		at := s.capturedAt
		result := make(chan screenshotResult, 1)
		s.writing = result
		go func() {
			path, err := saveGameScreenshot(frame, at)
			result <- screenshotResult{path: path, err: err}
		}()
	}
}

func (s *gameScreenshots) close() {
	defer s.readback.Close()
	if s.writing != nil {
		(<-s.writing).report(nil)
		s.writing = nil
	}
	if s.readback.Pending() {
		// Finish a pending request on exit while the context is still live.
		gl.Finish()
		s.poll(nil)
		if s.writing != nil {
			(<-s.writing).report(nil)
		}
	}
}

func saveGameScreenshot(frame *render.CapturedFrame, capturedAt time.Time) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(exe), "screenshot")
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, capturedAt.Format("2006-01-02_15-04-05")+"-*.png")
	if err != nil {
		return "", err
	}
	path := file.Name()
	err = frame.WritePNG(file)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}
