package v1

import (
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// RealtimeHandler issues short-lived one-time connect tickets for WS/SSE.
type RealtimeHandler struct {
	tickets *services.RealtimeTicketService
}

func NewRealtimeHandler(tickets *services.RealtimeTicketService) *RealtimeHandler {
	return &RealtimeHandler{tickets: tickets}
}

// IssueTicket godoc
// @Summary Issue realtime connect ticket
// @Tags realtime
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /admin/v1/realtime/ticket [post]
func (h *RealtimeHandler) IssueTicket(c *gin.Context) {
	if h.tickets == nil {
		response.ServerError(c)
		return
	}
	raw, ok := c.Get("user")
	if !ok {
		response.UnauthorizedError(c)
		return
	}
	user, ok := raw.(*models.User)
	if !ok || user == nil {
		response.UnauthorizedError(c)
		return
	}
	ticket, ttl, err := h.tickets.Issue(c.Request.Context(), user.ID)
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, gin.H{
		"ticket":     ticket,
		"expires_in": int(ttl.Seconds()),
	})
}
