package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

// Код реализован с помощью ИИ, но отревьюен мной (человеком).
func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.port),
		Handler:           newHandler(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("Starting proxy on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
