package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("operator not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*Operator, error) {
	var o Operator
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM operators WHERE email = $1
	`, email).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Operator, error) {
	var o Operator
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM operators WHERE id = $1
	`, id).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}
