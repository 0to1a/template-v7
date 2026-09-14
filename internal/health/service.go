// The route probes the process only; the cron probes the database
package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	gen "project/internal/gen/api"
)

// The pool is the cron's dependency only — GetHealth must stay dep-free, or
// adding any would change what a healthy process means.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (*Service) GetHealth(context.Context, *gen.GetHealthRequest) (*gen.GetHealthResponse, error) {
	return &gen.GetHealthResponse{Status: "ok"}, nil
}

// Bounded so a hung pool surfaces as an error instead of stalling the job
// past its own tick. Returning err is enough: RunCrons logs it.
//
// A failed ping means every pooled connection is suspect (network drop, server
// restart), so Reset drops them all; the pool stays open and redials lazily on
// the next query. Checked-out connections close when they are returned, so no
// in-flight request is cut off.
func (s *Service) CheckDatabase(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		s.pool.Reset()
		return err
	}
	return nil
}
