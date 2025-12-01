package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Shop struct {
	ID                 int64
	ShopDomain         string
	OfflineAccessToken string
	Scopes             string
	InstalledAt        time.Time
	UpdatedAt          time.Time
}

type ShopRepository struct {
	pool *pgxpool.Pool
}

func NewShopRepository(pool *pgxpool.Pool) *ShopRepository {
	return &ShopRepository{pool: pool}
}

// GetByDomain retrieves a shop by its domain from the database
func (r *ShopRepository) GetByDomain(ctx context.Context, shopDomain string) (*Shop, error) {
	const q = `
SELECT id, shop_domain, offline_access_token, scopes, installed_at, updated_at
FROM shops
WHERE shop_domain = $1
LIMIT 1;
`
	var s Shop
	err := r.pool.QueryRow(ctx, q, shopDomain).Scan(
		&s.ID,
		&s.ShopDomain,
		&s.OfflineAccessToken,
		&s.Scopes,
		&s.InstalledAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// upsert inserts a new shop or updates existing shop's token and scopes
func (r *ShopRepository) Upsert(ctx context.Context, shopDomain, token, scopes string) (*Shop, error) {
	const q = `
INSERT INTO shops (shop_domain, offline_access_token, scopes)
VALUES ($1, $2, $3)
ON CONFLICT (shop_domain) DO UPDATE
SET offline_access_token = EXCLUDED.offline_access_token,
    scopes = EXCLUDED.scopes,
    updated_at = NOW()
RETURNING id, shop_domain, offline_access_token, scopes, installed_at, updated_at;
`
	var s Shop
	if err := r.pool.QueryRow(ctx, q, shopDomain, token, scopes).Scan(
		&s.ID, &s.ShopDomain, &s.OfflineAccessToken, &s.Scopes, &s.InstalledAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}
