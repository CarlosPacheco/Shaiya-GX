package db

import (
	"errors"
	"modernc.org/sqlite"
	"sync/atomic"
)

// StorageUnavailable distinguishes an unsafe storage failure from a refused
// gameplay plan or a transient busy database. The world must stop admitting
// mutations if durable storage is no longer writable or trustworthy.
func StorageUnavailable(err error) bool {
	var failure *sqlite.Error
	if !errors.As(err, &failure) {
		return false
	}
	switch failure.Code() & 255 {
	case 3, 8, 10, 11, 13, 26:
		return true
	}
	return false
}

type failureCallback struct{ run func(error) }

var failureHandler atomic.Pointer[failureCallback]

func OnStorageFailure(handler func(error)) func() {
	box := &failureCallback{run: handler}
	failureHandler.Store(box)
	return func() { failureHandler.CompareAndSwap(box, nil) }
}
func reportStorageFailure(err error) {
	if StorageUnavailable(err) {
		if handler := failureHandler.Load(); handler != nil {
			handler.run(err)
		}
	}
}
