package config

import (
	"context"
	"os"

	"ai-video/internal/pkg/tracing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.SugaredLogger

// Logger returns the application logger enriched with the trace identity stored
// in ctx. Startup/background callers without a trace continue using the base
// logger unchanged.
func Logger(ctx context.Context) *zap.SugaredLogger {
	logger := Log
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	span, ok := tracing.SpanFromContext(ctx)
	if !ok {
		return logger
	}
	return logger.With("trace_id", span.TraceID, "span_id", span.SpanID)
}

func InitLogger() {
	cfg := Cfg.Log

	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	fileWriter := &lumberjack.Logger{
		Filename:   cfg.Directory + "/app.log",
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true, // 轮转的旧日志 gzip 压缩，省磁盘
		LocalTime:  true, // 轮转文件名用本地时间，与业务时区一致
	}

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
		zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Log = logger.Sugar()
}
