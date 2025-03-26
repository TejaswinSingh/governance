package governance

import (
	"context"
	"net/http"
	"os"

	"github.com/TejaswinSingh/version0/services/governance/auth"
	"github.com/TejaswinSingh/version0/services/governance/errors"
	"github.com/go-kit/log"
)

type governanceService struct {
	secretKey string
}

func NewService(secretKey string) Service {
	return &governanceService{secretKey: secretKey}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, *errors.ServiceError) {
	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, username, password string) (string, *errors.ServiceError) {
	if username == "" || password == "" {
		return "", errors.ErrMissingCreds
	}
	// TODO: implement proper credentials check
	if username == "admin" && password == "admin" {
		tokenString, err := auth.CreateToken(username, s.secretKey)
		if err != nil {
			return "", err
		}
		return tokenString, nil
	} else {
		return "", errors.ErrInvalidCreds
	}
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
