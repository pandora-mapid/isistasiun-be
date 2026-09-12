package auth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret-with-at-least-32-characters"

type fakeRepository struct {
	operator       *Operator
	created        RefreshSession
	rotatedFrom    string
	rotatedTo      RefreshSession
	revokedTokenID string
	rotateErr      error
}

func (f *fakeRepository) GetByEmail(_ context.Context, email string) (*Operator, error) {
	if f.operator == nil || email != f.operator.Email {
		return nil, ErrNotFound
	}
	return f.operator, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (*Operator, error) {
	if f.operator == nil || id != f.operator.ID {
		return nil, ErrNotFound
	}
	return f.operator, nil
}

func (f *fakeRepository) CreateRefreshSession(_ context.Context, session RefreshSession) error {
	f.created = session
	return nil
}

func (f *fakeRepository) RotateRefreshSession(_ context.Context, oldTokenID, _ string, next RefreshSession) error {
	f.rotatedFrom = oldTokenID
	f.rotatedTo = next
	return f.rotateErr
}

func (f *fakeRepository) RevokeRefreshFamily(_ context.Context, tokenID, _ string) error {
	f.revokedTokenID = tokenID
	return nil
}

func testOperator(t *testing.T) *Operator {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)
	return &Operator{
		ID:           "11111111-1111-1111-1111-111111111111",
		Email:        "operator@example.com",
		PasswordHash: string(hash),
		Role:         RoleOperator,
	}
}

func parseTestClaims(t *testing.T, raw string) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	require.NoError(t, err)
	require.True(t, token.Valid)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

func TestLoginIssuesAccessAndServerTrackedRefreshSession(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t)}
	service := NewService(repo, testSecret, 15, 168)

	tokens, err := service.Login(context.Background(), "  OPERATOR@example.com ", "correct-password")
	require.NoError(t, err)
	require.Equal(t, repo.operator.Email, tokens.User.Email)
	require.Equal(t, RoleOperator, tokens.User.Role)
	require.Equal(t, "access", parseTestClaims(t, tokens.AccessToken)["typ"])

	refreshClaims := parseTestClaims(t, tokens.RefreshToken)
	require.Equal(t, "refresh", refreshClaims["typ"])
	tokenID, ok := refreshClaims["jti"].(string)
	require.True(t, ok)
	require.Equal(t, tokenID, repo.created.TokenID)
	require.Equal(t, tokenID, repo.created.FamilyID)
	require.Equal(t, repo.operator.ID, repo.created.OperatorID)
	require.WithinDuration(t, time.Now().Add(168*time.Hour), repo.created.ExpiresAt, time.Second)

	encoded, err := json.Marshal(tokens)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "refresh_token")
	require.NotContains(t, string(encoded), tokens.RefreshToken)
}

func TestLoginRejectsWrongPasswordWithoutCreatingSession(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t)}
	service := NewService(repo, testSecret, 15, 168)

	_, err := service.Login(context.Background(), repo.operator.Email, "wrong-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.Empty(t, repo.created.TokenID)
}

func TestRefreshRotatesTrackedToken(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t)}
	service := NewService(repo, testSecret, 15, 168)
	first, err := service.Login(context.Background(), repo.operator.Email, "correct-password")
	require.NoError(t, err)
	firstTokenID := repo.created.TokenID

	second, err := service.Refresh(context.Background(), first.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, firstTokenID, repo.rotatedFrom)
	require.NotEqual(t, firstTokenID, repo.rotatedTo.TokenID)
	require.Equal(t, repo.operator.ID, repo.rotatedTo.OperatorID)
	require.Equal(t, repo.rotatedTo.TokenID, parseTestClaims(t, second.RefreshToken)["jti"])
}

func TestRefreshRejectsAccessTokenAndConsumedSession(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t)}
	service := NewService(repo, testSecret, 15, 168)
	tokens, err := service.Login(context.Background(), repo.operator.Email, "correct-password")
	require.NoError(t, err)

	_, err = service.Refresh(context.Background(), tokens.AccessToken)
	require.ErrorIs(t, err, ErrInvalidRefreshToken)

	repo.rotateErr = ErrInvalidRefreshSession
	_, err = service.Refresh(context.Background(), tokens.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestLogoutRevokesRefreshFamilyAndIsIdempotentForBadCookie(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t)}
	service := NewService(repo, testSecret, 15, 168)
	tokens, err := service.Login(context.Background(), repo.operator.Email, "correct-password")
	require.NoError(t, err)

	require.NoError(t, service.Logout(context.Background(), tokens.RefreshToken))
	require.Equal(t, repo.created.TokenID, repo.revokedTokenID)
	require.NoError(t, service.Logout(context.Background(), "not-a-token"))
}

func TestRefreshPropagatesUnexpectedRepositoryFailure(t *testing.T) {
	repo := &fakeRepository{operator: testOperator(t), rotateErr: errors.New("database unavailable")}
	service := NewService(repo, testSecret, 15, 168)
	tokens, err := service.Login(context.Background(), repo.operator.Email, "correct-password")
	require.NoError(t, err)

	_, err = service.Refresh(context.Background(), tokens.RefreshToken)
	require.EqualError(t, err, "database unavailable")
}
