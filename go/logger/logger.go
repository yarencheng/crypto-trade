package logger

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}

func Get(filename string) *logrus.Entry {

	_, file, _, ok := runtime.Caller(1)

	if ok {
		return logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		)
	} else {
		return logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		)
	}
}
