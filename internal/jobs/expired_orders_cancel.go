package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
)

func StartExpiredOrdersCancelWorker(ctx context.Context, repo *repositories.OrderRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("Cancel Expired Offers task started with interval:", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Cancel Expired Offers task stopping...")
			return
		case <-ticker.C:
			n, err := repo.ExpirePendingOrders(ctx)
			if err != nil {
				slog.Error("Expired Orders Cancellation failed: ", err)
				continue
			}
			if n > 0 {
				slog.Info("%w expired orders has been cancelled.", n)
			}
		}
	}
}
