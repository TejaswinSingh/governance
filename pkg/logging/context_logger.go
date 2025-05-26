package logging

import (
	"fmt"
	"os"
)

type ContextLogger struct {
	lw      LogWriter
	Context string
}

func NewContextLogger(lw LogWriter, context string) *ContextLogger {
	return &ContextLogger{
		lw:      lw,
		Context: context,
	}
}

func (cl *ContextLogger) Info(message string) {
	cl.lw.print(INFO, nil, message, cl.Context)
}

func (cl *ContextLogger) Infof(format string, params ...any) {
	cl.lw.print(INFO, nil, fmt.Sprintf(format, params...), cl.Context)
}

func (cl *ContextLogger) InfoMap(keyVals map[string]any) {
	cl.lw.printMap(INFO, nil, keyVals, cl.Context)
}

func (cl *ContextLogger) Warn(message string) {
	cl.lw.print(WARN, nil, message, cl.Context)
}

func (cl *ContextLogger) Warnf(format string, params ...any) {
	cl.lw.print(WARN, nil, fmt.Sprintf(format, params...), cl.Context)
}

func (cl *ContextLogger) WarnMap(keyVals map[string]any) {
	cl.lw.printMap(WARN, nil, keyVals, cl.Context)
}

func (cl *ContextLogger) Debug(message string) {
	cl.lw.print(DEBUG, nil, message, cl.Context)
}

func (cl *ContextLogger) Debugf(format string, params ...any) {
	cl.lw.print(DEBUG, nil, fmt.Sprintf(format, params...), cl.Context)
}

func (cl *ContextLogger) DebugMap(keyVals map[string]any) {
	cl.lw.printMap(DEBUG, nil, keyVals, cl.Context)
}

func (cl *ContextLogger) ErrorNt(err error, message string) {
	cl.lw.print(ERROR_NO_TRACE, err, message, cl.Context)
}

func (cl *ContextLogger) ErrorNtf(err error, format string, params ...any) {
	cl.lw.print(ERROR_NO_TRACE, err, fmt.Sprintf(format, params...), cl.Context)
}

func (cl *ContextLogger) ErrorNtMap(err error, keyVals map[string]any) {
	cl.lw.printMap(ERROR_NO_TRACE, err, keyVals, cl.Context)
}

func (cl *ContextLogger) Error(err error, message string) {
	cl.lw.print(ERROR, err, message, cl.Context)
}

func (cl *ContextLogger) Errorf(err error, format string, params ...any) {
	cl.lw.print(ERROR, err, fmt.Sprintf(format, params...), cl.Context)
}

func (cl *ContextLogger) ErrorMap(err error, keyVals map[string]any) {
	cl.lw.printMap(ERROR, err, keyVals, cl.Context)
}

func (cl *ContextLogger) Fatal(err error, message string) {
	cl.lw.print(FATAL, err, message, cl.Context)
	os.Exit(1)
}

func (cl *ContextLogger) Fatalf(err error, format string, params ...any) {
	cl.lw.print(FATAL, err, fmt.Sprintf(format, params...), cl.Context)
	os.Exit(1)
}

func (cl *ContextLogger) FatalMap(err error, keyVals map[string]any) {
	cl.lw.printMap(FATAL, err, keyVals, cl.Context)
	os.Exit(1)
}
