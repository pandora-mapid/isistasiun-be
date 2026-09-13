package auth

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/middleware"
	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	requestvalidate "github.com/list-pandora/isi-stasiun-backend/internal/validate"
)

const refreshCookieName = "isi_stasiun_refresh"

type authService interface {
	Login(ctx context.Context, email, password string) (*TokenPairResponse, error)
	Register(ctx context.Context, email, password string) (*TokenPairResponse, error)
	Upgrade(ctx context.Context, operatorID string) (*TokenPairResponse, error)
	CreateOperator(ctx context.Context, email, password, stationID string) (*Operator, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPairResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

type CookieConfig struct {
	Secure bool
	Domain string
	TTL    time.Duration
}

type Handler struct {
	svc    authService
	cookie CookieConfig
}

func NewHandler(svc authService, cookie CookieConfig) *Handler {
	return &Handler{svc: svc, cookie: cookie}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/upgrade", h.Upgrade)
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/logout", h.Logout)
	r.Post("/admin/operators", h.CreateOperator)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if message := requestvalidate.Struct(&req); message != "" {
		return response.BadRequest(c, message)
	}

	tokens, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return response.Unauthorized(c, "invalid email or password")
	}
	if err != nil {
		return response.Internal(c, "login failed")
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return response.OKWithMessage(c, "Berhasil masuk", tokens)
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if message := requestvalidate.Struct(&req); message != "" {
		return response.BadRequest(c, message)
	}

	tokens, err := h.svc.Register(c.Context(), req.Email, req.Password)
	if errors.Is(err, ErrEmailTaken) {
		return response.BadRequest(c, "email already registered")
	}
	if err != nil {
		return response.Internal(c, "registration failed")
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return response.OKWithMessage(c, "Berhasil daftar", tokens)
}

// Upgrade is the "pretend to pay" flow: no payment gateway, the caller's own
// account flips from RoleUser to RolePremium on request. Reads the operator
// id RequireAuth already put in Locals — the request body carries nothing.
func (h *Handler) Upgrade(c *fiber.Ctx) error {
	operatorID, _ := c.Locals(middleware.CtxUserIDKey).(string)
	tokens, err := h.svc.Upgrade(c.Context(), operatorID)
	if errors.Is(err, ErrNotEligibleForUpgrade) {
		return response.BadRequest(c, "only a user account can upgrade to premium")
	}
	if errors.Is(err, ErrNotFound) {
		return response.Unauthorized(c, "invalid session")
	}
	if err != nil {
		return response.Internal(c, "upgrade failed")
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return response.OKWithMessage(c, "Berhasil upgrade ke premium", tokens)
}

// CreateOperator is admin-only (guarded on the /admin prefix in router.go).
// Unlike Register, it never logs the caller in as the new account — the
// admin stays the admin, the new operator gets its own credentials to log in
// with separately.
func (h *Handler) CreateOperator(c *fiber.Ctx) error {
	var req CreateOperatorRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if message := requestvalidate.Struct(&req); message != "" {
		return response.BadRequest(c, message)
	}

	op, err := h.svc.CreateOperator(c.Context(), req.Email, req.Password, req.StationID)
	if errors.Is(err, ErrEmailTaken) {
		return response.BadRequest(c, "email already registered")
	}
	if err != nil {
		return response.Internal(c, "failed to create operator")
	}
	return response.OKWithMessage(c, "Operator dibuat", UserResponse{
		ID:        op.ID,
		Email:     op.Email,
		Role:      op.Role,
		StationID: op.StationID,
	})
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	refreshToken := c.Cookies(refreshCookieName)
	if refreshToken == "" {
		return response.Unauthorized(c, "refresh session is missing")
	}

	tokens, err := h.svc.Refresh(c.Context(), refreshToken)
	if err != nil {
		h.clearRefreshCookie(c)
		if errors.Is(err, ErrInvalidRefreshToken) {
			return response.Unauthorized(c, "invalid or expired refresh session")
		}
		return response.Internal(c, "failed to refresh session")
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return response.OKWithMessage(c, "Sesi diperbarui", tokens)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	refreshToken := c.Cookies(refreshCookieName)
	if err := h.svc.Logout(c.Context(), refreshToken); err != nil {
		return response.Internal(c, "failed to end session")
	}
	h.clearRefreshCookie(c)
	return response.OKWithMessage(c, "Berhasil keluar", fiber.Map{})
}

func (h *Handler) setRefreshCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Domain:   h.cookie.Domain,
		MaxAge:   int(h.cookie.TTL.Seconds()),
		Expires:  time.Now().Add(h.cookie.TTL),
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (h *Handler) clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Domain:   h.cookie.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
