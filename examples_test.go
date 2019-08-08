package rollrus

import "github.com/sirupsen/logrus"

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
