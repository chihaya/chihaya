// Package log adds a thin wrapper around logrus to improve non-debug logging
// performance.
package log

import (
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

var (
	l     = logrus.New()
	debug = false
)

// SetDebug controls debug logging.
func SetDebug(to bool) {
	debug = to
	l.Level = logrus.DebugLevel
}

// SetFormatter sets the formatter.
func SetFormatter(to logrus.Formatter) {
	l.Formatter = to
}

// SetOutput sets the output.
func SetOutput(to io.Writer) {
	l.Out = to
}

// Fields is a map of logging fields.
type Fields map[string]interface{}

// LogFields implements Fielder for Fields.
func (f Fields) LogFields() Fields {
	return f
}

// A Fielder provides Fields via the LogFields method.
type Fielder interface {
	LogFields() Fields
}

// err is a wrapper around an error.
type err struct {
	e error
}

// LogFields provides Fields for logging.
func (e err) LogFields() Fields {
	return Fields{
		"error": e.e.Error(),
		"type":  fmt.Sprintf("%T", e.e),
	}
}

// Err is a wrapper around errors that implements Fielder.
func Err(e error) Fielder {
	return err{e}
}

// mergeFielders merges the Fields of multiple Fielders.
// Fields from the first Fielder will be used unchanged, Fields from subsequent
// Fielders will be prefixed with "%d.", starting from 1.
//
// must be called with len(fielders) > 0
func mergeFielders(fielders ...Fielder) logrus.Fields {
	if fielders[0] == nil {
		return nil
	}

	fields := fielders[0].LogFields()
	for i := 1; i < len(fielders); i++ {
		if fielders[i] == nil {
			continue
		}
		prefix := fmt.Sprint(i, ".")
		ff := fielders[i].LogFields()
		for k, v := range ff {
			fields[prefix+k] = v
		}
	}

	return logrus.Fields(fields)
}

// Debug logs at the debug level if debug logging is enabled.
func Debug(v interface{}, fielders ...Fielder) {
	if debug {
		if len(fielders) != 0 {
			l.WithFields(mergeFielders(fielders...)).Debug(v)
		} else {
			l.Debug(v)
		}
	}
}

// Info logs at the info level.
func Info(v interface{}, fielders ...Fielder) {
	if len(fielders) != 0 {
		l.WithFields(mergeFielders(fielders...)).Info(v)
	} else {
		l.Info(v)
	}
}

// Warn logs at the warning level.
func Warn(v interface{}, fielders ...Fielder) {
	if len(fielders) != 0 {
		l.WithFields(mergeFielders(fielders...)).Warn(v)
	} else {
		l.Warn(v)
	}
}

// Error logs at the error level.
func Error(v interface{}, fielders ...Fielder) {
	if len(fielders) != 0 {
		l.WithFields(mergeFielders(fielders...)).Error(v)
	} else {
		l.Error(v)
	}
}

// Fatal logs at the fatal level and exits with a status code != 0.
func Fatal(v interface{}, fielders ...Fielder) {
	if len(fielders) != 0 {
		l.WithFields(mergeFielders(fielders...)).Fatal(v)
	} else {
		l.Fatal(v)
	}
}
