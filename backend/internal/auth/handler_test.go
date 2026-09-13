package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type fakeAuthService struct {
	tokens       *TokenPairResponse
	loginErr     error
	refreshErr   error
	logoutErr    error
	refreshInput string
	logoutInput  string
}

func (f *fakeAuthService) Login(context.Context, string, string) (*TokenPairResponse, error) {
	return f.tokens, f.loginErr
}

func (f *fakeAuthService) Register(context.Context, string, string) (*TokenPairResponse, error) {
	return f.tokens, f.loginErr
}

func (f *fakeAuthService) Upgrade(context.Context, string) (*TokenPairResponse, error) {
	return f.tokens, f.loginErr
}

func (f *fakeAuthService) CreateOperator(context.Context, string, string, string) (*Operator, error) {
	return nil, f.loginErr
}

func (f *fakeAuthService) Refresh(_ context.Context, token string) (*TokenPairResponse, error) {
	f.refreshInput = token
	return f.tokens, f.refreshErr
}

func (f *fakeAuthService) Logout(_ context.Context, token string) error {
	f.logoutInput = token
	return f.logoutErr
}

func testHandler(service authService) *Handler {
	return NewHandler(service, CookieConfig{TTL: time.Hour})
}

func testTokenResponse() *TokenPairResponse {
	return &TokenPairResponse{
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		ExpiresIn:    900,
		User: UserResponse{
			ID:    "11111111-1111-1111-1111-111111111111",
			Email: "operator@example.com",
			Role:  RoleOperator,
		},
	}
}

func performRequest(t *testing.T, app *fiber.App, request *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := app.Test(request, -1)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp, string(body)
}

func TestLoginSetsHttpOnlyRefreshCookieAndHidesItFromJSON(t *testing.T) {
	service := &fakeAuthService{tokens: testTokenResponse()}
	app := fiber.New()
	app.Post("/login", testHandler(service).Login)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"operator@example.com","password":"correct-password"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, body := performRequest(t, app, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "access-secret")
	require.NotContains(t, body, "refresh-secret")

	cookies := resp.Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, refreshCookieName, cookies[0].Name)
	require.Equal(t, "refresh-secret", cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	require.Equal(t, "/api/v1/auth", cookies[0].Path)
}

func TestRefreshReadsCookieAndRotatesIt(t *testing.T) {
	service := &fakeAuthService{tokens: testTokenResponse()}
	app := fiber.New()
	app.Post("/refresh", testHandler(service).Refresh)
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "old-refresh"})

	resp, _ := performRequest(t, app, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "old-refresh", service.refreshInput)
	require.Contains(t, resp.Header.Get(fiber.HeaderSetCookie), "refresh-secret")
}

func TestRefreshWithoutCookieIsUnauthorized(t *testing.T) {
	service := &fakeAuthService{tokens: testTokenResponse()}
	app := fiber.New()
	app.Post("/refresh", testHandler(service).Refresh)

	resp, _ := performRequest(t, app, httptest.NewRequest(http.MethodPost, "/refresh", nil))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogoutIsIdempotentAndClearsCookie(t *testing.T) {
	service := &fakeAuthService{}
	app := fiber.New()
	app.Post("/logout", testHandler(service).Logout)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-secret"})

	resp, _ := performRequest(t, app, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "refresh-secret", service.logoutInput)
	require.Contains(t, resp.Header.Get(fiber.HeaderSetCookie), refreshCookieName+"=")
	require.Contains(t, resp.Header.Get(fiber.HeaderSetCookie), "expires=Thu, 01 Jan 1970")
}
