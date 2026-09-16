package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Username  string
	Password  string
	IsActive  bool
	JoinedAt  time.Time
	UpdatedAt time.Time
}
