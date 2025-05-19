package common

import (
	"os"

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
		l, _ := zap.NewDevelopment()
		Logger = l.Sugar()
		return
	}

	var cores []zapcore.Core

	// Add console logging if enabled
	if cfg.Console.Enabled {
		cores = append(cores, createConsoleCore(cfg))
	}

	// Add file logging if enabled
	if cfg.File.Enabled {
		cores = append(cores, createFileCore(cfg))
	}

	// If no cores are configured, use a basic development logger
	if len(cores) == 0 {
		l, _ := zap.NewDevelopment()
		Logger = l.Sugar()
		return
	}

	// Combine all logging outputs into one logger
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
func Infof(ctx *gin.Context, template string, arg ...interface{}) {
	Logger.Infof(concatTid(ctx, template), arg...)
}

// Errorf logs error messages with request context for traceability
func Errorf(ctx *gin.Context, template string, arg ...interface{}) {
	Logger.Errorf(concatTid(ctx, template), arg...)
}

// Debugf logs debug messages with request context for traceability
func Debugf(ctx *gin.Context, template string, arg ...interface{}) {
	Logger.Debugf(concatTid(ctx, template), arg...)
}

// Warnf logs warning messages with request context for traceability
func Warnf(ctx *gin.Context, template string, arg ...interface{}) {
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
	l, _ := zap.NewDevelopment()
	Logger = l.Sugar()
}

// For backward compatibility with tests that don't pass config
func InitLogger(args ...*config.LoggingConfig) {
	if len(args) == 0 || args[0] == nil {
		// Backward compatibility mode - use development logger
		l, _ := zap.NewDevelopment()
		Logger = l.Sugar()
		return
	}

	// New implementation with config
	initLoggerWithConfig(args[0])
}
