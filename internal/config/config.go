package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort          string
	DatabaseURL      string
	ShopifyAPIKey    string
	ShopifyAPISecret string
	ShopifyScopes    string
	CallbackURL      string
}

func Load() Config {
	return Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		ShopifyAPIKey:    mustEnv("SHOPIFY_API_KEY"),
		ShopifyAPISecret: mustEnv("SHOPIFY_API_SECRET"),
		ShopifyScopes:    getEnv("SHOPIFY_SCOPES", "read_products"),
		CallbackURL:      mustEnv("OAUTH_CALLBACK_URL"),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing env: %s", key)
	}
	return v
}
