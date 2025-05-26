package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/Resolyte-official/service-governance/pkg/logging"
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
	RequestID string `key:"request_id"`
	UserID    string `key:"user_id"`
	Transport string `key:"transport"`
	Method    string `key:"method"`
	Path      string `key:"path"`
	RPC       string `key:"rpc"`
	Status    int    `key:"status"`
	Error     string `key:"error"`
	Duration  string `key:"duration"`
}

func (e *LogEntry) Log(logger logging.Logger) {
	if e.RequestID == "" {
		e.RequestID = "Missing"
	}

	v := reflect.ValueOf(*e)
	t := v.Type()

	keyVals := make(map[string]any)
	for i, fieldType := range reflect.VisibleFields(t) {
		fieldVal := v.Field(i)
		tag := fieldType.Tag.Get("key")

		switch fieldVal.Kind() {
		case reflect.String:
			val := fieldVal.String()
			// skip 500 errors, they are logged below explicitly
			if val != "" && !(tag == "error" && e.Status >= 500) {
				keyVals[tag] = val
			}
		case reflect.Int:
			if val := fieldVal.Int(); val != 0 {
				keyVals[tag] = val
			}
		}
	}
	if e.Status >= 500 {
		logger.ErrorMap(errors.New(e.Error), keyVals)
	} else {
		logger.InfoMap(keyVals)
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
