package helpers

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func Logger(withFields ...zap.Field) *zap.Logger {
	zap.NewDevelopmentConfig()
	jsonEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	})
	core := zapcore.NewCore(jsonEncoder, os.Stdout, zap.LevelEnablerFunc(func(zapcore.Level) bool { return true }))
	return zap.New(core)
}
