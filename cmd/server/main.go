package main

import (
	"log"
	"shopify-auth-app/internal/config"
	"shopify-auth-app/internal/httpapi"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	r := httpapi.NewRouter()

	addr := ":" + cfg.AppPort

	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
