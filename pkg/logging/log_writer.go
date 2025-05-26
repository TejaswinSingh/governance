package logging

type LogWriter interface {
	print(level Level, err error, message string, context string) (int, error)
	printMap(level Level, err error, keyVals map[string]any, context string) (int, error)
}
