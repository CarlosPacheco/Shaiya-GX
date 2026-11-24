package clientlog

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"
)

func TestWarningWindow(t *testing.T) {
	t.Cleanup(FlushWarnings)
	now := time.Now()
	format := "asset %s unavailable"
	if emit, _ := admitWarning(format, now); !emit {
		t.Fatal("first warning was suppressed")
	}
	if emit, _ := admitWarning(format, now.Add(time.Second)); emit {
		t.Fatal("repeated warning was emitted")
	}
	if emit, suppressed := admitWarning(format, now.Add(time.Minute)); !emit || suppressed != 1 {
		t.Fatalf("next window emitted=%t suppressed=%d", emit, suppressed)
	}
}

func TestDefaultAndVerboseOutput(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() {
		SetVerbose(false)
		FlushWarnings()
		log.SetOutput(previous)
	})
	Debugf("shader %s", "example")
	Warnf("texture %s unavailable", "one")
	Warnf("texture %s unavailable", "two")
	FlushWarnings()
	if got := output.String(); !strings.Contains(got, "WARN texture one") ||
		!strings.Contains(got, "1 additional recurring messages suppressed") ||
		strings.Contains(got, "two") || strings.Contains(got, "shader") {
		t.Fatalf("unexpected default output: %s", got)
	}

	output.Reset()
	SetVerbose(true)
	Debugf("shader %s", "example")
	Warnf("texture %s unavailable", "one")
	Warnf("texture %s unavailable", "two")
	if got := output.String(); !strings.Contains(got, "DEBUG shader example") ||
		!strings.Contains(got, "WARN texture one") || !strings.Contains(got, "WARN texture two") {
		t.Fatalf("unexpected verbose output: %s", got)
	}
}
