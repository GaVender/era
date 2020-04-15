package log

type Logger interface {
	Error(msg string)
	Print(v ...interface{})

	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Panicf(format string, v ...interface{})
}
