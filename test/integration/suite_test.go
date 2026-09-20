package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Samandar-Komilov/qulpunoy/internal/auth"
	"github.com/Samandar-Komilov/qulpunoy/internal/cache"
	"github.com/Samandar-Komilov/qulpunoy/internal/config"
	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/Samandar-Komilov/qulpunoy/internal/routers"
	"github.com/Samandar-Komilov/qulpunoy/internal/services"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type APISuite struct {
	suite.Suite
	pool        *pgxpool.Pool
	rdb         *redis.Client
	server      *httptest.Server
	client      *http.Client
	userRepo    *repositories.UserRepository
	productRepo *repositories.ProductRepository
	orderRepo   *repositories.OrderRepository
}

func TestOrderAPI(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (s *APISuite) SetupSuite() {
	cfg, err := config.Load("../../.env", ".env")
	s.Require().NoError(err)

	ctx := context.Background()
	s.Require().NoError(config.Migrate(ctx, cfg.DSN()))

	pool, err := config.NewPostgresPool(ctx, cfg.DSN())
	s.Require().NoError(err)
	s.pool = pool

	redisCache, err := cache.NewRedisCache(ctx, cfg.RedisHost+":"+cfg.RedisPort, cfg.RedisPassword, cfg.RedisDB)
	s.Require().NoError(err)

	s.rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	s.Require().NoError(s.rdb.Ping(ctx).Err())

	jwtManager := auth.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	s.userRepo = repositories.NewUserRepository(pool)
	s.productRepo = repositories.NewProductRepository(pool)
	s.orderRepo = repositories.NewOrderRepository(pool)

	userService := services.NewUserService(s.userRepo)
	authService := services.NewAuthService(s.userRepo, jwtManager)
	productService := services.NewProductService(s.productRepo)
	orderService := services.NewOrderService(s.orderRepo, redisCache)

	userHandler := routers.NewUserHandler(userService)
	authHandler := routers.NewAuthHandler(authService)
	productHandler := routers.NewProductHandler(productService)
	orderHandler := routers.NewOrderHandler(orderService)

	router := routers.NewRouter(routers.Deps{
		JWT:            jwtManager,
		UserHandler:    userHandler,
		AuthHandler:    authHandler,
		ProductHandler: productHandler,
		OrderHandler:   orderHandler,
	})

	s.server = httptest.NewServer(router)
	s.client = s.server.Client()
	gofakeit.Seed(0)
}

func (s *APISuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
	s.truncateAll()
	if s.rdb != nil {
		_ = s.rdb.Close()
	}
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *APISuite) SetupTest() {
	s.truncateAll()
	s.flushRedis()
}

func (s *APISuite) truncateAll() {
	_, err := s.pool.Exec(context.Background(),
		`TRUNCATE order_items, orders, products, users RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
}

func (s *APISuite) flushRedis() {
	s.Require().NoError(s.rdb.FlushDB(context.Background()).Err())
}

func (s *APISuite) do(method, path, token string, body any) (*http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		s.Require().NoError(err)
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, s.server.URL+path, bodyReader)
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.client.Do(req)
	s.Require().NoError(err)

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	return resp, respBody
}

func (s *APISuite) registerUser() (string, string) {
	username := gofakeit.Username() + fmt.Sprintf("%d", gofakeit.Number(1000, 9999))
	password := gofakeit.Password(true, true, true, false, false, 12)

	payload := map[string]string{
		"username": username,
		"password": password,
	}

	resp, _ := s.do(http.MethodPost, "/register", "", payload)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	return username, password
}

func (s *APISuite) login(username, password string) (string, string) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	resp, body := s.do(http.MethodPost, "/token", "", payload)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	s.Require().NoError(json.Unmarshal(body, &res))
	return res.AccessToken, res.RefreshToken
}

func (s *APISuite) loginSomeUser() string {
	u, p := s.registerUser()
	access, _ := s.login(u, p)
	return access
}

func (s *APISuite) seedProduct(stock int, price string) uuid.UUID {
	decPrice, err := decimal.NewFromString(price)
	s.Require().NoError(err)

	p := &models.Product{
		ID:            uuid.New(),
		Name:          gofakeit.ProductName(),
		Price:         decPrice,
		StockQuantity: stock,
	}

	err = s.productRepo.Create(context.Background(), p)
	s.Require().NoError(err)
	return p.ID
}

func (s *APISuite) createOrder(token, idempotencyKey string, items ...map[string]any) (*http.Response, []byte) {
	payload := map[string]any{
		"items": items,
	}
	jsonBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	req, err := http.NewRequest(http.MethodPost, s.server.URL+"/orders", bytes.NewReader(jsonBytes))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := s.client.Do(req)
	s.Require().NoError(err)

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	return resp, respBody
}

func (s *APISuite) productStock(id uuid.UUID) int {
	var stock int
	err := s.pool.QueryRow(context.Background(), `SELECT stock_quantity FROM products WHERE id = $1`, id).Scan(&stock)
	s.Require().NoError(err)
	return stock
}

