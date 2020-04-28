package log

import "context"

type Logger interface {
	Error(msg string)
	Print(v ...interface{})

	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Panicf(format string, v ...interface{})

	ContextDebugf(ctx context.Context, format string, v ...interface{})
	ContextInfof(ctx context.Context, format string, v ...interface{})
	ContextErrorf(ctx context.Context, format string, v ...interface{})
	ContextPanicf(ctx context.Context, format string, v ...interface{})
}
