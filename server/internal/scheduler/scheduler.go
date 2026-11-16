// Package scheduler owns the cancellable world background timeline. All jobs
// execute serially; an overdue job runs once rather than replaying a tick storm.
package scheduler

import (
	"context"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/telemetry"
	"sync"
	"time"
)

type Job struct {
	Name     string
	Interval time.Duration
	Run      func(time.Time)
}
type task struct {
	Job
	next     time.Time
	duration *telemetry.Histogram
	late     *telemetry.Histogram
}
type Scheduler struct {
	mu    sync.Mutex
	tasks []task
	now   func() time.Time
}

func New(start time.Time, jobs []Job) *Scheduler {
	s := &Scheduler{now: time.Now}
	seen := map[string]bool{}
	for _, j := range jobs {
		if j.Interval <= 0 || j.Run == nil || seen[j.Name] {
			panic(fmt.Sprintf("invalid scheduler job %q", j.Name))
		}
		seen[j.Name] = true
		s.tasks = append(s.tasks, task{Job: j, next: start.Add(j.Interval), duration: telemetry.DurationMetric("gx_tick_" + j.Name + "_seconds"), late: telemetry.DurationMetric("gx_tick_" + j.Name + "_delay_seconds")})
	}
	return s
}

// Step also permits deterministic tests without sleeping or starting goroutines.
func (s *Scheduler) Step(ctx context.Context, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.tasks {
		if ctx.Err() != nil {
			return
		}
		t := &s.tasks[i]
		if now.Before(t.next) {
			continue
		}
		t.late.Observe(now.Sub(t.next))
		t.next = t.next.Add((now.Sub(t.next)/t.Interval + 1) * t.Interval)
		start := s.now()
		t.Run(now)
		t.duration.Observe(s.now().Sub(start))
	}
}
func (s *Scheduler) Run(ctx context.Context) {
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			s.Step(ctx, now)
		}
	}
}
