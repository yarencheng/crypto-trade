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

func Debug(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Debug(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Debug(args...)
	}
}

func Info(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Info(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Info(args...)
	}
}

func Warn(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Warn(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Warn(args...)
	}
}

func Error(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Error(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Error(args...)
	}
}

func Fatal(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Fatal(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Fatal(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Debugf(format, args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Debugf(format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Infof(format, args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Infof(format, args...)
	}
}

func Warnf(format string, args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Warnf(format, args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Warnf(format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Errorf(format, args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Errorf(format, args...)
	}
}

func Fatalf(format string, args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Fatalf(format, args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Fatalf(format, args...)
	}
}

func Debugln(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Debugln(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Debugln(args...)
	}
}

func Infoln(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Infoln(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Infoln(args...)
	}

}

func Warnln(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Warnln(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Warnln(args...)
	}
}

func Errorln(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Errorln(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Errorln(args...)
	}
}

func Fatalln(args ...interface{}) {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		logrus.WithFields(
			logrus.Fields{
				"prefix": filepath.Base(file),
			},
		).Fatalln(args...)
	} else {
		logrus.WithFields(
			logrus.Fields{
				"prefix": "unknown",
			},
		).Fatalln(args...)
	}
}
