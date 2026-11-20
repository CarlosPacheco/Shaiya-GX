// Package db owns the portable SQLite store. Original EP table/column names
// are retained; database names become table prefixes in the single file.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/filelock"
	"github.com/Spelunkern/Shaiya-GX/server/internal/telemetry"
)

const (
	UserData = "PS_UserData"
	GameData = "PS_GameData"
	GameDefs = "PS_GameDefs"
	Billing  = "PS_Billing"
)
const Timeout = 10 * time.Second

var (
	mu    sync.RWMutex
	pools = map[string]*sql.DB{}
)

// Initialize copies the versioned seed once. A process lock and atomic rename
// prevent login/world from observing a partially copied database at startup.
func Initialize() error {
	path := config.DatabasePath()
	exists, err := runtimeExists(path)
	if err != nil || exists {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	c, cancel := ctx()
	defer cancel()
	release, err := filelock.Acquire(c, path+".init.lock")
	if err != nil {
		return fmt.Errorf("database initialization lock: %w", err)
	}
	defer release()

	exists, err = runtimeExists(path)
	if err != nil || exists {
		return err
	}
	seed, err := os.Open(config.DatabaseSeed())
	if err != nil {
		return fmt.Errorf("open SQLite seed: %w", err)
	}
	defer seed.Close()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".rapture-init-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, seed); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func runtimeExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if seed, err := os.Stat(config.DatabaseSeed()); err == nil && os.SameFile(info, seed) {
		return false, fmt.Errorf("runtime SQLite database aliases the distributed seed")
	}
	return true, nil
}

// Pool shares one durable writer per process. SQLite coordinates the login and
// world writers through WAL; each process serializes its own write transactions.
func Pool(database string) (*sql.DB, error) {
	if p := openPool("sqlite"); p != nil {
		return p, nil
	}
	mu.Lock()
	defer mu.Unlock()
	if p := pools["sqlite"]; p != nil {
		return p, nil
	}
	if err := Initialize(); err != nil {
		return nil, err
	}
	dsn, err := config.DSN(database)
	if err != nil {
		return nil, err
	}
	p, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	p.SetMaxOpenConns(1)
	c, cancel := ctx()
	defer cancel()
	var version int
	if err = p.QueryRowContext(c, "PRAGMA user_version").Scan(&version); err == nil && version != 1 {
		err = fmt.Errorf("unsupported SQLite schema version %d", version)
	}
	if err != nil {
		p.Close()
		return nil, err
	}
	if err = migrate(c, p, migrations); err != nil {
		p.Close()
		return nil, err
	}

	pools["sqlite"] = p
	return p, nil
}
func openPool(name string) *sql.DB {
	mu.RLock()
	defer mu.RUnlock()
	return pools[name]
}

// ReadPool serves queries whose decisions do not require the writer lock.
// Always revalidate state inside the write transaction before committing.
func ReadPool() (*sql.DB, error) {
	if p := openPool("reader"); p != nil {
		return p, nil
	}
	if _, err := Pool(GameData); err != nil {
		return nil, err
	}
	mu.Lock()
	defer mu.Unlock()
	if p := pools["reader"]; p != nil {
		return p, nil
	}
	dsn, err := config.ReadDSN()
	if err != nil {
		return nil, err
	}
	p, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	p.SetMaxOpenConns(4)
	p.SetMaxIdleConns(4)
	pools["reader"] = p
	return p, nil
}

func Open(databases ...string) error {
	for _, d := range databases {
		if _, err := Pool(d); err != nil {
			return err
		}
	}
	return nil
}
func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), Timeout)
}
func longCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// Close runs bounded planner maintenance and closes process-owned connections.
// Call only after network handlers have drained.
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	var result error
	for key, p := range pools {
		if key == "sqlite" {
			c, cancel := ctx()
			_, err := p.ExecContext(c, "PRAGMA optimize")
			cancel()
			result = errors.Join(result, err)
		}
		writeQueues.Delete(p)
		result = errors.Join(result, p.Close())
		delete(pools, key)
	}
	return result
}

// Observe reports bounded database contention and WAL size without acquiring the
// writer connection for checkpointing. Maintenance never truncates a live WAL.
func Observe() {
	p := openPool("sqlite")
	if p == nil {
		return
	}
	stats := p.Stats()
	dbWaits.Set(stats.WaitCount)
	dbWaitNanos.Set(stats.WaitDuration.Nanoseconds())
	dbOpen.Set(int64(stats.OpenConnections))
	if reader := openPool("reader"); reader != nil {
		stats := reader.Stats()
		dbReaderWaits.Set(stats.WaitCount)
		dbReaderWaitNanos.Set(stats.WaitDuration.Nanoseconds())
	}
	if info, err := os.Stat(config.DatabasePath() + "-wal"); err == nil {
		dbWAL.Set(info.Size())
	} else if os.IsNotExist(err) {
		dbWAL.Set(0)
	}
}

var dbWaits = telemetry.NumberMetric("gx_db_connection_wait_total")
var dbWaitNanos = telemetry.NumberMetric("gx_db_connection_wait_nanoseconds_total")
var dbOpen = telemetry.NumberMetric("gx_db_connections_open")
var dbWAL = telemetry.NumberMetric("gx_db_wal_bytes")

var dbReaderWaits = telemetry.NumberMetric("gx_db_reader_wait_total")
var dbReaderWaitNanos = telemetry.NumberMetric("gx_db_reader_wait_nanoseconds_total")
