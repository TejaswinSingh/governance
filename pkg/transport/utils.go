package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/golang-jwt/jwt/v5"
)

type ResponseWriter struct {
	http.ResponseWriter
	status int
	Err    string
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
			// skip error field, it is logged below explicitly
			if val != "" && tag != "error" {
				keyVals[tag] = val
			}
		case reflect.Int:
			if val := fieldVal.Int(); val != 0 {
				keyVals[tag] = val
			}
		}
	}
	if e.Status <= 300 {
		logger.InfoMap(keyVals)
	} else if e.Status >= 400 && e.Status < 500 { // no stack trace for 400 errors
		logger.ErrorNtMap(errors.New(e.Error), keyVals)
	} else { // 500 errors
		logger.ErrorMap(errors.New(e.Error), keyVals)
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
		Role: getStringClaim(m, "role"),
		ParentClaims: &pb.ParentClaims{
			LinkedStudents:    getStringArrayClaim(m, "parent_claims", "linked_students"),
			DelegatedStudents: getStringArrayClaim(m, "parent_claims", "delegated_students"),
		},
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

func convertAnyToStringSlice(input []any) []string {
	var result []string
	for _, v := range input {
		result = append(result, fmt.Sprintf("%s", v))
	}
	return result
}

func getStringArrayClaim(m jwt.MapClaims, key, subkey string) []string {
	if submap, ok := m[key]; ok {
		if smap, ok := submap.(map[string]any); ok {
			if val, ok := smap[subkey]; ok {
				if s, ok := val.([]any); ok {
					return convertAnyToStringSlice(s)
				}
			}
		}
	}
	return nil
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
