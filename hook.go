package rollrus

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stvp/roll"
)

var _ logrus.Hook = &Hook{} //assert that *Hook is a logrus.Hook

// Hook is a wrapper for the rollbar Client and is usable as a logrus.Hook.
type Hook struct {
	roll.Client
	triggers        []logrus.Level
	ignoredErrors   []error
	ignoreErrorFunc func(error) bool
	ignoreFunc      func(error, map[string]string) bool

	// only used for tests to verify whether or not a report happened.
	reported bool
}

// NewHookForLevels provided by the caller. Otherwise works like NewHook.
func NewHookForLevels(token string, env string, levels []logrus.Level) *Hook {
	return &Hook{
		Client:          roll.New(token, env),
		triggers:        levels,
		ignoredErrors:   make([]error, 0),
		ignoreErrorFunc: func(error) bool { return false },
		ignoreFunc:      func(error, map[string]string) bool { return false },
	}
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

// Levels returns the logrus log.Levels that this hook handles
func (r *Hook) Levels() []logrus.Level {
	if r.triggers == nil {
		return defaultTriggerLevels
	}
	return r.triggers
}

// Fire the hook. This is called by Logrus for entries that match the levels
// returned by Levels().
func (r *Hook) Fire(entry *logrus.Entry) error {
	trace, cause := extractError(entry)
	for _, ie := range r.ignoredErrors {
		if ie == cause {
			return nil
		}
	}

	if r.ignoreErrorFunc(cause) {
		return nil
	}

	m := convertFields(entry.Data)
	if _, exists := m["time"]; !exists {
		m["time"] = entry.Time.Format(time.RFC3339)
	}

	if r.ignoreFunc(cause, m) {
		return nil
	}

	return r.report(entry, cause, m, trace)
}

func (r *Hook) report(entry *logrus.Entry, cause error, m map[string]string, trace []uintptr) (err error) {
	hasTrace := len(trace) > 0
	level := entry.Level

	r.reported = true

	switch {
	case hasTrace && level == logrus.FatalLevel:
		_, err = r.Client.CriticalStack(cause, trace, m)
	case hasTrace && level == logrus.PanicLevel:
		_, err = r.Client.CriticalStack(cause, trace, m)
	case hasTrace && level == logrus.ErrorLevel:
		_, err = r.Client.ErrorStack(cause, trace, m)
	case hasTrace && level == logrus.WarnLevel:
		_, err = r.Client.WarningStack(cause, trace, m)
	case level == logrus.FatalLevel || level == logrus.PanicLevel:
		_, err = r.Client.Critical(cause, m)
	case level == logrus.ErrorLevel:
		_, err = r.Client.Error(cause, m)
	case level == logrus.WarnLevel:
		_, err = r.Client.Warning(cause, m)
	case level == logrus.InfoLevel:
		_, err = r.Client.Info(entry.Message, m)
	case level == logrus.DebugLevel:
		_, err = r.Client.Debug(entry.Message, m)
	}
	return err
}

// convertFields converts from log.Fields to map[string]string so that we can
// report extra fields to Rollbar
func convertFields(fields logrus.Fields) map[string]string {
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

// extractError attempts to extract an error from a well known field, err or error
func extractError(entry *logrus.Entry) ([]uintptr, error) {
	var trace []uintptr
	fields := entry.Data

	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	for _, f := range wellKnownErrorFields {
		e, ok := fields[f]
		if !ok {
			continue
		}
		err, ok := e.(error)
		if !ok {
			continue
		}

		cause := errors.Cause(err)
		if cause == nil {
			cause = err
		}
		tracer, ok := err.(stackTracer)
		if ok {
			return copyStackTrace(tracer.StackTrace()), cause
		}
		return trace, cause
	}

	// when no error found, default to the logged message.
	return trace, fmt.Errorf(entry.Message)
}

func copyStackTrace(trace errors.StackTrace) (out []uintptr) {
	for _, frame := range trace {
		out = append(out, uintptr(frame))
	}
	return
}
