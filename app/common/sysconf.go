package common

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GetLogger returns the global logger
func GetLogger() *zap.SugaredLogger {
	// Return logger if already initialized
	if Logger != nil {
		return Logger
	}

	// Otherwise initialize a basic logger
	logger, _ := newProductionLogger(zap.NewProductionConfig())
	Logger = logger.Sugar()
	return Logger
}

// newProductionLogger builds cfg with the redacting core around the leaf
// (encoder) core and cfg's sampler, if any, outermost. Passing
// zap.WrapCore(newRedactingCore) to cfg.Build would wrap the sampler from the
// outside, and since redactingCore.Check adds itself to the entry without
// consulting the wrapped core's Check, sampling would be silently disabled.
func newProductionLogger(cfg zap.Config) (*zap.Logger, error) {
	sampling := cfg.Sampling
	cfg.Sampling = nil
	return cfg.Build(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		c = newRedactingCore(c)
		if sampling != nil {
			c = zapcore.NewSamplerWithOptions(c, time.Second, sampling.Initial, sampling.Thereafter)
		}
		return c
	}))
}
