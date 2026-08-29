// Command createoperator inserts (or updates) an operator account so the
// premium tier can actually be signed into.
//
// There is deliberately no seed migration for this: a migration would commit a
// password hash to the repository, and every checkout would share the same
// credentials. Run this instead, once per environment:
//
//	go run ./cmd/createoperator -email ops@kai.id -role operator
//	docker compose exec backend go run ./cmd/createoperator -email ops@kai.id
//
// The password is read from OPERATOR_PASSWORD so it never lands in shell
// history or the process list.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/list-pandora/isi-stasiun-backend/internal/config"
)

const minPasswordLen = 8

func main() {
	email := flag.String("email", "", "operator email (required)")
	role := flag.String("role", "operator", "role: operator or admin")
	flag.Parse()

	if *email == "" {
		log.Fatal("-email is required")
	}
	if *role != "operator" && *role != "admin" {
		log.Fatalf("-role must be operator or admin, got %q", *role)
	}

	password := os.Getenv("OPERATOR_PASSWORD")
	if len(password) < minPasswordLen {
		log.Fatalf("OPERATOR_PASSWORD must be set and at least %d characters", minPasswordLen)
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Upsert so re-running it rotates the password instead of failing on the
	// unique email constraint — the usual reason to run this twice.
	var id string
	err = pool.QueryRow(ctx, `
		INSERT INTO operators (email, password_hash, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		  SET password_hash = EXCLUDED.password_hash,
		      role          = EXCLUDED.role
		RETURNING id
	`, *email, string(hash), *role).Scan(&id)
	if err != nil {
		log.Fatalf("upsert operator: %v", err)
	}

	fmt.Printf("operator ready: %s (%s) id=%s\n", *email, *role, id)
}
