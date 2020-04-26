package log

type NullLogger struct{}

func (n NullLogger) Error(msg string)                       {}
func (n NullLogger) Print(v ...interface{})                 {}
func (n NullLogger) Debugf(format string, v ...interface{}) {}
func (n NullLogger) Infof(format string, v ...interface{})  {}
func (n NullLogger) Errorf(format string, v ...interface{}) {}
func (n NullLogger) Panicf(format string, v ...interface{}) {}
