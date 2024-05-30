package log

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LevelEnablerFunc func(Level) bool

type TeeOption struct {
	Out io.Writer
	LevelEnablerFunc
}

// NewTee writes multiple outputs based on log level
// https://pkg.go.dev/go.uber.org/zap#example-package-AdvancedConfiguration
func NewTee(tees []TeeOption, opts ...Option) *Logger {
	var cores []zapcore.Core
	for _, tee := range tees {
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.RFC3339TimeEncoder
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg),
			zapcore.AddSync(tee.Out),
			zap.LevelEnablerFunc(tee.LevelEnablerFunc),
		)
		cores = append(cores, core)
	}
	return &Logger{l: zap.New(zapcore.NewTee(cores...), opts...)}
}
