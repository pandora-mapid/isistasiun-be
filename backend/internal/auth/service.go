package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
var ErrNotEligibleForUpgrade = errors.New("only a user account can upgrade to premium")

type authRepository interface {
	GetByEmail(ctx context.Context, email string) (*Operator, error)
	GetByID(ctx context.Context, id string) (*Operator, error)
	Create(ctx context.Context, email, passwordHash string, role Role, stationID *string) (*Operator, error)
	UpdateRole(ctx context.Context, id string, role Role) (*Operator, error)
	CreateRefreshSession(ctx context.Context, session RefreshSession) error
	RotateRefreshSession(ctx context.Context, oldTokenID, operatorID string, next RefreshSession) error
	RevokeRefreshFamily(ctx context.Context, tokenID, operatorID string) error
}

type Service struct {
	repo       authRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(repo authRepository, jwtSecret string, accessTTLMinutes, refreshTTLHours int) *Service {
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLHours) * time.Hour,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPairResponse, error) {
	op, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(op.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueAndPersist(ctx, op)
}

// Register creates a public, free-tier account and logs it straight in — no
// email verification, no admin approval. Only RoleUser is reachable this way;
// operator accounts are admin-provisioned (CreateOperator) and premium is
// reached by Upgrade, never by registering directly.
func (s *Service) Register(ctx context.Context, email, password string) (*TokenPairResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	op, err := s.repo.Create(ctx, strings.ToLower(strings.TrimSpace(email)), string(hash), RoleUser, nil)
	if err != nil {
		return nil, err
	}
	return s.issueAndPersist(ctx, op)
}

// Upgrade is the entire "payment" flow: no gateway, no charge, a RoleUser
// account becomes RolePremium the moment it's called. Reissues a fresh token
// pair so the new role claim takes effect immediately, without waiting for
// the old access token to expire.
func (s *Service) Upgrade(ctx context.Context, operatorID string) (*TokenPairResponse, error) {
	op, err := s.repo.GetByID(ctx, operatorID)
	if err != nil {
		return nil, err
	}
	if op.Role != RoleUser {
		return nil, ErrNotEligibleForUpgrade
	}
	upgraded, err := s.repo.UpdateRole(ctx, operatorID, RolePremium)
	if err != nil {
		return nil, err
	}
	return s.issueAndPersist(ctx, upgraded)
}

// CreateOperator is admin-only (enforced by the route guard, not here): an
// operator represents a real organization and is scoped to one station from
// the moment it exists, unlike a self-registered user.
func (s *Service) CreateOperator(ctx context.Context, email, password, stationID string) (*Operator, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, strings.ToLower(strings.TrimSpace(email)), string(hash), RoleOperator, &stationID)
}

func (s *Service) issueAndPersist(ctx context.Context, op *Operator) (*TokenPairResponse, error) {
	tokens, session, err := s.issueTokenPair(op)
	if err != nil {
		return nil, err
	}
	session.FamilyID = session.TokenID
	if err := s.repo.CreateRefreshSession(ctx, session); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPairResponse, error) {
	operatorID, tokenID, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	op, err := s.repo.GetByID(ctx, operatorID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}

	tokens, next, err := s.issueTokenPair(op)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RotateRefreshSession(ctx, tokenID, operatorID, next); err != nil {
		if errors.Is(err, ErrInvalidRefreshSession) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}
	return tokens, nil
}

// Logout is deliberately idempotent. A missing, malformed, or already-expired
// cookie still results in a cleared browser cookie; a valid token revokes its
// complete rotation family in the database.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	operatorID, tokenID, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return nil
	}
	return s.repo.RevokeRefreshFamily(ctx, tokenID, operatorID)
}

func (s *Service) issueTokenPair(op *Operator) (*TokenPairResponse, RefreshSession, error) {
	access, err := s.signToken(op, "access", s.accessTTL, "")
	if err != nil {
		return nil, RefreshSession{}, err
	}
	tokenID := uuid.NewString()
	refresh, err := s.signToken(op, "refresh", s.refreshTTL, tokenID)
	if err != nil {
		return nil, RefreshSession{}, err
	}

	return &TokenPairResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    int(s.accessTTL.Seconds()),
			User: UserResponse{
				ID:        op.ID,
				Email:     op.Email,
				Role:      op.Role,
				StationID: op.StationID,
			},
		}, RefreshSession{
			TokenID:    tokenID,
			OperatorID: op.ID,
			ExpiresAt:  time.Now().Add(s.refreshTTL),
		}, nil
}

func (s *Service) signToken(op *Operator, typ string, ttl time.Duration, tokenID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  op.ID,
		"role": string(op.Role),
		"typ":  typ,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(ttl).Unix(),
	}
	if tokenID != "" {
		claims["jti"] = tokenID
	}
	if op.StationID != nil {
		claims["station_id"] = *op.StationID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

func (s *Service) parseRefreshToken(tokenStr string) (operatorID, tokenID string, err error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil || claims["typ"] != "refresh" {
		return "", "", ErrInvalidRefreshToken
	}
	operatorID, _ = claims["sub"].(string)
	tokenID, _ = claims["jti"].(string)
	if _, parseErr := uuid.Parse(operatorID); parseErr != nil {
		return "", "", ErrInvalidRefreshToken
	}
	if _, parseErr := uuid.Parse(tokenID); parseErr != nil {
		return "", "", ErrInvalidRefreshToken
	}
	return operatorID, tokenID, nil
}
