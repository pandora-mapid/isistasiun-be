package auth

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	requestvalidate "github.com/list-pandora/isi-stasiun-backend/internal/validate"
)

const refreshCookieName = "isi_stasiun_refresh"

type authService interface {
	Login(ctx context.Context, email, password string) (*TokenPairResponse, error)
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
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/logout", h.Logout)
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
