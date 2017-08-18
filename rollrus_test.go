package rollrus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stvp/roll"
)

func ExampleSetupLogging() {
	SetupLogging("some-long-token", "staging")

	// This will not be reported to Rollbar
	logrus.Info("OHAI")

	// This will be reported to Rollbar
	logrus.WithFields(logrus.Fields{"hi": "there"}).Fatal("The end.")
}

func ExampleNewHook() {
	log := logrus.New()
	hook := NewHook("my-secret-token", "production")
	log.Hooks.Add(hook)

	// This will not be reported to Rollbar
	log.WithFields(logrus.Fields{"power_level": "9001"}).Debug("It's over 9000!")

	// This will be reported to Rollbar
	log.Panic("Boom.")
}

func TestLogrusHookInterface(t *testing.T) {
	var hook interface{} = NewHook("", "foo")
	if _, ok := hook.(logrus.Hook); !ok {
		t.Fatal("expected NewHook's return value to implement logrus.Hook")
	}
}

func TestIntConversion(t *testing.T) {
	i := make(logrus.Fields)
	i["test"] = 5

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "5" {
		t.Fatal("Expected value to equal 5, but instead it is: ", v)
	}
}

func TestErrConversion(t *testing.T) {
	i := make(logrus.Fields)
	i["test"] = fmt.Errorf("This is an error")

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "This is an error" {
		t.Fatal("Expected value to be a string of the error but instead it is: ", v)
	}
}

func TestStringConversion(t *testing.T) {
	i := make(logrus.Fields)
	i["test"] = "This is a string"

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "This is a string" {
		t.Fatal("Expected value to equal a certain string, but instead it is: ", v)
	}
}

func TestTimeConversion(t *testing.T) {
	now := time.Now()
	i := make(logrus.Fields)
	i["test"] = now

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != now.Format(time.RFC3339) {
		t.Fatal("Expected value to equal, but instead it is: ", v)
	}
}

func TestExtractError(t *testing.T) {
	entry := logrus.NewEntry(nil)
	entry.Data["err"] = fmt.Errorf("foo bar baz")

	trace, cause := extractError(entry)
	if len(trace) != 0 {
		t.Fatal("Expected length of trace to be equal to 0, but instead is: ", len(trace))
	}

	if cause.Error() != "foo bar baz" {
		t.Fatalf("Expected error as string to be 'foo bar baz', but was instead: %q", cause)
	}
}

func TestExtractErrorDefault(t *testing.T) {
	entry := logrus.NewEntry(nil)
	entry.Data["no-err"] = fmt.Errorf("foo bar baz")
	entry.Message = "message error"

	trace, cause := extractError(entry)
	if len(trace) != 0 {
		t.Fatal("Expected length of trace to be equal to 0, but instead is: ", len(trace))
	}

	if cause.Error() != "message error" {
		t.Fatalf("Expected error as string to be 'message error', but was instead: %q", cause)
	}
}

func TestExtractErrorFromStackTracer(t *testing.T) {
	entry := logrus.NewEntry(nil)
	entry.Data["err"] = errors.Errorf("foo bar baz")

	trace, cause := extractError(entry)
	if len(trace) != 3 {
		t.Fatal("Expected length of trace to be == 3, but instead is: ", len(trace))
	}

	if cause.Error() != "foo bar baz" {
		t.Fatalf("Expected error as string to be 'foo bar baz', but was instead: %q", cause.Error())
	}
}

func TestTriggerLevels(t *testing.T) {
	client := roll.New("", "testing")
	underTest := &Hook{Client: client}
	if !reflect.DeepEqual(underTest.Levels(), defaultTriggerLevels) {
		t.Fatal("Expected Levels() to return defaultTriggerLevels")
	}

	newLevels := []logrus.Level{logrus.InfoLevel}
	underTest.triggers = newLevels
	if !reflect.DeepEqual(underTest.Levels(), newLevels) {
		t.Fatal("Expected Levels() to return newLevels")
	}
}

func TestWithIgnoredErrors(t *testing.T) {
	h := NewHook("", "testing", WithIgnoredErrors(io.EOF))
	entry := logrus.NewEntry(nil)
	entry.Message = "This is a test"

	// Exact error is skipped.
	entry.Data["err"] = io.EOF
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if h.reported {
		t.Fatal("expected no report to have happened")
	}

	// Wrapped error is also skipped.
	entry.Data["err"] = errors.Wrap(io.EOF, "hello")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if h.reported {
		t.Fatal("expected no report to have happened")
	}

	// Non blacklisted errors get reported.
	entry.Data["err"] = errors.New("hello")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if !h.reported {
		t.Fatal("expected a report to have happened")
	}

	// no err gets reported.
	delete(entry.Data, "err")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if !h.reported {
		t.Fatal("expected a report to have happened")
	}
}

type isTemporary interface {
	Temporary() bool
}

func TestWithIgnoreErrorFunc(t *testing.T) {
	h := NewHook("", "testing", WithIgnoreErrorFunc(func(err error) bool {
		if err == io.EOF {
			return true
		}

		if e, ok := err.(isTemporary); ok && e.Temporary() {
			return true
		}

		return false
	}))

	entry := logrus.NewEntry(nil)
	entry.Message = "This is a test"

	// Exact error is skipped.
	entry.Data["err"] = io.EOF
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if h.reported {
		t.Fatal("expected no report to have happened")
	}

	// Wrapped error is also skipped.
	entry.Data["err"] = errors.Wrap(io.EOF, "hello")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if h.reported {
		t.Fatal("expected no report to have happened")
	}

	srv := httptest.NewServer(nil)
	srv.Close()

	// Temporary error skipped
	_, err := http.Get(srv.URL)

	entry.Data["err"] = err
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}

	// Non blacklisted errors get reported.
	entry.Data["err"] = errors.New("hello")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if !h.reported {
		t.Fatal("expected a report to have happened")
	}

	// no err gets reported.
	delete(entry.Data, "err")
	if err := h.Fire(entry); err != nil {
		t.Fatal("unexpected error ", err)
	}
	if !h.reported {
		t.Fatal("expected a report to have happened")
	}
}
