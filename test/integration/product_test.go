package integration

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/shopspring/decimal"
)

func (s *APISuite) TestProductAPIs() {
	token := s.loginSomeUser()

	s.Run("product create as authenticated user returns 201", func() {
		resp, body := s.do(http.MethodPost, "/products", token, map[string]any{
			"name":           gofakeit.ProductName(),
			"price":          "19.99",
			"stock_quantity": 25,
		})
		s.Require().Equal(http.StatusCreated, resp.StatusCode)

		var res struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Price         string `json:"price"`
			StockQuantity int    `json:"stock_quantity"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.NotEmpty(res.ID)
		s.NotEmpty(res.Name)
		s.True(decimal.RequireFromString("19.99").Equal(decimal.RequireFromString(res.Price)))
		s.Equal(25, res.StockQuantity)
	})

	s.Run("product create as unauthenticated user returns 401", func() {
		resp, _ := s.do(http.MethodPost, "/products", "", map[string]any{
			"name":           gofakeit.ProductName(),
			"price":          "19.99",
			"stock_quantity": 25,
		})
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("product create with malformed body returns 400", func() {
		resp, _ := s.do(http.MethodPost, "/products", token, "nimadir")
		s.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("product create with empty name returns 422", func() {
		resp, _ := s.do(http.MethodPost, "/products", token, map[string]any{
			"name":           "",
			"price":          "19.99",
			"stock_quantity": 25,
		})
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("product create with zero price returns 422", func() {
		resp, _ := s.do(http.MethodPost, "/products", token, map[string]any{
			"name":           gofakeit.ProductName(),
			"price":          "0",
			"stock_quantity": 25,
		})
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("product create with negative price returns 422", func() {
		resp, _ := s.do(http.MethodPost, "/products", token, map[string]any{
			"name":           gofakeit.ProductName(),
			"price":          "-5.00",
			"stock_quantity": 25,
		})
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("product create with negative stock returns 422", func() {
		resp, _ := s.do(http.MethodPost, "/products", token, map[string]any{
			"name":           gofakeit.ProductName(),
			"price":          "19.99",
			"stock_quantity": -1,
		})
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})
}
