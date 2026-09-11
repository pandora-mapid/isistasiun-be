package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v5"

	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/router"
)

// These tests pin the access tier of every route, and exist because that tier
// was silently wrong: mounting middleware with Group("", ...) compiles to
// Use("/api/v1", ...), which applies to every route registered afterwards. The
// service-key guard therefore swallowed transparency, auth and premium — all
// seven endpoints answered 401 while looking correctly wired in the source.
//
// A route's tier is a property of the whole router, not of one handler, so it
// can only be tested here. Handlers run against a nil pool; anything that
// reaches the database panics and the recover middleware turns that into a
// 500. That is fine and in fact the point — a 500 proves the request got past
// the guards, which is exactly what these tests assert.

const (
	testJWTSecret = "test-secret"
	testSvcKey    = "test-service-key"
)

func newTestApp() *fiber.App {
	app := fiber.New()
	app.Use(recover.New())

	router.Setup(app, nil, &config.Config{
		JWTSecret:             testJWTSecret,
		JWTAccessTTLMins:      15,
		JWTRefreshTTLHours:    168,
		PipelineServiceAPIKey: testSvcKey,
		AIServiceURL:          "http://127.0.0.1:1", // guaranteed refused: exercises the fallback path
		RateLimitRPM:          1000,                 // high enough not to interfere
	})
	return app
}

func do(t *testing.T, app *fiber.App, method, path string, headers map[string]string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func token(t *testing.T, role, typ string, ttl time.Duration) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "11111111-1111-1111-1111-111111111111",
		"role": role,
		"typ":  typ,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(ttl).Unix(),
	})
	signed, err := tok.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// TestPublicRoutesNeedNoCredentials covers the regression directly: these must
// never answer 401 or 403, whoever asks. Transparency is public by design —
// section 9 makes traceability the product's central claim, so putting the
// evidence panel behind a key would defeat it.
func TestPublicRoutesNeedNoCredentials(t *testing.T) {
	app := newTestApp()

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/stations"},
		{http.MethodGet, "/api/v1/stations/abc"},
		{http.MethodGet, "/api/v1/stations/abc/entrances"},
		{http.MethodGet, "/api/v1/analytics/spending-gap"},
		{http.MethodGet, "/api/v1/analytics/spending-gap/abc"},
		{http.MethodGet, "/api/v1/analytics/category-gap"},
		{http.MethodGet, "/api/v1/analytics/rent-flow-index"},
		{http.MethodGet, "/api/v1/analytics/event-potential"},
		{http.MethodGet, "/api/v1/analytics/station-summary"},
		{http.MethodGet, "/api/v1/analytics/station-summary/abc"},
		{http.MethodGet, "/api/v1/confidence-layer"},
		{http.MethodGet, "/api/v1/transparency/struk/abc"},
		{http.MethodGet, "/api/v1/transparency/gerai/abc"},
		{http.MethodGet, "/api/v1/transparency/properti/abc"},
		{http.MethodGet, "/api/v1/transparency/station/abc/records"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/copilot/query"},
	} {
		got := do(t, app, tc.method, tc.path, nil)
		if got == http.StatusUnauthorized || got == http.StatusForbidden {
			t.Errorf("%s %s: public route rejected with %d — a guard is leaking onto it",
				tc.method, tc.path, got)
		}
	}
}

func TestServiceRoutesRequireServiceKey(t *testing.T) {
	app := newTestApp()

	paths := []string{
		"/api/v1/survey/flow-observations",
		"/api/v1/survey/entry-conversion-observations",
		"/api/v1/pipeline/extractions/struk",
		"/api/v1/pipeline/extractions/properti",
		"/api/v1/pipeline/extractions/gerai",
		"/api/v1/pipeline/simulations/monte-carlo",
	}

	for _, p := range paths {
		if got := do(t, app, http.MethodPost, p, nil); got != http.StatusUnauthorized {
			t.Errorf("POST %s without service key: got %d, want 401", p, got)
		}
	}

	for _, p := range paths {
		got := do(t, app, http.MethodPost, p, map[string]string{"X-Service-Key": testSvcKey})
		if got == http.StatusUnauthorized {
			t.Errorf("POST %s with valid service key: still 401", p)
		}
	}

	// The GET variant of survey is service-key protected too — it is raw
	// observation data, not a public analytics view.
	if got := do(t, app, http.MethodGet, "/api/v1/survey/flow-observations", nil); got != http.StatusUnauthorized {
		t.Errorf("GET survey without service key: got %d, want 401", got)
	}
}

func TestPremiumRequiresOperatorToken(t *testing.T) {
	const path = "/api/v1/premium/deep-analysis/abc"
	app := newTestApp()

	cases := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"no token", nil, http.StatusUnauthorized},
		{"malformed header", map[string]string{"Authorization": "Token abc"}, http.StatusUnauthorized},
		{"wrong secret", map[string]string{"Authorization": "Bearer " + "not.a.jwt"}, http.StatusUnauthorized},
		{"expired token", map[string]string{"Authorization": "Bearer " + token(t, "operator", "access", -time.Minute)}, http.StatusUnauthorized},
		{"authenticated but wrong role", map[string]string{"Authorization": "Bearer " + token(t, "surveyor", "access", time.Hour)}, http.StatusForbidden},
	}

	for _, tc := range cases {
		if got := do(t, app, http.MethodGet, path, tc.headers); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}

	for _, role := range []string{"operator", "admin"} {
		got := do(t, app, http.MethodGet, path, map[string]string{
			"Authorization": "Bearer " + token(t, role, "access", time.Hour),
		})
		if got == http.StatusUnauthorized || got == http.StatusForbidden {
			t.Errorf("role %q: rejected with %d, want to reach the handler", role, got)
		}
	}
}

// A service key must not stand in for a user token, and vice versa. Without
// this, a leaked pipeline key would read the paid tier.
func TestTiersDoNotSubstituteForEachOther(t *testing.T) {
	app := newTestApp()

	got := do(t, app, http.MethodGet, "/api/v1/premium/deep-analysis/abc",
		map[string]string{"X-Service-Key": testSvcKey})
	if got != http.StatusUnauthorized {
		t.Errorf("service key on premium route: got %d, want 401", got)
	}

	got = do(t, app, http.MethodPost, "/api/v1/survey/flow-observations",
		map[string]string{"Authorization": "Bearer " + token(t, "admin", "access", time.Hour)})
	if got != http.StatusUnauthorized {
		t.Errorf("admin JWT on service route: got %d, want 401", got)
	}
}

func TestHealthzIsOutsideTheAPIGroup(t *testing.T) {
	if got := do(t, newTestApp(), http.MethodGet, "/healthz", nil); got != http.StatusOK {
		t.Errorf("GET /healthz: got %d, want 200", got)
	}
}
