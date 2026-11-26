package common

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/AnarManafov/dataharbor/app/config"
)

var Logger *zap.SugaredLogger

// initLoggerWithConfig is the main implementation
func initLoggerWithConfig(cfg *config.LoggingConfig) {
	if cfg == nil {
		// Provide a basic development logger if no config exists
		l, _ := zap.NewDevelopment(zap.WrapCore(newRedactingCore))
		Logger = l.Sugar()
		return
	}

	var cores []zapcore.Core

	// Add console logging if enabled
	if cfg.Console.Enabled {
		cores = append(cores, newRedactingCore(createConsoleCore(cfg)))
	}

	// Add file logging if enabled
	if cfg.File.Enabled {
		cores = append(cores, newRedactingCore(createFileCore(cfg)))
	}

	// If no cores are configured, use a basic development logger
	if len(cores) == 0 {
		l, _ := zap.NewDevelopment(zap.WrapCore(newRedactingCore))
		Logger = l.Sugar()
		return
	}

	// Combine all logging outputs into one logger. Each leaf core is wrapped
	// with newRedactingCore individually: a Tee's Write fans out without a
	// per-core level check, so wrapping the Tee would break per-output levels.
	core := zapcore.NewTee(cores...)
	Logger = zap.New(core).WithOptions(zap.WithCaller(true), zap.AddCallerSkip(1)).Sugar()
}

// DestroyLogger cleans up logger resources
func DestroyLogger() {
	Logger = nil
}

// concatTid adds transaction ID to log messages for request tracing
func concatTid(ctx *gin.Context, template string) string {
	if ctx != nil {
		tid := ctx.GetString("tid")
		if len(tid) > 0 {
			template = " tid: " + tid + " " + template
		}
	}

	return template
}

// Infof logs informational messages with request context for traceability
func Infof(ctx *gin.Context, template string, arg ...any) {
	Logger.Infof(concatTid(ctx, template), arg...)
}

// Errorf logs error messages with request context for traceability
func Errorf(ctx *gin.Context, template string, arg ...any) {
	Logger.Errorf(concatTid(ctx, template), arg...)
}

// Debugf logs debug messages with request context for traceability
func Debugf(ctx *gin.Context, template string, arg ...any) {
	Logger.Debugf(concatTid(ctx, template), arg...)
}

// Warnf logs warning messages with request context for traceability
func Warnf(ctx *gin.Context, template string, arg ...any) {
	Logger.Warnf(concatTid(ctx, template), arg...)
}

// createConsoleCore creates a console logger core
func createConsoleCore(cfg *config.LoggingConfig) zapcore.Core {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Console.Format == "json" {
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		encoder = zapcore.NewConsoleEncoder(config)
	}

	level := getZapLevel(cfg.Console.Level)
	if level == zapcore.InvalidLevel {
		level = getZapLevel(cfg.Level) // fallback to global level
	}

	return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
}

// createFileCore creates a file logger core with rotation
func createFileCore(cfg *config.LoggingConfig) zapcore.Core {
	// Use lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   cfg.File.Filename,
		MaxSize:    cfg.File.MaxSize,
		MaxBackups: cfg.File.MaxBackups,
		MaxAge:     cfg.File.MaxAge,
		Compress:   cfg.File.Compress,
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.File.Format == "json" {
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		encoder = zapcore.NewConsoleEncoder(config)
	}

	level := getZapLevel(cfg.File.Level)
	if level == zapcore.InvalidLevel {
		level = getZapLevel(cfg.Level) // fallback to global level
	}

	return zapcore.NewCore(encoder, zapcore.AddSync(lumberjackLogger), level)
}

// getZapLevel converts string level names to zap log levels
func getZapLevel(levelName string) zapcore.Level {
	switch levelName {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zapcore.InvalidLevel
	}
}

// InitLoggerFromViper initializes logger from viper configuration (deprecated)
// This function is kept for backward compatibility with tests
func InitLoggerFromViper() {
	// This is the old implementation - provide a basic development logger
	l, _ := zap.NewDevelopment(zap.WrapCore(newRedactingCore))
	Logger = l.Sugar()
}

