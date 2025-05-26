package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"runtime/debug"
	"sync"
	"time"
)

type JsonLogWriter struct {
	o    io.Writer
	minL Level
	mu   sync.Mutex
}

func NewJsonLogWriter(out io.Writer, minLevel Level) *JsonLogWriter {
	return &JsonLogWriter{
		minL: minLevel,
		o:    out,
	}
}

func (jl *JsonLogWriter) print(level Level, err error, message string, context string) (int, error) {

	if level < jl.minL {
		return 0, nil
	}

	t := time.Now().Format(time.RFC3339)

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	aux := struct {
		Service string `json:"service"`
		Level   string `json:"level"`
		Time    string `json:"time"`
		Context string `json:"context"`
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
		Trace   string `json:"trace,omitempty"`
	}{
		Service: "GOVERNANCE",
		Level:   level.String(),
		Time:    t,
		Context: context,
		Message: message,
		Error:   errMsg,
	}

	if level >= ERROR {
		aux.Trace = string(debug.Stack())
	}

	var line []byte

	line, err = json.Marshal(aux)

	if err != nil {
		errMsg := fmt.Sprintf("%s (%s)", "unable to marshal log message", err.Error())
		line = []byte(fmt.Sprintf("GOVERNANCE | %s %v[%s]: %s", t, ERROR, "JsonLogger", errMsg))
	}

	jl.mu.Lock()

	defer jl.mu.Unlock()

	return jl.o.Write(append(line, '\n'))
}

func (jl *JsonLogWriter) printMap(level Level, err error, keyVals map[string]any, context string) (int, error) {

	if level < jl.minL {
		return 0, nil
	}

	t := time.Now().Format(time.RFC3339)

	aux := map[string]any{
		"service": "GOVERNANCE",
		"level":   level.String(),
		"time":    t,
		"context": context,
	}

	if err != nil {
		aux["error"] = err.Error()
	}
	if len(keyVals) > 0 {
		maps.Copy(aux, keyVals)
	}
	if level >= ERROR {
		aux["trace"] = string(debug.Stack())
	}

	var line []byte

	line, err = json.Marshal(aux)

	if err != nil {
		errMsg := fmt.Sprintf("%s (%s)", "unable to marshal log message", err.Error())
		line = []byte(fmt.Sprintf("GOVERNANCE | %s %v[%s]: %s", t, ERROR, "JsonLogger", errMsg))
	}

	jl.mu.Lock()

	defer jl.mu.Unlock()

	return jl.o.Write(append(line, '\n'))
}
