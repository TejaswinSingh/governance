package transport

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/Resolyte-official/service-governance/pkg/pb"
	"github.com/golang-jwt/jwt/v5"
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

func toPBClaims(m jwt.MapClaims) *pb.Claims {
	return &pb.Claims{
		Sub:  getStringClaim(m, "sub"),
		Aud:  getStringClaim(m, "aud"),
		Iss:  getStringClaim(m, "iss"),
		Iat:  getInt64Claim(m, "iat"),
		Exp:  getInt64Claim(m, "exp"),
		Type: getStringClaim(m, "type"),
	}
}

func getStringClaim(m jwt.MapClaims, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64Claim(m jwt.MapClaims, key string) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case json.Number:
			i, _ := v.Int64()
			return i
		}
	}
	return 0
}
