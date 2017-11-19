package rollrus

import (
	"fmt"
	"io"
	"os"
	"time"

	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/stvp/roll"
)

type noopCloser struct{}

func (c noopCloser) Close() error {
	return nil
}

var defaultTriggerLevels = []log.Level{
	log.ErrorLevel,
	log.FatalLevel,
	log.PanicLevel,
}

// Hook wrapper for the rollbar Client
// May be used as a rollbar client itself
type Hook struct {
	roll.Client
	triggers []log.Level
	entries  chan *log.Entry
	closed   chan struct{}
	once     *sync.Once
}

// Setup a new hook with default reporting levels, useful for adding to
// your own logger instance.
func NewHook(token string, env string) (*Hook, io.Closer) {
	return NewHookForLevels(token, env, defaultTriggerLevels)
}

// Setup a new hook with specified reporting levels, useful for adding to
// your own logger instance.
func NewHookForLevels(token string, env string, levels []log.Level) (*Hook, io.Closer) {
	h := &Hook{
		Client:   roll.New(token, env),
		triggers: levels,
		closed:   make(chan struct{}),
		entries:  make(chan *log.Entry, 100),
		once:     new(sync.Once),
	}

	go h.sendToRollbar()

	return h, h
}

// SetupLogging sets up logging. If token is not an empty string a rollbar
// hook is added with the environment set to env. The log formatter is set to a
// TextFormatter with timestamps disabled, which is suitable for use on Heroku.
func SetupLogging(token, env string) io.Closer {
	return setupLogging(token, env, defaultTriggerLevels)
}

// SetupLoggingForLevels works like SetupLogging, but allows you to
// set the levels on which to trigger this hook.
func SetupLoggingForLevels(token, env string, levels []log.Level) io.Closer {
	return setupLogging(token, env, levels)
}

func setupLogging(token, env string, levels []log.Level) io.Closer {
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})

	var closer io.Closer
	if token != "" {
		h, c := NewHookForLevels(token, env, levels)
		log.AddHook(h)
		closer = c
	} else {
		closer = noopCloser{}
	}

	return closer
}

// ReportPanic attempts to report the panic to rollbar using the provided
// client and then re-panic. If it can't report the panic it will print an
// error to stderr.
func (r *Hook) ReportPanic() {
	if p := recover(); p != nil {
		if _, err := r.Client.Critical(fmt.Errorf("panic: %q", p), nil); err != nil {
			fmt.Fprintf(os.Stderr, "reporting_panic=false err=%q\n", err)
		}
		panic(p)
	}
}

// ReportPanic attempts to report the panic to rollbar if the token is set
func ReportPanic(token, env string) {
	if token != "" {
		h := &Hook{Client: roll.New(token, env)}
		h.ReportPanic()
	}
}

// Fire the hook. This is called by Logrus for entries that match the levels
// returned by Levels(). See below.
func (r *Hook) Fire(entry *log.Entry) (err error) {
	select {
	case <-r.closed:
		//do nothing
	default:
		r.entries <- entry
	}

	return nil
}

func (r *Hook) sendToRollbar() {
	defer close(r.closed)
	for entry := range r.entries {
		e := fmt.Errorf(entry.Message)
		m := convertFields(entry.Data)
		if _, exists := m["time"]; !exists {
			m["time"] = entry.Time.Format(time.RFC3339)
		}

		var err error
		switch entry.Level {
		case log.FatalLevel, log.PanicLevel:
			_, err = r.Client.Critical(e, m)
		case log.ErrorLevel:
			_, err = r.Client.Error(e, m)
		case log.WarnLevel:
			_, err = r.Client.Warning(e, m)
		case log.InfoLevel:
			_, err = r.Client.Info(entry.Message, m)
		case log.DebugLevel:
			_, err = r.Client.Debug(entry.Message, m)
		default:
			err = fmt.Errorf("Unknown level: %s", entry.Level)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not send entry to rollbar: %v", err)
		}
	}
}

func (r *Hook) Close() error {
	r.once.Do(func() {
		close(r.entries)
	})

	select {
	case <-r.closed:
		return nil
	}
}

// Levels returns the logrus log levels that this hook handles
func (r *Hook) Levels() []log.Level {
	if r.triggers == nil {
		return defaultTriggerLevels
	}
	return r.triggers
}

// convertFields converts from log.Fields to map[string]string so that we can
// report extra fields to Rollbar
func convertFields(fields log.Fields) map[string]string {
	m := make(map[string]string)
	for k, v := range fields {
		switch t := v.(type) {
		case time.Time:
			m[k] = t.Format(time.RFC3339)
		default:
			if s, ok := v.(fmt.Stringer); ok {
				m[k] = s.String()
			} else {
				m[k] = fmt.Sprintf("%+v", t)
			}
		}
	}

	return m
}
