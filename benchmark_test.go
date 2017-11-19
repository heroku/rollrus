package rollrus

import (
	"io/ioutil"
	"os"
	"testing"

	"io"

	"github.com/sirupsen/logrus"
)

var vanillaLogger *logrus.Logger
var rollrusLogger *logrus.Logger
var rollrusCloser io.Closer

func init() {
	token := os.Getenv("ROLLBAR_TOKEN")
	if token == "" {
		panic("Could not get rollbar token")
	}

	vanillaLogger = logrus.New()
	rollrusLogger = logrus.New()

	vanillaLogger.Out = ioutil.Discard
	rollrusLogger.Out = ioutil.Discard

	rollrus, closer := NewHook(token, "test")
	rollrusLogger.AddHook(rollrus)
	rollrusCloser = closer
}

func BenchmarkVanillaLogger(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vanillaLogger.Error("test")
	}
}

func BenchmarkRollrusLogger(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rollrusLogger.Error("test")
	}
	rollrusCloser.Close()
}
