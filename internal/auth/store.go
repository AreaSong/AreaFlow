package auth

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool        *pgxpool.Pool
	tokenMaxTTL time.Duration
}

func NewService(pool *pgxpool.Pool) Service {
	return Service{pool: pool, tokenMaxTTL: 90 * 24 * time.Hour}
}

func (s Service) WithTokenMaxTTL(value time.Duration) Service {
	s.tokenMaxTTL = value
	return s
}
