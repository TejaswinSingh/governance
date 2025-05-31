package logging

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type TextLogWriter struct {
	o    io.Writer
	minL Level
	mu   sync.Mutex
}

func NewTextLogWriter(out io.Writer, minLevel Level) *TextLogWriter {
	return &TextLogWriter{
		o:    out,
		minL: minLevel,
	}
}

func (tl *TextLogWriter) print(level Level, err error, message string, context string) (int, error) {

	if level < tl.minL {
		return 0, nil
	}

	t := time.Now().Format(time.RFC3339)

	var line string
	if err != nil {
		line = fmt.Sprintf("GOVERNANCE | %s %v[%s]: error=%s message=%s\n", t, level, context, err.Error(), message)
	} else {
		line = fmt.Sprintf("GOVERNANCE | %s %v[%s]: %s\n", t, level, context, message)
	}

	if level >= ERROR {
		line += fmt.Sprintf("trace: %s", string(debug.Stack()))
	}

	line = colorizeLog(line, getLevelColor(level))

	tl.mu.Lock()

	defer tl.mu.Unlock()

	return tl.o.Write([]byte(line))
}

func (tl *TextLogWriter) printMap(level Level, err error, keyVals map[string]any, context string) (int, error) {

	if level < tl.minL {
		return 0, nil
	}

	t := time.Now().Format(time.RFC3339)

	var fields string
	if len(keyVals) > 0 {
		var parts []string
		for k, v := range keyVals {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fields = strings.Join(parts, " ")
	}

	var line string
	if err != nil {
		line = fmt.Sprintf("GOVERNANCE | %s %v[%s]: error=%s %s\n", t, level, context, err.Error(), fields)
	} else {
		line = fmt.Sprintf("GOVERNANCE | %s %v[%s]: %s\n", t, level, context, fields)
	}

	if level >= ERROR {
		line += fmt.Sprintf("trace: %s", string(debug.Stack()))
	}

	line = colorizeLog(line, getLevelColor(level))

	tl.mu.Lock()

	defer tl.mu.Unlock()

	return tl.o.Write([]byte(line))
}
