package controlplane

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthenticatedScopedShutdownAndMetrics(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := Start(ctx, "test_service", "", cancel)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + s.Address() + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.StatusCode)
	}
	response, err = client.Post("http://"+s.Address()+"/shutdown", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 || ctx.Err() != nil {
		t.Fatal("unauthenticated shutdown")
	}
	exe, _ := os.Executable()
	if err := RequestStop(".", "test_service", os.Getpid()+1, exe); err == nil {
		t.Fatal("wrong process accepted")
	}
	if err := RequestStop(".", "test_service", os.Getpid(), exe); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel")
	}
	s.Close()
	if _, err := os.Stat(filepath.Join("runtime/control/test_service.json")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
func TestRefusesPublicDiagnostics(t *testing.T) {
	if _, err := Start(context.Background(), "test", "0.0.0.0:0", func() {}); err == nil {
		t.Fatal("public diagnostics accepted")
	}
}
