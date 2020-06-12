package rollrus

import (
	"context"
	"errors"
	"sync"

	"github.com/rollbar/rollbar-go"
)

var (
	errBufferFull = errors.New("rollbar message buffer full")
	errClosed     = errors.New("rollbar transport closed")
)

// bufferedTransport is an alternative to rollbar's AsyncTransport, providing
// threadsafe and predictable built on top of the SyncTransport.
type bufferedTransport struct {
	queue chan transportOp
	once  sync.Once
	ctx   context.Context

	rollbar.Transport
}

// transportOp represents an operation queued for transport. It is only valid
// to set a single field in the struct to represent the operation that should
// be performed.
type transportOp struct {
	send  map[string]interface{}
	wait  chan struct{}
	close bool
}

func newBufferTransport(inner rollbar.Transport, bufSize int) *bufferedTransport {
	ctx, cancel := context.WithCancel(context.Background())

	t := &bufferedTransport{
		queue:     make(chan transportOp, bufSize),
		ctx:       ctx,
		Transport: inner,
	}

	go t.run(cancel)

	return t
}

// Send enqueues delivery of the message body to Rollbar without waiting for
// the result. If the buffer is full, it will immediately return an error.
func (t *bufferedTransport) Send(body map[string]interface{}) error {
	select {
	case t.queue <- transportOp{send: body}:
		return nil
	case <-t.ctx.Done():
		return errClosed
	default:
		return errBufferFull
	}
}

// Wait blocks until all messages buffered before calling Wait are
// delivered.
func (t *bufferedTransport) Wait() {
	done := make(chan struct{})
	select {
	case t.queue <- transportOp{wait: done}:
	case <-t.ctx.Done():
		return
	}

	select {
	case <-done:
	case <-t.ctx.Done():
	}
}

// Close shuts down the transport and waits for queued messages to be
// delivered.
func (t *bufferedTransport) Close() error {
	t.once.Do(func() {
		t.queue <- transportOp{close: true}
	})

	<-t.ctx.Done()
	return nil
}

func (t *bufferedTransport) run(cancel func()) {
	defer cancel()

	for m := range t.queue {
		switch {
		case m.send != nil:
			_ = t.Transport.Send(m.send)
		case m.wait != nil:
			close(m.wait)
		case m.close:
			t.Transport.Close()
			return
		}
	}
}
