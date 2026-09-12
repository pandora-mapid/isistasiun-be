package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/database"
	"github.com/list-pandora/isi-stasiun-backend/internal/rental"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 20 * time.Second}
	if err := rental.SyncSpaceKAI(ctx, db, client); err != nil {
		log.Fatalf("rental sync failed: %v", err)
	}
	log.Printf("Space KAI Manggarai rental snapshot synced")
}
