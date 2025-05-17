package transport

import (
	"net/http"
	"reflect"
)

type ResponseWriter struct {
	http.ResponseWriter
	status    int
	Err       string
	RequestID string
	UserID    string
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

type LogEntry struct {
	RequestID string `key:"REQUEST_ID"`
	UserID    string `key:"USER_ID"`
	Transport string `key:"TRANSPORT"`
	Method    string `key:"METHOD"`
	Path      string `key:"PATH"`
	RPC       string `key:"RPC"`
	Status    int    `key:"STATUS"`
	Error     string `key:"ERROR"`
	Duration  string `key:"DURATION"`
}

func (e *LogEntry) Log() {
	if e.RequestID == "" {
		e.RequestID = "Missing"
	}

	v := reflect.ValueOf(*e)
	t := v.Type()

	var keyvals []any
	for i, fieldType := range reflect.VisibleFields(t) {
		fieldVal := v.Field(i)
		tag := fieldType.Tag.Get("key")

		switch fieldVal.Kind() {
		case reflect.String:
			if val := fieldVal.String(); val != "" {
				keyvals = append(keyvals, tag, val)
			}
		case reflect.Int:
			if val := fieldVal.Int(); val != 0 {
				keyvals = append(keyvals, tag, val)
			}
		}
	}

	if len(keyvals) > 0 {
		_ = logger.Log(keyvals...)
	}
}
