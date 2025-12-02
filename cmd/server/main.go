package main

import (
	"log"
	"log/slog"
	"os"
	"shopify-auth-app/internal/config"
	"shopify-auth-app/internal/db"
	"shopify-auth-app/internal/httpapi"
	"shopify-auth-app/internal/repository"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// db connection
	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	shopRepo := repository.NewShopRepository(pool)
	stateRepo := repository.NewStateRepository(pool)

	handlers := httpapi.NewHandlers(cfg, shopRepo, stateRepo, logger)
	r := httpapi.NewRouter(handlers)

	addr := ":" + cfg.AppPort

	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
