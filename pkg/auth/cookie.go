package auth

import (
	"encoding/base64"
	"net/http"

	"github.com/Resolyte-official/service-governance/pkg/cryptoutil"
	"github.com/Resolyte-official/service-governance/pkg/errors"
)

// returns silently if anything goes wrong, following http.SetCookie behaviour
func WriteEncryptedCookie(w http.ResponseWriter, cookie *http.Cookie, secretKey []byte) {
	// encrypt cookie
	sealed, err := cryptoutil.Encrypt([]byte(cookie.Value), secretKey)
	if err != nil {
		return
	}

	// for preserving special characters
	// that might be generated during encryption
	cookie.Value = base64.URLEncoding.EncodeToString(sealed)

	if len(cookie.String()) > 4096 {
		return
	}

	http.SetCookie(w, cookie)
}

func ReadEncryptedCookie(r *http.Request, name string, secretKey []byte) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil || cookie.Valid() != nil {
		return "", errors.ErrUnauthorized
	}

	// base64 decode
	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", errors.ErrUnauthorized.WithInternalErr(err)
	}

	// decrypt cookie
	cookieVal, err := cryptoutil.Decrypt(value, secretKey)
	if err != nil {
		return "", errors.ErrUnauthorized.WithInternalErr(err)
	}

	return string(cookieVal), nil
}
