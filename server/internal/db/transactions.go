package db

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/telemetry"
)

const (
	writeGroupSize  = 32
	writeQueueLimit = 2048
	writeGatherTime = time.Millisecond
)

type groupedWrite struct {
	context.Context
	run  func(context.Context, *sql.Tx) error
	done chan error
}

type writeQueue struct {
	sync.Mutex
	pending []groupedWrite
	running bool
}

var writeQueues sync.Map // keyed by the owning writer pool
var groupedCommits = telemetry.NumberMetric("gx_db_group_commits_total")
var groupedOperations = telemetry.NumberMetric("gx_db_group_operations_total")

// groupWrite shares a durable commit across independent operations. Each
// operation has its own savepoint and learns its result only after COMMIT.
// A caller whose request is already executing waits for the commit outcome.
func groupWrite(c context.Context, pool *sql.DB, fn func(context.Context, *sql.Tx) error) error {
	q, ok := writeQueues.Load(pool)
	if !ok {
		q, _ = writeQueues.LoadOrStore(pool, &writeQueue{})
	}
	queue := q.(*writeQueue)
	req := groupedWrite{Context: c, run: fn, done: make(chan error, 1)}
	queue.Lock()
	if len(queue.pending) >= writeQueueLimit {
		queue.Unlock()
		return errors.New("SQLite write queue is full")
	}
	queue.pending = append(queue.pending, req)
	if !queue.running {
		queue.running = true
		go queue.run(pool)
	}
	queue.Unlock()
	return <-req.done
}

func (q *writeQueue) run(pool *sql.DB) {
	time.Sleep(writeGatherTime)
	for {
		q.Lock()
		if len(q.pending) == 0 {
			q.running = false
			q.Unlock()
			return
		}
		var tasks [writeGroupSize]groupedWrite
		n := copy(tasks[:], q.pending)
		clear(q.pending[:n])
		q.pending = q.pending[n:]
		if len(q.pending) == 0 {
			q.pending = nil
		}
		q.Unlock()
		commitWriteGroup(pool, tasks[:n])
	}
}

func commitWriteGroup(pool *sql.DB, tasks []groupedWrite) {
	c, cancel := ctx()
	defer cancel()
	var outcomes [writeGroupSize]error
	committed := int64(0)
	err := withTx(c, pool, func(tx *sql.Tx) error {
		for i, task := range tasks {
			if expired := task.Err(); expired != nil {
				outcomes[i] = expired
				continue
			}
			if _, err := tx.ExecContext(c, "SAVEPOINT gx_write"); err != nil {
				return err
			}
			outcomes[i] = task.run(c, tx)
			if outcomes[i] != nil {
				if _, err := tx.ExecContext(c, "ROLLBACK TO gx_write"); err != nil {
					return err
				}
			} else {
				committed++
			}
			if _, err := tx.ExecContext(c, "RELEASE gx_write"); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil && committed > 0 {
		groupedCommits.Add(1)
		groupedOperations.Add(committed)
	}
	for i, task := range tasks {
		result := outcomes[i]
		if result == nil {
			result = err
		} else {
			transactionFailures.Add(1)
			reportStorageFailure(result)
		}
		task.done <- result
	}
}

var transactionTime = telemetry.DurationMetric("gx_db_transaction_seconds")
var transactionFailures = telemetry.NumberMetric("gx_db_transaction_failures_total")

func withTx(c context.Context, pool *sql.DB, fn func(*sql.Tx) error) (err error) {
	start := time.Now()
	defer func() {
		transactionTime.Observe(time.Since(start))
		if err != nil {
			transactionFailures.Add(1)
			reportStorageFailure(err)
		}
	}()
	tx, err := pool.BeginTx(c, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // also releases the writer lock if the operation panics
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
