package governance

import (
	"context"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
}
