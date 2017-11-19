package rollrus

import (
	"fmt"
	"reflect"
	"testing"
	"time"

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
	hook, _ := NewHook("my-secret-token", "production")
	log.Hooks.Add(hook)

	// This will not be reported to Rollbar
	log.WithFields(logrus.Fields{"power_level": "9001"}).Debug("It's over 9000!")

	// This will be reported to Rollbar
	log.Panic("Boom.")
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

func TestTriggerLevels(t *testing.T) {
	client := roll.New("foobar", "testing")
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