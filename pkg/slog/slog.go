// Package slog takes only the good parts from the log/slog package and
// extends them to improve ergonomics and performance.
package slog

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/lmittmann/tint"
)

// Names for common levels.
//
// Level numbers are inherently arbitrary,
// but we picked them to satisfy three constraints.
// Any system can map them to another numbering scheme if it wishes.
//
// First, we wanted the default level to be Info, Since Levels are ints, Info is
// the default value for int, zero.
//
// Second, we wanted to make it easy to use levels to specify logger verbosity.
// Since a larger level means a more severe event, a logger that accepts events
// with smaller (or more negative) level means a more verbose logger. Logger
// verbosity is thus the negation of event severity, and the default verbosity
// of 0 accepts all events at least as severe as INFO.
//
// Third, we wanted some room between levels to accommodate schemes with named
// levels between ours. For example, Google Cloud Logging defines a Notice level
// between Info and Warn. Since there are only a few of these intermediate
// levels, the gap between the numbers need not be large. Our gap of 4 matches
// OpenTelemetry's mapping. Subtracting 9 from an OpenTelemetry level in the
// DEBUG, INFO, WARN and ERROR ranges converts it to the corresponding slog
// Level range. OpenTelemetry also has the names TRACE and FATAL, which slog
// does not. But those OpenTelemetry levels can still be represented as slog
// Levels by using the appropriate integers.
const (
	LevelDebug Level = slog.LevelDebug
	LevelInfo  Level = slog.LevelInfo
	LevelWarn  Level = slog.LevelWarn
	LevelError Level = slog.LevelError
)

// A Handler handles log records produced by a Logger.
//
// A typical handler may print log records to standard error,
// or write them to a file or database, or perhaps augment them
// with additional attributes and pass them on to another handler.
//
// Any of the Handler's methods may be called concurrently with itself
// or with other methods. It is the responsibility of the Handler to
// manage this concurrency.
//
// Users of the slog package should not invoke Handler methods directly.
// They should use the methods of [Logger] instead.
//
// Before implementing your own handler, consult https://go.dev/s/slog-handler-guide.
type Handler = slog.Handler

// A Level is the importance or severity of a log event.
// The higher the level, the more important or severe the event.
type Level = slog.Level

// A Value can represent any Go value, but unlike type any,
// it can represent most small values without an allocation.
// The zero Value corresponds to nil.
type Value = slog.Value

// A LogValuer is any Go value that can convert itself into a Value for logging.
//
// This mechanism may be used to defer expensive operations until they are
// needed, or to expand a single value into a sequence of components.
type LogValuer = slog.LogValuer

// Valuer returns an Attr for an implementer of slog.LogValuer.
//
// This is useful because it enforces the interface at compile-time rather than
// relying on slog.Any which has additional behavior that can be triggered
// accidentally.
func Valuer(name string, valuer LogValuer) slog.Attr {
	return slog.Any(name, valuer)
}

// Any returns an Attr for the supplied value.
// See [slog.AnyValue] for how values are treated.
func Any(name string, value any) slog.Attr {
	return slog.Any(name, value)
}

// String returns an Attr for a string value.
func String(name, value string) slog.Attr {
	return slog.String(name, value)
}

// Bool returns an Attr for a bool.
func Bool(name string, value bool) slog.Attr {
	return slog.Bool(name, value)
}

// Int converts an int to an int64 and returns
// an Attr with that value.
func Int(name string, value int) slog.Attr {
	return slog.Int(name, value)
}

// Uint64 returns an Attr for a uint64.
func Uint64(name string, value uint64) slog.Attr {
	return slog.Uint64(name, value)
}

// Float64 returns an Attr for a floating-point number.
func Float64(name string, value float64) slog.Attr {
	return slog.Float64(name, value)
}

// Time returns an Attr for a [time.Time].
// It discards the monotonic portion.
func Time(name string, value time.Time) slog.Attr {
	return slog.Time(name, value)
}

// Duration returns an Attr for a [time.Duration].
func Duration(name string, value time.Duration) slog.Attr {
	return slog.Duration(name, value)
}

// Err returns an Attr for an [error].
func Err(value error) slog.Attr {
	return slog.Any("error", value)
}

// GroupValue returns a new [Value] for a list of Attrs.
// The caller must not subsequently mutate the argument slice.
func GroupValue(as ...slog.Attr) Value {
	return slog.GroupValue(as...)
}

// DebugEnabled reports whether the default handler handles records at the
// debug level.
func DebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}

// ErrorEnabled reports whether the default handler handles records at the
// error level.
func ErrorEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelError)
}

// Debug logs at [slog.LevelDebug] using [slog.Attr]s only for performance.
func Debug(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

// Info logs at [slog.LevelInfo] using [slog.Attr]s only for performance.
func Info(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// Warn logs at [slog.LevelWarn] using [slog.Attr]s only for performance.
func Warn(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// Error logs at [slog.LevelError] using [slog.Attr]s only for performance.
func Error(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// SetDefaultHandler makes h the handler used by the default [Logger], which
// is used by the top-level functions [Info], [Debug] and so on.
//
// After this call, output from the log package's default Logger
// (as with [log.Print], etc.) will be logged using l's Handler,
// at a level controlled by [SetLogLoggerLevel].
func SetDefaultHandler(h Handler) {
	slog.SetDefault(slog.New(h))
}

// NewJSONHandler creates a [Handler] that writes JSON to w using the given
// level.
func NewJSONHandler(w io.Writer, l Level) Handler {
	return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: l})
}

// NewTextHandler creates a [Handler] that writes tinted logs to Writer w,
// using the given level and color options.
func NewTextHandler(w io.Writer, l Level, noColor bool) Handler {
	return tint.NewHandler(w, &tint.Options{
		TimeFormat: time.RFC3339,
		Level:      l,
		NoColor:    noColor,
	})
}
