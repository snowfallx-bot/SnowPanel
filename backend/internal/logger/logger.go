package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(appEnv string) (*zap.Logger, error) {
	if appEnv == "production" {
		return zap.NewProduction()
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

func Err(err error) zap.Field {
	return zap.Error(err)
}
