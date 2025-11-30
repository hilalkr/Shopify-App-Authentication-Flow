package main

import (
	"log"
	"shopify-auth-app/internal/httpapi"
)

func main() {
	r := httpapi.NewRouter()

	addr := ":8080"

	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
