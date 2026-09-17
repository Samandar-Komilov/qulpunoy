package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Samandar-Komilov/qulpunoy/internal/auth"
	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (access, refresh string, err error)
	Refresh(ctx context.Context, refresh_token string) (access_token string, err error)
}

type authService struct {
	repo *repositories.UserRepository
	jwt  auth.JWTManager
}

func NewAuthService(repo *repositories.UserRepository, jwt auth.JWTManager) AuthService {
	return &authService{
		repo: repo,
		jwt:  jwt,
	}
}

func (s *authService) Login(ctx context.Context, username, password string) (access, refresh string, err error) {
	username = strings.TrimSpace(username)
	if len(username) < 5 || len(password) < 8 {
		return "", "", models.ErrInvalidInput
	}

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", models.ErrUnauthorized
	}
	if !user.IsActive {
		return "", "", models.ErrForbidden
	}

	access, err = s.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return "", "", err
	}
	refresh, err = s.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *authService) Refresh(ctx context.Context, refresh_token string) (access_token string, err error) {
	claims, err := s.jwt.Parse(refresh_token)
	if err != nil {
		return "", fmt.Errorf("Failed to parse refresh token: %w", models.ErrUnauthorized)
	}
	if claims.Type != "refresh" {
		return "", fmt.Errorf("Invalid token type: %w", models.ErrUnauthorized)
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return "", fmt.Errorf("Invalid Claims Subject (must be UUID): %w", err)
	}

	access_token, err = s.jwt.GenerateAccessToken(userID)
	if err != nil {
		return "", fmt.Errorf("Failed to generate access token: %w", err)
	}
	return access_token, nil
}
