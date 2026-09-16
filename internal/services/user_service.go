package services

import (
	"context"
	"fmt"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) Register(ctx context.Context, username, password string) (*models.User, error) {
	if len(username) < 5 || len(password) < 8 {
		return nil, models.ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Unable to generate password hash: %w", err)
	}

	u := &models.User{
		ID:       uuid.New(),
		Username: username,
		Password: string(hash),
		IsActive: true,
	}

	err = s.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}
