package logging

import (
	"fmt"
	"strings"
)

type LevelColor string

var (
	CYAN   LevelColor = "\033[36m"
	GREEN  LevelColor = "\033[32m"
	YELLOW LevelColor = "\033[33m"
	RED    LevelColor = "\033[31m"
	RESET  LevelColor = "\033[0m"
)

func getLevelColor(level Level) LevelColor {
	switch level {
	case DEBUG:
		return CYAN
	case INFO:
		return GREEN
	case WARN:
		return YELLOW
	case ERROR, ERROR_NO_TRACE, FATAL:
		return RED
	default:
		return RESET
	}
}

func colorizeLog(logText string, color LevelColor) string {
	lines := strings.Split(logText, "\n")
	for i, line := range lines {
		lines[i] = fmt.Sprintf("%s%s%s", color, line, RESET)
	}
	return strings.Join(lines, "\n")
}
