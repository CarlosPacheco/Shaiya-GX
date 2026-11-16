package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestDeterministicTimelineSkipsMissedTicksAndCancels(t *testing.T) {
	start := time.Unix(100, 0)
	var got []string
	ctx, cancel := context.WithCancel(context.Background())
	s := New(start, []Job{{Name: "test_fast", Interval: time.Second, Run: func(time.Time) { got = append(got, "fast") }}, {Name: "test_slow", Interval: 2 * time.Second, Run: func(time.Time) { got = append(got, "slow") }}})
	s.Step(ctx, start.Add(time.Second))
	s.Step(ctx, start.Add(10*time.Second))
	s.Step(ctx, start.Add(10*time.Second))
	if !reflect.DeepEqual(got, []string{"fast", "fast", "slow"}) {
		t.Fatal(got)
	}
	cancel()
	s.Step(ctx, start.Add(20*time.Second))
	if len(got) != 3 {
		t.Fatal("ran after cancellation")
	}
}
func TestRunStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := New(time.Now(), nil)
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler leaked")
	}
}
