package app

import "context"

// An unbuffered result transfers ownership only while the attempt is active.
// A connection that finishes after leaving the screen belongs to its worker.
type connectionAttempt struct {
	result chan func()
	ctx    context.Context
	cancel context.CancelFunc
}

func (a *App) beginConnection() *connectionAttempt {
	ctx, cancel := context.WithCancel(context.Background())
	attempt := &connectionAttempt{
		result: make(chan func()),
		ctx:    ctx,
		cancel: cancel,
	}
	a.connection = attempt
	return attempt
}

func (a *App) pollConnection() {
	if attempt := a.connection; attempt != nil {
		select {
		case apply := <-attempt.result:
			a.connection = nil
			attempt.cancel()
			apply()
		default:
		}
	}
}

func (attempt *connectionAttempt) deliver(apply, discard func()) {
	select {
	case attempt.result <- apply:
	case <-attempt.ctx.Done():
		discard()
	}
}

func (a *App) cancelConnection() {
	if attempt := a.connection; attempt != nil {
		attempt.cancel()
		a.connection = nil
	}
	if a.sess != nil {
		a.sess.Close()
		a.sess = nil
	}
	if a.login != nil {
		a.login.Busy = false
	}
	if a.servers != nil {
		a.servers.Busy = false
	}
}
