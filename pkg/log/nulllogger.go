package log

import "context"

type NullLogger struct{}

func (n NullLogger) Error(msg string)                                                   {}
func (n NullLogger) Print(v ...interface{})                                             {}
func (n NullLogger) Debugf(format string, v ...interface{})                             {}
func (n NullLogger) Infof(format string, v ...interface{})                              {}
func (n NullLogger) Errorf(format string, v ...interface{})                             {}
func (n NullLogger) Panicf(format string, v ...interface{})                             {}
func (n NullLogger) ContextDebugf(ctx context.Context, format string, v ...interface{}) {}
func (n NullLogger) ContextInfof(ctx context.Context, format string, v ...interface{})  {}
func (n NullLogger) ContextErrorf(ctx context.Context, format string, v ...interface{}) {}
func (n NullLogger) ContextPanicf(ctx context.Context, format string, v ...interface{}) {}

var _ Logger = NullLogger{}
