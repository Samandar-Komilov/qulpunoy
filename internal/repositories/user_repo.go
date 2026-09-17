package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, username, password, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, is_active, joined_at
	`, u.ID, u.Username, u.Password, u.IsActive).Scan(&u.ID, &u.Username, &u.IsActive, &u.JoinedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.ErrUsernameTaken
		}
		return fmt.Errorf("User create error: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, password, is_active, joined_at
		FROM users
		WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.Password, &u.IsActive, &u.JoinedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("Failed to get user by username: %w", err)
	}

	return u, nil
}
