package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/cache"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
)

func StartExpiredOrdersCancelWorker(ctx context.Context, repo *repositories.OrderRepository, c cache.Cache, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("Cancel Expired Offers task started with interval:", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Cancel Expired Offers task stopping...")
			return
		case <-ticker.C:
			expiredOrders, err := repo.ExpirePendingOrders(ctx)
			if err != nil {
				slog.Error("Expired Orders Cancellation failed: ", "error", err)
				continue
			}
			n := len(expiredOrders)
			if n == 0 {
				continue
			}

			keys := make([]string, 0, n*2)
			for _, o := range expiredOrders {
				keys = append(keys,
					cache.OrderDetailCacheKey(o.ID, o.UserID),
					cache.OrderListCacheKey(o.UserID),
				)
			}
			if err := c.Del(ctx, keys...); err != nil {
				slog.Debug("Job cache delete failed", "error", err)
			}
			slog.Info("Cancelled expired orders:", "count", n)
		}
	}
}
