package integration

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *APISuite) TestOrderCreate() {
	token := s.loginSomeUser()

	s.Run("sufficient stock returns 201", func() {
		productID := s.seedProduct(10, "15.50")
		resp, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   2,
		})

		s.Require().Equal(http.StatusCreated, resp.StatusCode)

		var res struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			TotalAmount string `json:"total_amount"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.NotEmpty(res.ID)
		s.Equal("pending", res.Status)
		s.True(decimal.RequireFromString("31.00").Equal(decimal.RequireFromString(res.TotalAmount)))
	})

	s.Run("insufficient stock returns 409", func() {
		productID := s.seedProduct(10, "15.50")
		resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   999,
		})
		s.Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("decrements stock", func() {
		productID := s.seedProduct(10, "5.00")
		resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   4,
		})
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Equal(6, s.productStock(productID))
	})

	s.Run("replay with same key returns 200, same order and decrements once", func() {
		productID := s.seedProduct(10, "5.00")
		key := gofakeit.UUID()

		resp1, body1 := s.createOrder(token, key, map[string]any{
			"product_id": productID.String(),
			"quantity":   3,
		})
		s.Require().Equal(http.StatusCreated, resp1.StatusCode)
		firstID := s.orderIDFromBody(body1)

		resp2, body2 := s.createOrder(token, key, map[string]any{
			"product_id": productID.String(),
			"quantity":   3,
		})
		s.Equal(http.StatusOK, resp2.StatusCode)
		s.Equal(firstID, s.orderIDFromBody(body2))
		s.Equal(7, s.productStock(productID))
	})

	s.Run("missing idempotency key returns 400", func() {
		productID := s.seedProduct(10, "9.99")
		resp, _ := s.createOrder(token, "", map[string]any{
			"product_id": productID.String(),
			"quantity":   1,
		})
		s.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("empty items returns 422", func() {
		resp, _ := s.createOrder(token, gofakeit.UUID())
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("non positive quantity returns 422", func() {
		productID := s.seedProduct(10, "9.99")
		resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   0,
		})
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("duplicate product returns 422", func() {
		productID := s.seedProduct(10, "9.99")
		resp, _ := s.createOrder(token, gofakeit.UUID(),
			map[string]any{"product_id": productID.String(), "quantity": 1},
			map[string]any{"product_id": productID.String(), "quantity": 2},
		)
		s.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("invalid product id returns 400", func() {
		resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": "not-a-uuid",
			"quantity":   1,
		})
		s.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("unknown product returns 404", func() {
		resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": uuid.New().String(),
			"quantity":   1,
		})
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("without token returns 401", func() {
		productID := s.seedProduct(10, "9.99")
		resp, _ := s.createOrder("", gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   1,
		})
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *APISuite) TestOrderGet() {
	token := s.loginSomeUser()
	productID := s.seedProduct(10, "12.00")
	_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
		"product_id": productID.String(),
		"quantity":   2,
	})
	orderID := s.orderIDFromBody(body)

	s.Run("get own order returns 200 with items", func() {
		resp, body := s.do(http.MethodGet, "/orders/"+orderID.String(), token, nil)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res struct {
			ID    string `json:"id"`
			Items []struct {
				ProductID string `json:"product_id"`
				Quantity  int    `json:"quantity"`
			} `json:"items"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Equal(orderID.String(), res.ID)
		s.Require().Len(res.Items, 1)
		s.Equal(productID.String(), res.Items[0].ProductID)
		s.Equal(2, res.Items[0].Quantity)
	})

	s.Run("get another user's order returns 404", func() {
		other := s.loginSomeUser()
		resp, _ := s.do(http.MethodGet, "/orders/"+orderID.String(), other, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("get unknown order returns 404", func() {
		resp, _ := s.do(http.MethodGet, "/orders/"+uuid.New().String(), token, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("get with invalid id returns 400", func() {
		resp, _ := s.do(http.MethodGet, "/orders/not-a-uuid", token, nil)
		s.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("get without token returns 401", func() {
		resp, _ := s.do(http.MethodGet, "/orders/"+orderID.String(), "", nil)
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *APISuite) TestOrderList() {
	token := s.loginSomeUser()
	productID := s.seedProduct(10, "5.00")
	s.createOrder(token, gofakeit.UUID(), map[string]any{"product_id": productID.String(), "quantity": 1})
	s.createOrder(token, gofakeit.UUID(), map[string]any{"product_id": productID.String(), "quantity": 1})

	s.Run("list returns only own orders", func() {
		resp, body := s.do(http.MethodGet, "/orders", token, nil)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res []map[string]any
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Len(res, 2)
	})

	s.Run("list for other user is empty", func() {
		other := s.loginSomeUser()
		resp, body := s.do(http.MethodGet, "/orders", other, nil)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res []map[string]any
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Len(res, 0)
	})
}

func (s *APISuite) TestOrderConfirm() {
	token := s.loginSomeUser()
	productID := s.seedProduct(10, "8.00")
	_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
		"product_id": productID.String(),
		"quantity":   3,
	})
	orderID := s.orderIDFromBody(body)
	s.Require().Equal(7, s.productStock(productID))

	s.Run("confirm pending order returns 200 and keeps stock", func() {
		resp, body := s.do(http.MethodPost, "/orders/"+orderID.String()+"/confirm", token, nil)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res struct {
			Status string `json:"status"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Equal("confirmed", res.Status)
		s.Equal(7, s.productStock(productID))
	})

	s.Run("confirm already confirmed order returns 409", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+orderID.String()+"/confirm", token, nil)
		s.Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("cancel confirmed order returns 409", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+orderID.String()+"/cancel", token, nil)
		s.Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("confirmed order is not auto cancelled by job", func() {
		s.expireOrderIntentionally(orderID, "16 minutes")
		expired, err := s.orderRepo.ExpirePendingOrders(context.Background())
		s.Require().NoError(err)
		s.Len(expired, 0)
		s.Equal("confirmed", s.orderStatus(orderID))
	})

	s.Run("confirm unknown order returns 404", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+uuid.New().String()+"/confirm", token, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("confirm another user's order returns 404", func() {
		token2 := s.loginSomeUser()
		_, body := s.createOrder(token2, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   1,
		})
		otherOrderID := s.orderIDFromBody(body)

		resp, _ := s.do(http.MethodPost, "/orders/"+otherOrderID.String()+"/confirm", token, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("confirm without token returns 401", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+orderID.String()+"/confirm", "", nil)
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *APISuite) TestOrderCancel() {
	token := s.loginSomeUser()
	productID := s.seedProduct(10, "8.00")
	_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
		"product_id": productID.String(),
		"quantity":   3,
	})
	orderID := s.orderIDFromBody(body)
	s.Require().Equal(7, s.productStock(productID))

	s.Run("cancel pending order returns 200 and restores stock", func() {
		resp, body := s.do(http.MethodPost, "/orders/"+orderID.String()+"/cancel", token, nil)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res struct {
			Status string `json:"status"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Equal("cancelled", res.Status)
		s.Equal(10, s.productStock(productID))
	})

	s.Run("cancel already cancelled order returns 409", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+orderID.String()+"/cancel", token, nil)
		s.Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("cancel does not restore stock twice", func() {
		s.Equal(10, s.productStock(productID))
	})

	s.Run("cancel unknown order returns 404", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+uuid.New().String()+"/cancel", token, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("cancel another user's order returns 404", func() {
		token2 := s.loginSomeUser()
		_, body := s.createOrder(token2, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   1,
		})
		otherOrderID := s.orderIDFromBody(body)

		resp, _ := s.do(http.MethodPost, "/orders/"+otherOrderID.String()+"/cancel", token, nil)
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("cancel without token returns 401", func() {
		resp, _ := s.do(http.MethodPost, "/orders/"+orderID.String()+"/cancel", "", nil)
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *APISuite) TestExpirePendingOrders() {
	token := s.loginSomeUser()
	productID := s.seedProduct(10, "6.00")

	s.Run("expired pending order is cancelled and stock restored", func() {
		_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   4,
		})
		orderID := s.orderIDFromBody(body)
		s.Require().Equal(6, s.productStock(productID))
		s.expireOrderIntentionally(orderID, "16 minutes")

		expired, err := s.orderRepo.ExpirePendingOrders(context.Background())
		s.Require().NoError(err)
		s.Len(expired, 1)
		s.Equal("cancelled", s.orderStatus(orderID))
		s.Equal(10, s.productStock(productID))
	})

	s.Run("fresh pending order is not expired", func() {
		_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   2,
		})
		orderID := s.orderIDFromBody(body)

		expired, err := s.orderRepo.ExpirePendingOrders(context.Background())
		s.Require().NoError(err)
		s.Len(expired, 0)
		s.Equal("pending", s.orderStatus(orderID))
		s.Equal(8, s.productStock(productID))
	})

	s.Run("already cancelled order is not restored again", func() {
		_, body := s.createOrder(token, gofakeit.UUID(), map[string]any{
			"product_id": productID.String(),
			"quantity":   3,
		})
		orderID := s.orderIDFromBody(body)
		s.do(http.MethodPost, "/orders/"+orderID.String()+"/cancel", token, nil)
		stockAfterCancel := s.productStock(productID)
		s.expireOrderIntentionally(orderID, "16 minutes")

		expired, err := s.orderRepo.ExpirePendingOrders(context.Background())
		s.Require().NoError(err)
		s.Len(expired, 0)
		s.Equal(stockAfterCancel, s.productStock(productID))
	})
}

/* Utility test functions */

func (s *APISuite) orderIDFromBody(body []byte) uuid.UUID {
	var res struct {
		ID string `json:"id"`
	}
	s.Require().NoError(json.Unmarshal(body, &res))
	id, err := uuid.Parse(res.ID)
	s.Require().NoError(err)
	return id
}

func (s *APISuite) orderStatus(id uuid.UUID) string {
	var status string
	err := s.pool.QueryRow(context.Background(), `SELECT status FROM orders WHERE id = $1`, id).Scan(&status)
	s.Require().NoError(err)
	return status
}

func (s *APISuite) expireOrderIntentionally(id uuid.UUID, interval string) {
	_, err := s.pool.Exec(context.Background(),
		`UPDATE orders SET created_at = NOW() - $1::interval WHERE id = $2`, interval, id)
	s.Require().NoError(err)
}
