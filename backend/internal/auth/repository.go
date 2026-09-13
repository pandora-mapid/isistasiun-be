package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("operator not found")
var ErrInvalidRefreshSession = errors.New("invalid refresh session")
var ErrEmailTaken = errors.New("email already registered")

const uniqueViolation = "23505"

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*Operator, error) {
	var o Operator
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, station_id, created_at
		FROM operators WHERE email = $1
	`, email).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.StationID, &o.CreatedAt)
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
		SELECT id, email, password_hash, role, station_id, created_at
		FROM operators WHERE id = $1
	`, id).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.StationID, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// Create inserts a new account of any role. stationID is nil for
// user/premium/admin — only an operator is ever scoped to one.
func (r *Repository) Create(ctx context.Context, email, passwordHash string, role Role, stationID *string) (*Operator, error) {
	var o Operator
	err := r.db.QueryRow(ctx, `
		INSERT INTO operators (email, password_hash, role, station_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, role, station_id, created_at
	`, email, passwordHash, string(role), stationID).
		Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.StationID, &o.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return &o, nil
}

// UpdateRole flips an account's role in place — used by the "upgrade to
// premium" demo flow, which has no real payment behind it.
func (r *Repository) UpdateRole(ctx context.Context, id string, role Role) (*Operator, error) {
	var o Operator
	err := r.db.QueryRow(ctx, `
		UPDATE operators SET role = $2
		WHERE id = $1
		RETURNING id, email, password_hash, role, station_id, created_at
	`, id, string(role)).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.Role, &o.StationID, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) CreateRefreshSession(ctx context.Context, session RefreshSession) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO operator_refresh_sessions
			(token_id, family_id, operator_id, expires_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
	`, session.TokenID, session.FamilyID, session.OperatorID, session.ExpiresAt)
	return err
}

// RotateRefreshSession consumes oldTokenID exactly once and creates next in
// the same transaction. A replay of an already-consumed token revokes the
// whole token family, including the newest token, so theft cannot remain
// invisible after the legitimate browser refreshes.
func (r *Repository) RotateRefreshSession(ctx context.Context, oldTokenID, operatorID string, next RefreshSession) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op after Commit

	var familyID, storedOperatorID string
	var expiresAt time.Time
	var revokedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT family_id::text, operator_id::text, expires_at, revoked_at
		FROM operator_refresh_sessions
		WHERE token_id = $1::uuid
		FOR UPDATE
	`, oldTokenID).Scan(&familyID, &storedOperatorID, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidRefreshSession
	}
	if err != nil {
		return err
	}

	if storedOperatorID != operatorID || revokedAt != nil || !expiresAt.After(time.Now()) {
		_, revokeErr := tx.Exec(ctx, `
			UPDATE operator_refresh_sessions
			SET revoked_at = COALESCE(revoked_at, now())
			WHERE family_id = $1::uuid
		`, familyID)
		if revokeErr != nil {
			return revokeErr
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return ErrInvalidRefreshSession
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO operator_refresh_sessions
			(token_id, family_id, operator_id, expires_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
	`, next.TokenID, familyID, next.OperatorID, next.ExpiresAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE operator_refresh_sessions
		SET revoked_at = now(), last_used_at = now(), replaced_by_token_id = $2::uuid
		WHERE token_id = $1::uuid
	`, oldTokenID, next.TokenID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) RevokeRefreshFamily(ctx context.Context, tokenID, operatorID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE operator_refresh_sessions
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE operator_id = $2::uuid
		  AND family_id = (
			SELECT family_id
			FROM operator_refresh_sessions
			WHERE token_id = $1::uuid AND operator_id = $2::uuid
		  )
	`, tokenID, operatorID)
	return err
}
