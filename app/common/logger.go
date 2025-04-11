package common

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.SugaredLogger

// InitLogger initializes the application's logging infrastructure based on configuration
// from viper. Supports both file and console logging with different log levels.
// Falls back to a development console logger if no configuration is found.
func InitLogger() {
	// Use viper to support both config files and environment variables
	viper.SetEnvPrefix("LOGGER")
	viper.AutomaticEnv()

	loggerList := viper.GetStringMap("logger")

	if len(loggerList) == 0 {
		// Provide a basic development logger if no config exists
		l, _ := zap.NewDevelopment()
		Logger = l.Sugar()
		return
	}

	// Create cores for each configured logger (file, console, etc.)
	var cList []zapcore.Core
	for loggerName := range loggerList {
		c, err := parseLogger(loggerName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse logger %s failed; err: %s\n", loggerName, err)
			continue
		}

		cList = append(cList, c)
	}

	if len(cList) == 0 {
		Logger = nil
		return
	}

	// Combine all logging outputs into one logger
	core := zapcore.NewTee(cList...)
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

// parseLogger creates the appropriate zapcore.Core based on logger configuration
func parseLogger(name string) (zapcore.Core, error) {
	cnf := viper.Sub("logger." + name)
	driverName := cnf.GetString("driver")
	switch driverName {
	case "console":
		return parseConsoleConf(cnf), nil
	case "file":
		return parseFileConf(cnf), nil
	default:
		return nil, fmt.Errorf("invalid logger driver name: \"%s\"", driverName)
	}
}

// parseFileConf creates a file logger with rotation support
func parseFileConf(cnf *viper.Viper) zapcore.Core {
	// Use lumberjack for log rotation to prevent log files from growing indefinitely
	lumberjackLogger := &lumberjack.Logger{
		Filename:   cnf.GetString("filename"),
		MaxSize:    cnf.GetInt("maxsize"),
		MaxBackups: cnf.GetInt("maxbackups"),
		MaxAge:     cnf.GetInt("maxage"),
		Compress:   cnf.GetBool("compress"),
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(fileEncoder, zapcore.AddSync(lumberjackLogger), getZapLevel(cnf.GetString("level")))
}

// parseConsoleConf creates a console logger for terminal output
func parseConsoleConf(cnf *viper.Viper) zapcore.Core {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(config)
	return zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), getZapLevel(cnf.GetString("level")))
}

// getZapLevel converts string level names to zap log levels
func getZapLevel(levelName string) zapcore.Level {
	switch levelName {
	case "info":
		return zap.InfoLevel
	default:
		return zap.DebugLevel
	}
}
