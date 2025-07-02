package log

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *zap.Logger
	once       sync.Once
	instanceID string
)

func InitLogger(service string) {
	once.Do(func() {
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.MessageKey = "msg"
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}

		l, err := cfg.Build()
		if err != nil {
			panic(err)
		}
		logger = l.With(zap.String("service", service), zap.String("instance_id", getInstanceID()))
	})
}

func getInstanceID() string {
	if instanceID != "" {
		return instanceID
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	instanceID = hostname
	return instanceID
}

func L() *zap.Logger {
	if logger == nil {
		panic("logger not initialized")
	}
	return logger
}
