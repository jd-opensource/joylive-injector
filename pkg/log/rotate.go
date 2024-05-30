package log

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	"io"
	"strings"
	"time"
)

type RotateConfig struct {
	// shared configuration
	Filename string // Full file name
	MaxAge   int    // Maximum number of days to keep old log files

	// Configuration by time rotation
	RotationTime time.Duration // Log file rotation time

	// Rotate configuration by size
	MaxSize    int  // Maximum log file size (MB)
	MaxBackups int  // Maximum number of log files to keep
	Compress   bool // Whether to compress and archive log files
	LocalTime  bool // Whether to use local time, default UTC time
}

// NewProductionRotateByTime Create an io.Writer that rotates by time
func NewProductionRotateByTime(filename string) io.Writer {
	return NewRotateByTime(NewProductionRotateConfig(filename))
}

// NewProductionRotateBySize Create an io.Writer that rotates by size
func NewProductionRotateBySize(filename string) io.Writer {
	return NewRotateBySize(NewProductionRotateConfig(filename))
}

func NewProductionRotateConfig(filename string) *RotateConfig {
	return &RotateConfig{
		Filename:     filename,
		MaxAge:       30,             // Logs retained for 30 days
		RotationTime: time.Hour * 24, // Rotates every 24 hours
		MaxSize:      100,            // 100M
		MaxBackups:   100,
		Compress:     true,
		LocalTime:    false,
	}
}

func NewRotateByTime(cfg *RotateConfig) io.Writer {
	opts := []rotatelogs.Option{
		rotatelogs.WithMaxAge(time.Duration(cfg.MaxAge) * time.Hour * 24),
		rotatelogs.WithRotationTime(cfg.RotationTime),
		rotatelogs.WithLinkName(cfg.Filename),
	}
	if !cfg.LocalTime {
		rotatelogs.WithClock(rotatelogs.UTC)
	}
	filename := strings.SplitN(cfg.Filename, ".", 2)
	l, _ := rotatelogs.New(
		filename[0]+".%Y-%m-%d-%H-%M-%S."+filename[1],
		opts...,
	)
	return l
}

func NewRotateBySize(cfg *RotateConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}
}
