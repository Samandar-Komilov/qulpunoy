package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
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
