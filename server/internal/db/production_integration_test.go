//go:build integration

package db

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Separate OS processes exercise the same driver/DSN used by login and world.
// The fixture and every subprocess use a disposable database, never runtime/.
func TestSQLiteConcurrentProcessesAndCrashRecovery(t *testing.T) {
	p := sqliteFixture(t)
	for _, id := range []int{997501, 997502} {
		if _, err := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',100000)`, id, fmt.Sprintf("sqlite_process_%d", id)); err != nil {
			t.Fatal(err)
		}
	}
	path := os.Getenv("RAPTURE_DB_PATH")
	workerContext, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := func(mode, report string) *exec.Cmd {
		c := exec.CommandContext(workerContext, os.Args[0], "-test.run=^TestSQLiteProcessWorker$")
		// A killed helper cannot run TestMain cleanup. Its own temporary root
		// therefore lives under a directory owned and cleaned by this parent.
		workerTemp := t.TempDir()
		c.Env = append(os.Environ(), "GX_SQLITE_WORKER_MODE="+mode, "GX_SQLITE_WORKER_PATH="+path, "GX_SQLITE_WORKER_REPORT="+report,
			"TMPDIR="+workerTemp, "TMP="+workerTemp, "TEMP="+workerTemp)
		return c
	}
	const workers = 4
	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, workers)
	reports := make([]string, workers)
	start := time.Now()
	for i := range workers {
		reports[i] = filepath.Join(t.TempDir(), "latencies.json")
		cmd := command("load", reports[i])
		go func() { out, err := cmd.CombinedOutput(); done <- result{out, err} }()
	}
	for range workers {
		r := <-done
		if r.err != nil {
			t.Fatalf("worker: %v\n%s", r.err, r.output)
		}
	}
	elapsed := time.Since(start)
	var timings []int64
	for _, report := range reports {
		var worker []int64
		raw, err := os.ReadFile(report)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &worker); err != nil {
			t.Fatal(err)
		}
		timings = append(timings, worker...)
	}
	sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
	if len(timings) != 800 {
		t.Fatalf("committed %d transfers, want 800", len(timings))
	}
	var a, b int
	check := func(wantA, wantB int) {
		t.Helper()
		if err := p.QueryRow("SELECT Point FROM PS_UserData_Users_Master WHERE UserUID=997501").Scan(&a); err != nil {
			t.Fatal(err)
		}
		if err := p.QueryRow("SELECT Point FROM PS_UserData_Users_Master WHERE UserUID=997502").Scan(&b); err != nil {
			t.Fatal(err)
		}
		if a != wantA || b != wantB {
			t.Fatalf("balances %d/%d, want %d/%d", a, b, wantA, wantB)
		}
	}
	check(99200, 100800)
	t.Logf("4 processes, %d committed two-row transfers in %s (%.0f tx/s), transaction p50=%s p95=%s max=%s; no lost balance or busy errors", len(timings), elapsed, float64(len(timings))/elapsed.Seconds(), time.Duration(timings[len(timings)/2]), time.Duration(timings[len(timings)*95/100]), time.Duration(timings[len(timings)-1]))
	for _, mode := range []string{"uncommitted", "committed"} {
		cmd := command(mode, "")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		ready := make(chan bool, 1)
		go func() {
			s := bufio.NewScanner(stdout)
			for s.Scan() {
				if s.Text() == "READY" {
					ready <- true
					return
				}
			}
			ready <- false
		}()
		select {
		case ok := <-ready:
			if !ok {
				cmd.Wait()
				t.Fatal("crash worker exited before reaching transaction boundary")
			}
		case <-time.After(20 * time.Second):
			cmd.Process.Kill()
			cmd.Wait()
			t.Fatal("crash worker readiness timeout")
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		cmd.Wait() // Forced process termination is the behavior under test.
		if mode == "uncommitted" {
			check(99200, 100800)
		} else {
			check(99199, 100801)
		}
		t.Logf("forced process termination after %s write: expected balances preserved", mode)
	}
	var integrity string
	if err := p.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity=%q: %v", integrity, err)
	}
}

// Invoked only by TestSQLiteConcurrentProcessesAndCrashRecovery's subprocesses.
func TestSQLiteProcessWorker(t *testing.T) {
	mode := os.Getenv("GX_SQLITE_WORKER_MODE")
	if mode == "" {
		t.Skip("subprocess helper")
	}
	path := os.Getenv("GX_SQLITE_WORKER_PATH")
	if path == "" {
		t.Fatal("missing disposable worker path")
	}
	t.Setenv("RAPTURE_DB_PATH", path)
	p, err := Pool(GameData)
	if err != nil {
		t.Fatal(err)
	}
	var timings []int64
	count := 200
	if mode != "load" {
		count = 1
	}
	for range count {
		start := time.Now()
		tx, err := p.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec("UPDATE PS_UserData_Users_Master SET Point=Point-1 WHERE UserUID=997501"); err == nil {
			_, err = tx.Exec("UPDATE PS_UserData_Users_Master SET Point=Point+1 WHERE UserUID=997502")
		}
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if mode == "uncommitted" {
			fmt.Println("READY")
			time.Sleep(time.Hour)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if mode == "committed" {
			fmt.Println("READY")
			time.Sleep(time.Hour)
		}
		timings = append(timings, time.Since(start).Nanoseconds())
	}
	if os.Getenv("GX_SQLITE_WORKER_REPORT") == "" {
		t.Fatal("missing worker report")
	}
	raw, _ := json.Marshal(timings)
	if err = os.WriteFile(os.Getenv("GX_SQLITE_WORKER_REPORT"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
