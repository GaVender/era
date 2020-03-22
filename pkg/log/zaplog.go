package log

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	*zap.Logger
}

func NewZapLogger(project string, isPrd bool, opt ...zap.Option) (ZapLogger, func()) {
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
	return ZapLogger{l}, func() {
		if err = l.Sync(); err != nil {
			panic("zap logger sync: " + err.Error())
		}
	}
}

func (z ZapLogger) Error(msg string) {
	z.With().Debug(msg)
}

func (z ZapLogger) Debugf(format string, v ...interface{}) {
	z.With().Debug(fmt.Sprintf(format, v))
}

func (z ZapLogger) Infof(format string, v ...interface{}) {
	z.With().Info(fmt.Sprintf(format, v))
}

func (z ZapLogger) Errorf(format string, v ...interface{}) {
	z.With().Error(fmt.Sprintf(format, v))
}

func (z ZapLogger) Panicf(format string, v ...interface{}) {
	z.With().Panic(fmt.Sprintf(format, v))
}

func (z ZapLogger) DebugField(msg string, fields ...zap.Field) {
	z.Logger.Debug(msg, fields...)
}

func (z ZapLogger) InfoField(msg string, fields ...zap.Field) {
	z.Logger.Info(msg, fields...)
}

func (z ZapLogger) ErrorField(msg string, fields ...zap.Field) {
	z.Logger.Error(msg, fields...)
}

func (z ZapLogger) PanicField(msg string, fields ...zap.Field) {
	z.Logger.Panic(msg, fields...)
}
