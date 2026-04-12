package handlers

import (
	"app/internal/core/services"
	"app/internal/core/sse"
	"app/pkg/ginext"
	"app/pkg/response"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type SSEHandler struct {
	manager *sse.Manager
}

func NewSSEHandler() *SSEHandler {
	manager := sse.NewManager()
	go manager.Start() // Start the SSE manager
	return &SSEHandler{manager: manager}
}

// HandleSSE handles SSE connections
func (h *SSEHandler) HandleSSE(c *gin.Context) {
	userID := c.Query("user_id")
	token := c.Query("token")

	if userID == "" {
		response.ParamError(c, "user_id is required")
		return
	}

	if token == "" {
		response.ParamError(c, "token is required")
		return
	}

	authSvc, ok := ginext.GetService[*services.AuthService](c, "authService")
	if !ok {
		return
	}
	claims, err := authSvc.ValidateToken(token)
	if err != nil {
		response.UnauthorizedError(c)
		return
	}

	// Verify that the token's user_id matches the provided userID
	tokenUserID, ok := claims["username"]
	if !ok || tokenUserID != userID {
		response.UnauthorizedError(c)
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Register client
	client := h.manager.Register(userID)
	defer h.manager.Unregister(client)

	// Send welcome message
	welcomeEvent := &sse.Event{
		Type: sse.EventTypeNotification,
		Data: fmt.Sprintf("Welcome %s!", userID),
		Time: time.Now(),
	}
	h.manager.SendEvent(welcomeEvent)

	// Create channel for client disconnect
	clientGone := c.Writer.CloseNotify()

	// Start event loop
	for {
		select {
		case <-clientGone:
			log.Printf("Client %s disconnected from SSE", userID)
			return

		case event := <-client.Messages:
			// Convert event to JSON
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
				continue
			}

			// Write event to response
			c.Writer.Write([]byte(fmt.Sprintf("id: %s\n", event.ID)))
			c.Writer.Write([]byte(fmt.Sprintf("event: %s\n", event.Type)))
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(data))))
			c.Writer.Flush()
		}
	}
}

// SendNotification sends a notification to specific user(s) or broadcasts it
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

// JoinGroup adds a user to an SSE group
func (h *SSEHandler) JoinGroup(c *gin.Context) {
	userID := c.Query("user_id")
	groupID := c.Query("group_id")

	if userID == "" || groupID == "" {
		response.ParamError(c, "user_id and group_id are required")
		return
	}

	h.manager.JoinGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully joined group"})
}

// LeaveGroup removes a user from an SSE group
func (h *SSEHandler) LeaveGroup(c *gin.Context) {
	userID := c.Query("user_id")
	groupID := c.Query("group_id")

	if userID == "" || groupID == "" {
		response.ParamError(c, "user_id and group_id are required")
		return
	}

	h.manager.LeaveGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully left group"})
}
