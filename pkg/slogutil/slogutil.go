// Package slogutil implements utility functions for improved usage of the
// log/slog package.
package slogutil

import "log/slog"

// Valuer returns an Attr for an implementer of slog.LogValuer.
//
// This is useful because it enforces the interface at compile-time rather than
// relying on slog.Any which has additional behavior that can be triggered
// accidentally.
func Valuer(name string, valuer slog.LogValuer) slog.Attr {
	return slog.Any(name, valuer)
}
