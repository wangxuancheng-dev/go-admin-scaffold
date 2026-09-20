package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/sse"
	"go-admin-scaffold/pkg/logger"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// SSEHandler handles Server-Sent Events with the same connect auth as WebSocket.
type SSEHandler struct {
	manager         *sse.Manager
	auth            *services.AuthService
	tickets         *services.RealtimeTicketService
	allowQueryToken bool
}

func NewSSEHandler(auth *services.AuthService, tickets *services.RealtimeTicketService, allowQueryToken bool) *SSEHandler {
	manager := sse.NewManager()
	go manager.Start()
	return &SSEHandler{manager: manager, auth: auth, tickets: tickets, allowQueryToken: allowQueryToken}
}

func (h *SSEHandler) HandleSSE(c *gin.Context) {
	token, ticket, _ := extractRealtimeToken(c, TokenExtractOptions{AllowQueryToken: h.allowQueryToken})
	if token == "" && ticket == "" {
		msg := "token is required (Authorization Bearer or ?ticket=)"
		if h.allowQueryToken {
			msg = "token is required (Authorization Bearer, ?ticket=, or ?token=)"
		}
		response.ParamError(c, msg)
		return
	}
	if h.auth == nil {
		response.ServerError(c)
		return
	}

	var userIDStr string
	if ticket != "" {
		if h.tickets == nil {
			response.UnauthorizedError(c)
			return
		}
		uid, err := h.tickets.Consume(c.Request.Context(), ticket)
		if err != nil {
			response.UnauthorizedError(c)
			return
		}
		user, err := h.auth.GetUserByID(c.Request.Context(), uid)
		if err != nil || user == nil {
			response.UnauthorizedError(c)
			return
		}
		userIDStr = clientIDFromUser(user)
	} else {
		claims, err := h.auth.ValidateToken(token)
		if err != nil {
			response.UnauthorizedError(c)
			return
		}
		user, err := h.auth.GetUserFromClaims(c.Request.Context(), claims)
		if err != nil || user == nil {
			response.UnauthorizedError(c)
			return
		}
		userIDStr = clientIDFromUser(user)
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	client := h.manager.Register(userIDStr)
	defer h.manager.Unregister(client)

	welcomeEvent := &sse.Event{
		Type: sse.EventTypeNotification,
		Data: fmt.Sprintf("Welcome %s!", userIDStr),
		Time: time.Now(),
	}
	h.manager.SendEvent(welcomeEvent)

	clientGone := c.Writer.CloseNotify()
	for {
		select {
		case <-clientGone:
			return
		case event := <-client.Messages:
			data, err := json.Marshal(event)
			if err != nil {
				logger.Warn(c.Request.Context(), "sse marshal event failed", "error", err)
				continue
			}
			_, _ = c.Writer.Write([]byte(fmt.Sprintf("id: %s\n", event.ID)))
			_, _ = c.Writer.Write([]byte(fmt.Sprintf("event: %s\n", event.Type)))
			_, _ = c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(data))))
			c.Writer.Flush()
		}
	}
}

func (h *SSEHandler) SendNotification(c *gin.Context) {
	var req struct {
		Type    string      `json:"type" binding:"required"`
		Data    interface{} `json:"data" binding:"required"`
		UserID  string      `json:"user_id"`
		GroupID string      `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if req.UserID != "" {
		h.manager.SendToUser(req.UserID, req.Type, req.Data)
	} else if req.GroupID != "" {
		h.manager.SendToGroup(req.GroupID, req.Type, req.Data)
	} else {
		h.manager.Broadcast(req.Type, req.Data)
	}
	response.Success(c, gin.H{"message": "Notification sent successfully"})
}

func (h *SSEHandler) JoinGroup(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	groupID, ok := requireGroupID(c)
	if !ok {
		return
	}
	h.manager.JoinGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully joined group"})
}

func (h *SSEHandler) LeaveGroup(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	groupID, ok := requireGroupID(c)
	if !ok {
		return
	}
	h.manager.LeaveGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully left group"})
}
