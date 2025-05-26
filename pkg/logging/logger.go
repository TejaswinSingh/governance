package logging

type Logger interface {
	Info(message string)
	Infof(format string, params ...any)
	InfoMap(keyVals map[string]any)
	Warn(message string)
	Warnf(format string, params ...any)
	WarnMap(keyVals map[string]any)
	Debug(message string)
	Debugf(format string, params ...any)
	DebugMap(keyVals map[string]any)
	ErrorNt(err error, message string)
	ErrorNtf(err error, format string, params ...any)
	ErrorNtMap(err error, keyVals map[string]any)
	Error(err error, message string)
	Errorf(err error, format string, params ...any)
	ErrorMap(err error, keyVals map[string]any)
	Fatal(err error, message string)
	Fatalf(err error, format string, params ...any)
	FatalMap(err error, keyVals map[string]any)
}
