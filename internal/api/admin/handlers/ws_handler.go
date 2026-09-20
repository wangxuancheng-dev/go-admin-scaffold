package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/ws"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler handles WebSocket endpoints.
//
// Auth strategy:
//   - GET /ws requires query `token` (JWT). Identity is taken from JWT claims only.
//   - Optional query `user_id` must match claims username when provided (legacy clients).
//   - Mutating endpoints (/ws/join|/leave|/send) require Authorization Bearer via JWT middleware.
type WSHandler struct {
	manager *ws.Manager
	auth    *services.AuthService
}

func NewWSHandler(auth *services.AuthService) *WSHandler {
	manager := ws.NewManager()
	go manager.Start()
	return &WSHandler{manager: manager, auth: auth}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.ParamError(c, "token is required")
		return
	}
	if h.auth == nil {
		response.ServerError(c)
		return
	}

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

	// Legacy optional check: if client sends user_id, it must equal username from JWT.
	if q := c.Query("user_id"); q != "" && q != user.Username && q != strconv.FormatUint(uint64(user.ID), 10) {
		response.UnauthorizedError(c)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.ServerError(c)
		return
	}

	clientID := user.Username
	if clientID == "" {
		clientID = fmt.Sprintf("%d", user.ID)
	}
	client := &ws.Client{
		ID:      clientID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Manager: h.manager,
		Groups:  make(map[string]bool),
	}
	h.manager.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

func (h *WSHandler) JoinGroup(c *gin.Context) {
	userID := c.Query("user_id")
	groupID := c.Query("group_id")
	if userID == "" || groupID == "" {
		response.ParamError(c, "user_id and group_id are required")
		return
	}
	h.manager.JoinGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully joined group"})
}

func (h *WSHandler) LeaveGroup(c *gin.Context) {
	userID := c.Query("user_id")
	groupID := c.Query("group_id")
	if userID == "" || groupID == "" {
		response.ParamError(c, "user_id and group_id are required")
		return
	}
	h.manager.LeaveGroup(groupID, userID)
	response.Success(c, gin.H{"message": "Successfully left group"})
}

func (h *WSHandler) SendMessage(c *gin.Context) {
	var message ws.Message
	if err := c.ShouldBindJSON(&message); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	message.Timestamp = time.Now().Unix()
	h.manager.Broadcast <- &message
	response.Success(c, gin.H{"message": "Message sent successfully"})
}
