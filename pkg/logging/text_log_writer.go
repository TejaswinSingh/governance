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
		line = fmt.Sprintf("%sGOVERNANCE | %s %v[%s]: error=%s message=%s%s\n", getLevelColor(level), t, level, context, err.Error(), message, RESET)
	} else {
		line = fmt.Sprintf("%sGOVERNANCE | %s %v[%s]: %s%s\n", getLevelColor(level), t, level, context, message, RESET)
	}

	if level >= ERROR {
		line += fmt.Sprintf("%strace: %s", getLevelColor(ERROR), colorizeDebugStack(string(debug.Stack())))
	}

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
		line = fmt.Sprintf("%sGOVERNANCE | %s %v[%s]: error=%s %s%s\n", getLevelColor(level), t, level, context, err.Error(), fields, RESET)
	} else {
		line = fmt.Sprintf("%sGOVERNANCE | %s %v[%s]: %s%s\n", getLevelColor(level), t, level, context, fields, RESET)
	}

	if level >= ERROR {
		line += fmt.Sprintf("%strace: %s", getLevelColor(ERROR), colorizeDebugStack(string(debug.Stack())))
	}

	tl.mu.Lock()

	defer tl.mu.Unlock()

	return tl.o.Write([]byte(line))
}
