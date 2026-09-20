package integration

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
)

func (s *APISuite) TestAuthRegister() {
	s.Run("register new user succeeds with 201", func() {
		username := gofakeit.Username() + "123"
		password := "SecretPass123!"

		resp, body := s.do(http.MethodPost, "/register", "", map[string]string{
			"username": username,
			"password": password,
		})

		s.Require().Equal(http.StatusCreated, resp.StatusCode)

		var res struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			JoinedAt string `json:"joined_at"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.Equal(username, res.Username)
		s.NotEmpty(res.ID)
		s.NotEmpty(res.JoinedAt)
	})

	s.Run("register duplicate username returns 409", func() {
		username, password := s.registerUser()

		resp, _ := s.do(http.MethodPost, "/register", "", map[string]string{
			"username": username,
			"password": password,
		})

		s.Equal(http.StatusConflict, resp.StatusCode)
	})
}

func (s *APISuite) TestAuthTokens() {
	username, password := s.registerUser()

	s.Run("token valid credentials return access and refresh tokens", func() {
		resp, body := s.do(http.MethodPost, "/token", "", map[string]string{
			"username": username,
			"password": password,
		})

		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.NotEmpty(res.AccessToken)
		s.NotEmpty(res.RefreshToken)
	})

	s.Run("token invalid password returns 401", func() {
		resp, _ := s.do(http.MethodPost, "/token", "", map[string]string{
			"username": username,
			"password": "wrong-password",
		})

		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("refresh valid refresh token returns new access token", func() {
		_, refreshToken := s.login(username, password)

		resp, body := s.do(http.MethodPost, "/refresh", "", map[string]string{
			"refresh_token": refreshToken,
		})

		s.Require().Equal(http.StatusOK, resp.StatusCode)

		var res struct {
			AccessToken string `json:"access_token"`
		}
		s.Require().NoError(json.Unmarshal(body, &res))
		s.NotEmpty(res.AccessToken)
	})

	s.Run("protected route without token returns 401", func() {
		resp, _ := s.do(http.MethodGet, "/orders", "", nil)
		s.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}
