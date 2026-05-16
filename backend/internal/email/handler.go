package email

import (
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// Handler exposes admin email endpoints.
type Handler struct {
	svc *EmailService
}

// NewHandler constructs a Handler.
func NewHandler(svc *EmailService) *Handler {
	return &Handler{svc: svc}
}

// HandleTestNotification sends a test email to the currently authenticated
// admin user.  Returns 503 if SMTP delivery fails.
//
// POST /api/v1/admin/notifications/test
func (h *Handler) HandleTestNotification(w http.ResponseWriter, r *http.Request) {
	userID := ctxkeys.UserIDFromCtx(r.Context())
	if userID == "" {
		api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	email, err := h.svc.GetUserEmail(r.Context(), userID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "USER_NOT_FOUND", "could not look up user email")
		return
	}

	if err := h.svc.TestSend(email); err != nil {
		api.WriteError(w, http.StatusServiceUnavailable, "EMAIL_UNAVAILABLE", "failed to send test email: "+err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{
		"recipient": email,
		"status":    "sent",
	})
}
