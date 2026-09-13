package auth

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest creates a public, free-tier account (RoleUser). No station,
// no admin approval — that's what separates it from CreateOperatorRequest.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// CreateOperatorRequest is admin-only: an operator represents a real
// organization and is scoped to one station from the moment it exists.
type CreateOperatorRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	StationID string `json:"station_id" validate:"required,uuid"`
}

type UserResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      Role    `json:"role"`
	StationID *string `json:"station_id,omitempty"`
}

type TokenPairResponse struct {
	AccessToken  string       `json:"access_token"`
	ExpiresIn    int          `json:"expires_in"` // seconds
	User         UserResponse `json:"user"`
	RefreshToken string       `json:"-"`
}
