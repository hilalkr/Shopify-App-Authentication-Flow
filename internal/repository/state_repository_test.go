package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// VS Code sometimes runs tests from internal/repository:
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Fatal("set TEST_DATABASE_URL or DATABASE_URL")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStateConsume_SingleUseAndTTL(t *testing.T) {
	pool := mustPool(t)
	repo := NewStateRepository(pool)
	ctx := context.Background()

	shop := "unit-test.myshopify.com"
	nonce := "nonce-" + time.Now().Format("150405.000000000")

	_, _ = pool.Exec(ctx, "DELETE FROM oauth_states WHERE shop_domain=$1 OR nonce=$2", shop, nonce)

	if err := repo.Create(ctx, shop, nonce, 1*time.Minute); err != nil {
		t.Fatalf("create: %v", err)
	}

	// first consume: true
	ok, err := repo.Consume(ctx, shop, nonce)
	if err != nil {
		t.Fatalf("consume1 err: %v", err)
	}
	if !ok {
		t.Fatalf("expected consume1 ok=true")
	}

	// second consume: false (single-use)
	ok, err = repo.Consume(ctx, shop, nonce)
	if err != nil {
		t.Fatalf("consume2 err: %v", err)
	}
	if ok {
		t.Fatalf("expected consume2 ok=false")
	}

	// expired should not consume
	nonce2 := nonce + "-expired"
	_, _ = pool.Exec(ctx, "DELETE FROM oauth_states WHERE nonce=$1", nonce2)
	if err := repo.Create(ctx, shop, nonce2, -1*time.Minute); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	ok, err = repo.Consume(ctx, shop, nonce2)
	if err != nil {
		t.Fatalf("consume expired err: %v", err)
	}
	if ok {
		t.Fatalf("expected expired ok=false")
	}

	_, _ = pool.Exec(ctx, "DELETE FROM oauth_states WHERE shop_domain=$1", shop)
}
