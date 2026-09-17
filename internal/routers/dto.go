package routers

import "github.com/shopspring/decimal"

type userRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userRegisterResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	JoinedAt string `json:"joinedAt"`
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
	StockQuantity int             `json:"stockQuantity"`
}

type productResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Price         string `json:"price"`
	StockQuantity int    `json:"stockQuantity"`
	CreatedAt     string `json:"createdAt"`
}
