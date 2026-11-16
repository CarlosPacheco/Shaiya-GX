package world

import (
	"context"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"testing"
)

func TestRuntimeStopsAndReleasesOwnership(t *testing.T) {
	log, err := logx.New("runtime_test")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	for range 2 {
		ctx, cancel := context.WithCancel(context.Background())
		runtime := StartSweeps(ctx, log)
		cancel()
		runtime.Close()
		realm.runtimeMu.Lock()
		active := realm.runtime
		realm.runtimeMu.Unlock()
		if active != nil {
			t.Fatal("runtime ownership leaked")
		}
	}
}
