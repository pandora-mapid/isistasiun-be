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
	CreatedAt    time.Time `json:"created_at"`
}
