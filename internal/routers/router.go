package routers

import (
	"net/http"

	"github.com/Samandar-Komilov/qulpunoy/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	JWT            auth.JWTManager
	UserHandler    *UserHandler
	AuthHandler    *AuthHandler
	ProductHandler *ProductHandler
	OrderHandler   *OrderHandler
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/register", d.UserHandler.Register)
	r.Post("/token", d.AuthHandler.Token)
	r.Post("/refresh", d.AuthHandler.Refresh)
	r.Group(func(pr chi.Router) {
		pr.Use(auth.Authenticator(d.JWT))
		pr.Post("/products", d.ProductHandler.Create)
		pr.Get("/orders", d.OrderHandler.List)
		pr.Get("/orders/{id}", d.OrderHandler.Get)
		pr.Post("/orders", d.OrderHandler.Create)
		pr.Post("/orders/{id}/cancel", d.OrderHandler.Cancel)
	})
	return r
}
