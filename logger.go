/*
FILE: logger.go

DESCRIPTION:
Public re-export of the Logger interface and field constructors. The
underlying types live in internal/bblog so the rest/ws/orderbook
sub-packages can import them without taking a dependency on the root
(which itself depends on these packages — circular import otherwise).

The SDK ships only a NoopLogger by default. Embedders are expected to
adapt their own logger (zerolog, zap, slog, log/slog) to bybit.Logger
once and pass it via Config.Logger.
*/

package bybit

import "github.com/tonymontanov/go-bybit/internal/bblog"

// Logger is the SDK logging facade. See internal/bblog for the full contract.
type Logger = bblog.Logger

// Field is a typed key/value pair used in log entries.
type Field = bblog.Field

// FieldKind enumerates supported field value types.
type FieldKind = bblog.FieldKind

// Field-kind aliases.
const (
	// FieldKindString — string value.
	FieldKindString = bblog.FieldKindString
	// FieldKindInt — int64 value.
	FieldKindInt = bblog.FieldKindInt
	// FieldKindFloat — float64 value.
	FieldKindFloat = bblog.FieldKindFloat
	// FieldKindBool — bool value.
	FieldKindBool = bblog.FieldKindBool
	// FieldKindError — error value.
	FieldKindError = bblog.FieldKindError
)

// Str is a shortcut for a string field.
func Str(key, value string) Field { return bblog.Str(key, value) }

// Int is a shortcut for an int64 field.
func Int(key string, value int64) Field { return bblog.Int(key, value) }

// Float is a shortcut for a float64 field.
func Float(key string, value float64) Field { return bblog.Float(key, value) }

// Bool is a shortcut for a bool field.
func Bool(key string, value bool) Field { return bblog.Bool(key, value) }

// Err is a shortcut for an error field with key "error".
func Err(err error) Field { return bblog.Err(err) }

// NoopLogger returns a Logger that discards every record.
func NoopLogger() Logger { return bblog.Noop() }
