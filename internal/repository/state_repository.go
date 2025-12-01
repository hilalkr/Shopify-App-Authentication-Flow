package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StateRepository struct {
	pool *pgxpool.Pool
}

func NewStateRepository(pool *pgxpool.Pool) *StateRepository {
	return &StateRepository{pool: pool}
}

// creat stores the generated OAuth state nonce and computes expires_at using the provided TTL
func (r *StateRepository) Create(ctx context.Context, shopDomain, nonce string, ttl time.Duration) error {
	expiresAt := time.Now().UTC().Add(ttl)

	const q = `
INSERT INTO oauth_states (shop_domain, nonce, expires_at)
VALUES ($1, $2, $3);
`
	_, err := r.pool.Exec(ctx, q, shopDomain, nonce, expiresAt)
	return err
}

func (r *StateRepository) Consume(ctx context.Context, shopDomain, nonce string) (bool, error) {
	const q = `
UPDATE oauth_states
SET used_at = NOW()
WHERE shop_domain = $1
  AND nonce = $2
  AND used_at IS NULL
  AND expires_at > NOW()
RETURNING id;
`
	var id int64
	err := r.pool.QueryRow(ctx, q, shopDomain, nonce).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
