// Package clientlog controls client diagnostics without changing server logging.
package clientlog

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var verbose atomic.Bool

func SetVerbose(enabled bool) { verbose.Store(enabled) }

func Debugf(format string, args ...any) {
	if verbose.Load() {
		log.Printf("DEBUG "+format, args...)
	}
}

type warning struct {
	last       time.Time
	suppressed int
}

var warnings = struct {
	sync.Mutex
	byFormat map[string]warning
}{byFormat: make(map[string]warning)}

// Warnf limits recurring failures by message template. Callers use a fixed
// format so changing asset names, object IDs or error details cannot flood the log.
func Warnf(format string, args ...any) {
	if !verbose.Load() {
		emit, suppressed := admitWarning(format, time.Now())
		if !emit {
			return
		}
		if suppressed > 0 {
			log.Printf("WARN %d similar messages suppressed for %q", suppressed, format)
		}
	}
	log.Printf("WARN "+format, args...)
}

func admitWarning(format string, now time.Time) (bool, int) {
	warnings.Lock()
	defer warnings.Unlock()
	entry := warnings.byFormat[format]
	if now.Sub(entry.last) < time.Minute {
		entry.suppressed++
		warnings.byFormat[format] = entry
		return false, 0
	}
	warnings.byFormat[format] = warning{last: now}
	return true, entry.suppressed
}

func FlushWarnings() {
	warnings.Lock()
	total := 0
	for _, entry := range warnings.byFormat {
		total += entry.suppressed
	}
	clear(warnings.byFormat)
	warnings.Unlock()
	if total > 0 {
		log.Printf("WARN %d additional recurring messages suppressed", total)
	}
}
