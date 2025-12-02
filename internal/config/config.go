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
	SessionSecret    string
}

func Load() Config {
	shopifySecret := mustEnv("SHOPIFY_API_SECRET")

	sessionSecret := os.Getenv("APP_SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = shopifySecret
	}

	return Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		ShopifyAPIKey:    mustEnv("SHOPIFY_API_KEY"),
		ShopifyAPISecret: shopifySecret,
		ShopifyScopes:    getEnv("SHOPIFY_SCOPES", "read_products"),
		CallbackURL:      mustEnv("OAUTH_CALLBACK_URL"),
		SessionSecret:    sessionSecret,
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
