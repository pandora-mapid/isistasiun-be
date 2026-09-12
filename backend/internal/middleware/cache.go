package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

// ResponseCache TTL-caches GET responses by full request URL (path + query),
// using Fiber's built-in in-memory store — no Redis needed at this scale.
// CacheControl also stamps a `Cache-Control` header so a CDN/browser in front
// of the API can skip the round trip entirely during the TTL window.
//
// Meant for read-only endpoints whose data is produced by the batch pipeline
// and does not change between re-runs (../../../Context/02-BACKEND-SPEC.md
// §3.2) — nothing here is per-user or per-request, so a shared cache can't
// leak one caller's data to another.
func ResponseCache(ttlSeconds int) fiber.Handler {
	return cache.New(cache.Config{
		Expiration:   time.Duration(ttlSeconds) * time.Second,
		CacheControl: true,
	})
}
