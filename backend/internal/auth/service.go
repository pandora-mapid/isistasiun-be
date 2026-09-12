package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type Service struct {
	repo       *Repository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(repo *Repository, jwtSecret string, accessTTLMinutes, refreshTTLHours int) *Service {
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLHours) * time.Hour,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPairResponse, error) {
	op, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(op.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(op)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPairResponse, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims["typ"] != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}

	sub, _ := claims["sub"].(string)
	op, err := s.repo.GetByID(ctx, sub)
	if err != nil {
		return nil, err
	}

	return s.issueTokenPair(op)
}

func (s *Service) issueTokenPair(op *Operator) (*TokenPairResponse, error) {
	access, err := s.signToken(op, "access", s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.signToken(op, "refresh", s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPairResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) signToken(op *Operator, typ string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  op.ID,
		"role": string(op.Role),
		"typ":  typ,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}
