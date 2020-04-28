package log

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/GaVender/era/config"
)

type ZapLogger struct {
	*zap.Logger
}

func NewZapLogger(project string, isPrd bool, opt ...zap.Option) (*ZapLogger, func()) {
	var (
		c   zap.Config
		l   *zap.Logger
		err error
	)

	if isPrd {
		c = zap.NewProductionConfig()
		c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		c = zap.NewDevelopmentConfig()
	}

	l, err = c.Build(opt...)
	if err != nil {
		panic("zap logger init: " + err.Error())
	}

	l = l.Named(project).WithOptions(zap.AddCallerSkip(1))
	return &ZapLogger{l}, func() {
		if err = l.Sync(); err != nil {
			panic("zap logger sync: " + err.Error())
		}
	}
}

func (z *ZapLogger) WithContext(ctx context.Context) *ZapLogger {
	z.Logger = z.Logger.With(zap.String(config.TraceID, z.gerTraceID(ctx)))
	return z
}

func (z *ZapLogger) gerTraceID(ctx context.Context) string {
	var traceID string

	sp := opentracing.SpanFromContext(ctx)
	if sp != nil {
		if jaegerSpanContext, ok := sp.Context().(jaeger.SpanContext); ok {
			traceID = jaegerSpanContext.TraceID().String()
		}
	}

	return traceID
}

func (z *ZapLogger) Error(msg string) {
	z.With().Error(msg)
}

func (z *ZapLogger) Print(v ...interface{}) {
	z.With().Info(fmt.Sprint(v...))
}

func (z *ZapLogger) Debugf(format string, v ...interface{}) {
	z.With().Debug(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) Infof(format string, v ...interface{}) {
	z.With().Info(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) Errorf(format string, v ...interface{}) {
	z.With().Error(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) Panicf(format string, v ...interface{}) {
	z.With().Panic(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) ContextDebugf(ctx context.Context, format string, v ...interface{}) {
	z.With(zap.String(config.TraceID, z.gerTraceID(ctx))).Debug(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) ContextInfof(ctx context.Context, format string, v ...interface{}) {
	z.With(zap.String(config.TraceID, z.gerTraceID(ctx))).Info(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) ContextErrorf(ctx context.Context, format string, v ...interface{}) {
	z.With(zap.String(config.TraceID, z.gerTraceID(ctx))).Error(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) ContextPanicf(ctx context.Context, format string, v ...interface{}) {
	z.With(zap.String(config.TraceID, z.gerTraceID(ctx))).Panic(fmt.Sprintf(format, v...))
}

func (z *ZapLogger) DebugField(msg string, fields ...zap.Field) {
	z.Logger.Debug(msg, fields...)
}

func (z *ZapLogger) InfoField(msg string, fields ...zap.Field) {
	z.Logger.Info(msg, fields...)
}

func (z *ZapLogger) ErrorField(msg string, fields ...zap.Field) {
	z.Logger.Error(msg, fields...)
}

func (z *ZapLogger) PanicField(msg string, fields ...zap.Field) {
	z.Logger.Panic(msg, fields...)
}
