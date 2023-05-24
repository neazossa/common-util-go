package logrus

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/neazzosa/common-util-go/logger/logger"
	"github.com/sirupsen/logrus"
)

type (
	logger struct {
		instance *logrus.Logger
		data     map[string]interface{}
	}

	Level     string
	Formatter string

	Option struct {
		Level       Level
		LogFilePath string
		Formatter   Formatter
		MaxSize     int
		MaxBackups  int
		MaxAge      time.Duration
		Compress    bool
	}
)

const (
	Info  Level = "INFO"
	Debug Level = "DEBUG"
	Error Level = "ERROR"

	JSONFormatter Formatter = "JSON"
	TextFormatter Formatter = "TEXT"
)

func New(option *Option) (log.Logger, error) {
	instance := logrus.New()

	if option.Level == Info {
		instance.Level = logrus.InfoLevel
	}

	if option.Level == Debug {
		instance.Level = logrus.DebugLevel
	}

	if option.Level == Error {
		instance.Level = logrus.ErrorLevel
	}

	var formatter logrus.Formatter

	if option.Formatter == JSONFormatter {
		formatter = &logrus.JSONFormatter{DisableHTMLEscape: true}
	} else {
		formatter = &logrus.TextFormatter{}
	}

	instance.Formatter = formatter

	// - check if log file path does exists
	if option.LogFilePath != "" {
		logf, err := rotatelogs.New(
			option.LogFilePath+"/RequestResponseDump.log.%Y%m%d",
			rotatelogs.WithLinkName(option.LogFilePath+"/RequestResponseDump.log"),
			rotatelogs.WithRotationTime(24*time.Hour),
			rotatelogs.WithMaxAge(option.MaxAge),
		)

		if err != nil {
			return nil, err
		}

		instance.SetOutput(io.MultiWriter(os.Stdout, logf))
	}

	return &logger{instance, nil}, nil
}

func fileInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if ok {
		return fmt.Sprintf("%s:%d", file, line)
	} else {
		return ""
	}
}

func (l *logger) createEntry() *logrus.Entry {
	entry := l.instance.WithFields(logrus.Fields{})
	if l.data != nil {
		entry = l.instance.WithFields(map[string]interface{}{
			"data": l.data,
		})
	}
	entry.Data["file"] = fileInfo(3)
	return entry
}

func (l *logger) WithFields(data map[string]interface{}) log.Logger {
	return &logger{
		instance: l.instance,
		data:     data,
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.createEntry().Debugf(format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.createEntry().Infof(format, args...)
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.createEntry().Printf(format, args...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.createEntry().Warnf(format, args...)
}

func (l *logger) Warningf(format string, args ...interface{}) {
	l.createEntry().Warningf(format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.createEntry().Errorf(format, args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	l.createEntry().Fatalf(format, args...)
}

func (l *logger) Panicf(format string, args ...interface{}) {
	l.createEntry().Panicf(format, args...)
}

func (l *logger) Debug(args ...interface{}) {
	l.createEntry().Debug(args...)
}

func (l *logger) Info(args ...interface{}) {
	l.createEntry().Info(args...)
}

func (l *logger) Print(args ...interface{}) {
	l.createEntry().Print(args...)
}

func (l *logger) Warn(args ...interface{}) {
	l.createEntry().Warn(args...)
}

func (l *logger) Warning(args ...interface{}) {
	l.createEntry().Warning(args...)
}

func (l *logger) Error(args ...interface{}) {
	l.createEntry().Error(args...)
}

func (l *logger) Fatal(args ...interface{}) {
	l.createEntry().Fatal(args...)
}

func (l *logger) Panic(args ...interface{}) {
	l.createEntry().Panic(args...)
}

func (l *logger) Debugln(args ...interface{}) {
	l.createEntry().Debugln(args...)
}

func (l *logger) Infoln(args ...interface{}) {
	l.createEntry().Infoln(args...)
}

func (l *logger) Println(args ...interface{}) {
	l.createEntry().Println(args...)
}

func (l *logger) Warnln(args ...interface{}) {
	l.createEntry().Warnln(args...)
}

func (l *logger) Warningln(args ...interface{}) {
	l.createEntry().Warningln(args...)
}

func (l *logger) Errorln(args ...interface{}) {
	l.createEntry().Errorln(args...)
}

func (l *logger) Fatalln(args ...interface{}) {
	l.createEntry().Fatalln(args...)
}

func (l *logger) Panicln(args ...interface{}) {
	l.createEntry().Panicln(args...)
}

func (l *logger) Trace(args ...interface{}) {
	l.createEntry().Trace(args...)
}
