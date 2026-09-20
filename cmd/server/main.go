package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/auth"
	"github.com/Samandar-Komilov/qulpunoy/internal/cache"
	"github.com/Samandar-Komilov/qulpunoy/internal/config"
	"github.com/Samandar-Komilov/qulpunoy/internal/jobs"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/Samandar-Komilov/qulpunoy/internal/routers"
	"github.com/Samandar-Komilov/qulpunoy/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := config.InitLogger()
	slog.SetDefault(logger)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := config.Migrate(rootCtx, cfg.DSN()); err != nil {
		slog.Error("Failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	pool, err := config.NewPostgresPool(rootCtx, cfg.DSN())
	if err != nil {
		slog.Error("Failed to initialize database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("Database pool initialized")

	jwtManager := auth.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	redisCache, err := cache.NewRedisCache(rootCtx, cfg.RedisHost+":"+cfg.RedisPort, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		slog.Error("Failed to initialize redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()
	slog.Info("Redis cache initialized")

	userRepo := repositories.NewUserRepository(pool)
	productRepo := repositories.NewProductRepository(pool)
	orderRepo := repositories.NewOrderRepository(pool)

	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, jwtManager)
	productService := services.NewProductService(productRepo)
	orderService := services.NewOrderService(orderRepo, redisCache)

	userHandler := routers.NewUserHandler(userService)
	authHandler := routers.NewAuthHandler(authService)
	productHandler := routers.NewProductHandler(productService)
	orderHandler := routers.NewOrderHandler(orderService)

	r := routers.NewRouter(routers.Deps{
		JWT:            jwtManager,
		UserHandler:    userHandler,
		AuthHandler:    authHandler,
		ProductHandler: productHandler,
		OrderHandler:   orderHandler,
	})

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		slog.Info("Server starting", "port", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		jobs.StartExpiredOrdersCancelWorker(rootCtx, orderRepo, redisCache, 1*time.Minute)
	}()

	<-rootCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced shutdown", "error", err)
	}
	wg.Wait()
}
