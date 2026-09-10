// No auth, no DB: probes the process only
package health

import (
	"context"

	gen "project/internal/gen/api"
)

// No deps: adding any would change what healthy means
type Service struct{}

func NewService() *Service { return &Service{} }

func (*Service) GetHealth(context.Context, *gen.GetHealthRequest) (*gen.GetHealthResponse, error) {
	return &gen.GetHealthResponse{Status: "ok"}, nil
}
