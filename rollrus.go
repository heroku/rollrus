package rollrus

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stvp/roll"
)

type Hook struct {
	Client roll.Client
}

func (r *Hook) Fire(entry *log.Entry) (err error) {
	e := fmt.Errorf(entry.Message)
	m := convertFields(entry.Data)
	if _, exists := m["time"]; !exists {
		m["time"] = entry.Time.Format(time.RFC3339)
	}

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
		return fmt.Errorf("Unknown level: %s", entry.Level)
	}

	return err
}

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
