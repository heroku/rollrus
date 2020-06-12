package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rollbar/rollbar-go"
)

func TestBufferedTransportSend(t *testing.T) {
	inner := &testTransport{
		sendHook: make(chan map[string]interface{}),
	}
	transport := NewBuffered(inner, 1)
	data := map[string]interface{}{"a": "b"}

	if err := transport.Send(data); err != nil {
		t.Fatal(err)
	}

	// Verify data is delivered to inner transport
	recv := <-inner.sendHook
	if recv["a"] != "b" {
		t.Errorf("transport sent %v, want %v", recv, data)
	}

	var lastErr error
	var sent int
	for ; sent < 10; sent++ {
		if err := transport.Send(data); err != nil {
			lastErr = err
			break
		}
	}

	if lastErr != errBufferFull {
		t.Fatal("send did not fill buffer")
	}

	// drain pending messages
	for i := 0; i < sent; i++ {
		<-inner.sendHook
	}

	transport.Close()

	lastErr = nil
	sent = 0
	for ; sent < 10; sent++ {
		if err := transport.Send(data); err != nil {
			lastErr = err
			break
		}
	}

	if lastErr != errClosed {
		t.Fatal("send after close did not return errClosed")
	}
}

func TestBufferedTransportWait(t *testing.T) {
	inner := &testTransport{}
	transport := NewBuffered(inner, 1)
	data := map[string]interface{}{"a": "b"}

	// Wait returns immediately when nothing is queued
	transport.Wait()
	transport.Wait()

	for i := 0; i < 100; i++ {
		_ = transport.Send(data)
		transport.Wait()
	}

	inner.sendHook = make(chan map[string]interface{})
	waitDone := make(chan struct{})

	if err := transport.Send(data); err != nil {
		t.Fatal(err)
	}
	go func() {
		transport.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("wait returned before message was sent")
	case <-inner.sendHook:
	}

	<-waitDone

	transport.Close()

	transport.Wait() // wait returns immediately after closed
}

// Regression test for original issue with async rollbar client:
//		https://github.com/rollbar/rollbar-go/issues/68#issuecomment-540308646
func TestBufferedTransportRace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sync := rollbar.NewSyncTransport("token", srv.URL)
	sync.SetLogger(&rollbar.SilentClientLogger{})

	transport := NewBuffered(sync, 1)
	body := map[string]interface{}{
		"hello": "world",
	}
	started := make(chan struct{})
	go func() {
		close(started)
		for {
			transport.Wait()
		}
	}()
	iter := make([]struct{}, 100)
	<-started
	for range iter {
		err := transport.Send(body)
		if err != nil {
			if err == errBufferFull {
				time.Sleep(time.Millisecond)
				continue
			}
			t.Error("Send returned an unexpected error:", err)
		}
	}
}

type testTransport struct {
	sendHook chan map[string]interface{}
	rollbar.Transport
}

func (t *testTransport) Send(body map[string]interface{}) error {
	if t.sendHook != nil {
		t.sendHook <- body
	}
	return nil
}

func (t *testTransport) Close() error {
	return nil
}
