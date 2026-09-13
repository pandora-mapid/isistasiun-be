package auth

import "time"

type Role string

const (
	RoleOperator Role = "operator" // KAI / KAI Commuter / kawasan operator — premium tier
	RoleAdmin    Role = "admin"
)

type Operator struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	// StationID scopes an operator to the single station it represents
	// (section 4.1). Always nil for admin, which sees every station.
	StationID *string   `json:"station_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshSession is the server-side half of a refresh JWT. The browser keeps
// the signed token in an HttpOnly cookie; the database keeps only its random
// token id so a stolen or replayed token can be revoked without storing the
// credential itself.
type RefreshSession struct {
	TokenID    string
	FamilyID   string
	OperatorID string
	ExpiresAt  time.Time
}
