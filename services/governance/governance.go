package governance

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/log"
)

type governanceService struct{}

func NewService() Service { return &governanceService{} }

func (w *governanceService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
