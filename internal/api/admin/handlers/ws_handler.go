package handlers

import (
	"net/http"
	"strings"
	"time"

	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/ws"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WSHandler handles WebSocket endpoints.
//
// Auth (connect):
//   - Prefer Authorization: Bearer <jwt>
//   - Or Sec-WebSocket-Protocol: access_token.<jwt>
//   - Legacy: GET /ws?token=<jwt> (may leak via access logs)
//
// Auth (control):
//   - POST /ws/join|leave|send — Authorization Bearer; identity from JWT context
type WSHandler struct {
	manager  *ws.Manager
	auth     *services.AuthService
	upgrader websocket.Upgrader
}

// NewWSHandler creates a WS handler. allowOrigins comes from CORS config;
// empty or ["*"] allows any Origin (dev only — production forbids CORS *).
func NewWSHandler(auth *services.AuthService, allowOrigins []string) *WSHandler {
	manager := ws.NewManager()
	go manager.Start()
	allowed := make(map[string]struct{}, len(allowOrigins))
	allowAll := false
	for _, o := range allowOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	return &WSHandler{
		manager: manager,
		auth:    auth,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
				if origin == "" {
					// Non-browser clients (CLI / native) often omit Origin.
					return true
				}
				if allowAll {
					return true
				}
				if len(allowed) == 0 {
					return false
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token, selectedProto := extractRealtimeToken(c)
	if token == "" {
		response.ParamError(c, "token is required (Authorization Bearer, Sec-WebSocket-Protocol access_token.<jwt>, or ?token=)")
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

	var respHeader http.Header
	if selectedProto != "" {
		respHeader = http.Header{}
		respHeader.Set("Sec-WebSocket-Protocol", selectedProto)
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, respHeader)
	if err != nil {
		return
	}

	clientID := clientIDFromUser(user)
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

func (h *WSHandler) LeaveGroup(c *gin.Context) {
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

func (h *WSHandler) SendMessage(c *gin.Context) {
	from, ok := currentUserID(c)
	if !ok {
		return
	}
	var message ws.Message
	if err := c.ShouldBindJSON(&message); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	message.From = from
	message.Timestamp = time.Now().Unix()
	h.manager.Broadcast <- &message
	response.Success(c, gin.H{"message": "Message sent successfully"})
}
