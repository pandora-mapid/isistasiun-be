package premium

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/auth"
	"github.com/list-pandora/isi-stasiun-backend/internal/middleware"
	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the premium routes. The JWT + operator-role guard is
// mounted on the /premium prefix in router.go, not here — see the comment
// there for why the guard must be bound to a path.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/premium/deep-analysis/:station_id", h.DeepAnalysis)
}

func (h *Handler) DeepAnalysis(c *fiber.Ctx) error {
	stationID := c.Params("station_id")
	if stationID == "" {
		return response.BadRequest(c, "station_id is required")
	}

	// An operator is scoped to the single station on its token (section 4.1);
	// admin has no station claim and reaches every station. RequireRole above
	// already rejects anything that isn't operator or admin.
	if role, _ := c.Locals(middleware.CtxRoleKey).(string); role == string(auth.RoleOperator) {
		scoped, _ := c.Locals(middleware.CtxStationIDKey).(string)
		if scoped == "" || scoped != stationID {
			return response.Forbidden(c, "operator can only access its own station")
		}
	}

	data, err := h.svc.DeepAnalysis(c.Context(), stationID)
	if errors.Is(err, ErrStationNotFound) {
		return response.NotFound(c, "station not found")
	}
	if err != nil {
		return response.Internal(c, "failed to build deep analysis")
	}
	return response.OK(c, data)
}
