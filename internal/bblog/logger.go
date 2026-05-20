/*
FILE: internal/bblog/logger.go

DESCRIPTION:
Lightweight logging interface used across all SDK sub-packages.

The SDK does NOT depend on any concrete logger (zerolog, zap, slog) so that
embedders can plug their own implementation. The provided NoopLogger has
zero overhead and is the default if Config.Logger is nil.

Field constructors mirror zerolog/zap API style (typed key/value). They
allocate only on call — this is acceptable because logging happens off
the hot path (reconnects, parse failures, occasional warnings); high-rate
events go through bbmet counters instead.

DESIGN NOTES:
  - Logger lives in internal/bblog to avoid import cycles between the root
    package, internal/rest and internal/ws. The root okx-style package
    re-exports the interface as okx.Logger via a type alias (see logger.go
    in the package root).
  - Field is a closed struct on purpose: no reflection inside the SDK.
*/

package bblog

// FieldKind enumerates supported field value types. Closed enum on purpose:
// adding a new kind requires updating every Logger implementation.
type FieldKind uint8

const (
	// FieldKindString — string value.
	FieldKindString FieldKind = iota
	// FieldKindInt — int64 value (covers all signed integers).
	FieldKindInt
	// FieldKindFloat — float64 value.
	FieldKindFloat
	// FieldKindBool — bool value.
	FieldKindBool
	// FieldKindError — error value.
	FieldKindError
)

// Field — a single typed log field.
type Field struct {
	Key    string
	Kind   FieldKind
	Str    string
	Int    int64
	Float  float64
	Bool   bool
	ErrVal error
}

// Str creates a string field.
func Str(key, value string) Field {
	return Field{Key: key, Kind: FieldKindString, Str: value}
}

// Int creates an int64 field. The variadic int-family helpers all funnel here.
func Int(key string, value int64) Field {
	return Field{Key: key, Kind: FieldKindInt, Int: value}
}

// Float creates a float64 field.
func Float(key string, value float64) Field {
	return Field{Key: key, Kind: FieldKindFloat, Float: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Kind: FieldKindBool, Bool: value}
}

// Err creates an error field with key "error". Returns a zero-value Field if
// err is nil so the caller can pass it unconditionally.
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Kind: FieldKindError}
	}
	return Field{Key: "error", Kind: FieldKindError, ErrVal: err}
}

// Logger is the SDK's logging facade. Implementations must be safe for
// concurrent use.
type Logger interface {
	// Debug logs at debug level. Used for verbose diagnostics; should be a
	// no-op in production unless explicitly enabled by the embedder.
	Debug(msg string, fields ...Field)
	// Info logs at info level. Used for lifecycle events (connect, login,
	// resync, shutdown). Should be infrequent.
	Info(msg string, fields ...Field)
	// Warn logs at warn level. Used for recoverable anomalies (parse failed,
	// reconnect with backoff, gap detected).
	Warn(msg string, fields ...Field)
	// Error logs at error level. Used for unrecoverable conditions inside a
	// single subscription/request that the caller should know about.
	Error(msg string, fields ...Field)
}

// noopLogger discards all log records. Zero size, zero overhead.
type noopLogger struct{}

func (noopLogger) Debug(string, ...Field) {}
func (noopLogger) Info(string, ...Field)  {}
func (noopLogger) Warn(string, ...Field)  {}
func (noopLogger) Error(string, ...Field) {}

// Noop returns a Logger that discards everything. Used when Config.Logger
// is nil.
func Noop() Logger { return noopLogger{} }
