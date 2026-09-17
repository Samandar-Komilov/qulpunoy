package routers

import "github.com/shopspring/decimal"

type userRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userRegisterResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	JoinedAt string `json:"joined_at"`
}

type tokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

type productCreateRequest struct {
	Name          string          `json:"name"`
	Price         decimal.Decimal `json:"price"`
	StockQuantity int             `json:"stock_quantity"`
}

type productResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Price         string `json:"price"`
	StockQuantity int    `json:"stock_quantity"`
	CreatedAt     string `json:"created_at"`
}

type orderResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Status      string `json:"status"`
	TotalAmount string `json:"total_amount"`
	CreatedAt   string `json:"created_at"`
}
type orderItemResponse struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	ProductName   string `json:"product_name"`
	ProductPrice  string `json:"product_price"`
	Quantity      int    `json:"quantity"`
	PriceSnapshot string `json:"price_snapshot"`
}
type orderDetailResponse struct {
	orderResponse
	Items []orderItemResponse `json:"items"`
}