// For backward compatibility with tests that don't pass config
func InitLogger(args ...*config.LoggingConfig) {
	if len(args) == 0 || args[0] == nil {
		// Backward compatibility mode - use development logger
		l, _ := zap.NewDevelopment(zap.WrapCore(newRedactingCore))
		Logger = l.Sugar()
		return
	}

	// New implementation with config
	initLoggerWithConfig(args[0])
}

// redactingCore is the defence-in-depth net of GH-82: it applies RedactText to
// every entry message and redacts string, []byte, error and Stringer fields
// (and, via a JSON round trip, reflected/array/object fields) before the
// wrapped core encodes them. Call sites should still redact explicitly with
// the Redact* helpers; this only catches what they miss.
//
// Wrap leaf cores (zapcore.NewCore results), not a Tee: Check consults only
// Enabled, and a Tee's Write does not filter per member level.
type redactingCore struct {
	zapcore.Core
}

func newRedactingCore(core zapcore.Core) zapcore.Core {
	if core == nil {
		return nil
	}
	if _, ok := core.(*redactingCore); ok {
		return core
	}
	return &redactingCore{Core: core}
}

func (c *redactingCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactingCore{Core: c.Core.With(redactFields(fields))}
}

func (c *redactingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *redactingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = RedactText(ent.Message)
	return c.Core.Write(ent, redactFields(fields))
}

// redactFields returns a redacted copy; the caller's slice is not modified.
func redactFields(fields []zapcore.Field) []zapcore.Field {
	if len(fields) == 0 {
		return fields
	}
	out := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		out[i] = redactField(f)
	}
	return out
}

func redactField(f zapcore.Field) zapcore.Field {
	switch f.Type {
	case zapcore.StringType:
		return redactedStringField(f.Key, f.String)
	case zapcore.ByteStringType:
		b, _ := f.Interface.([]byte)
		return redactedStringField(f.Key, string(b))
	case zapcore.ErrorType:
		// Flattened to a string: zap's native error encoding can also emit an
		// unredacted "errorVerbose" field.
		return redactedStringField(f.Key, safeString(f.Interface, func() string {
			return f.Interface.(error).Error()
		}))
	case zapcore.StringerType:
		return redactedStringField(f.Key, safeString(f.Interface, func() string {
			return f.Interface.(fmt.Stringer).String()
		}))
	case zapcore.ReflectType, zapcore.ArrayMarshalerType, zapcore.ObjectMarshalerType:
		return redactCompositeField(f)
	default:
		return f
	}
}

func redactedStringField(key, val string) zapcore.Field {
	if isSensitiveKey(key) && !keepValue(val) {
		val = RedactedPlaceholder
	} else {
		val = RedactText(val)
	}
	return zapcore.Field{Key: key, Type: zapcore.StringType, String: val}
}

// safeString calls fn, mirroring zap's handling of panicking String/Error
// methods (nil pointer receivers print "<nil>").
func safeString(v any, fn func() string) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
				s = "<nil>"
				return
			}
			s = fmt.Sprintf("<PANIC=%v>", r)
		}
	}()
	return fn()
}

// redactCompositeField renders a reflected/array/object field through zap's
// map encoder and JSON, redacts it like a JSON body and re-emits it as raw
// JSON, so the output shape stays a nested value.
func redactCompositeField(f zapcore.Field) (out zapcore.Field) {
	if isStrongKey(f.Key) {
		return zapcore.Field{Key: f.Key, Type: zapcore.StringType, String: RedactedPlaceholder}
	}
	unloggable := func(why string) zapcore.Field {
		return zapcore.Field{Key: f.Key, Type: zapcore.StringType, String: "[unloggable: " + RedactText(why) + "]"}
	}
	defer func() {
		if r := recover(); r != nil {
			out = unloggable(fmt.Sprintf("panic: %v", r))
		}
	}()
	enc := zapcore.NewMapObjectEncoder()
	f.AddTo(enc)
	v, ok := enc.Fields[f.Key]
	if !ok {
		return unloggable("encoding failed")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return unloggable(err.Error())
	}
	red, ok := redactJSONDocument(b, true)
	if !ok {
		return unloggable("encoding failed")
	}
	return zapcore.Field{Key: f.Key, Type: zapcore.ReflectType, Interface: json.RawMessage(red)}
}
