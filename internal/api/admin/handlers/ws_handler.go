package handlers

import (
	"net/http"
	"strings"
	"time"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/ws"
	"go-admin-scaffold/pkg/response"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// WSHandler handles WebSocket endpoints (github.com/coder/websocket).
//
// Connect auth (in order): Bearer JWT, Sec-WebSocket-Protocol access_token./ticket.,
// ?ticket=, optional legacy ?token= when allowQueryToken.
type WSHandler struct {
	manager         *ws.Manager
	auth            *services.AuthService
	tickets         *services.RealtimeTicketService
	allowQueryToken bool
	checkOrigin     func(*http.Request) bool
}

// NewWSHandler creates a WS handler. allowOrigins comes from CORS config;
// empty or ["*"] allows any Origin (dev only — production forbids CORS *).
func NewWSHandler(auth *services.AuthService, tickets *services.RealtimeTicketService, allowOrigins []string, allowQueryToken bool) *WSHandler {
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
		manager:         manager,
		auth:            auth,
		tickets:         tickets,
		allowQueryToken: allowQueryToken,
		checkOrigin: func(r *http.Request) bool {
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
	}
}

func (h *WSHandler) resolveConnectUser(c *gin.Context) (*models.User, string, bool) {
	token, ticket, selectedProto := extractRealtimeToken(c, TokenExtractOptions{AllowQueryToken: h.allowQueryToken})
	if token == "" && ticket == "" {
		msg := "token is required (Authorization Bearer, Sec-WebSocket-Protocol, or ?ticket=)"
		if h.allowQueryToken {
			msg = "token is required (Authorization Bearer, Sec-WebSocket-Protocol, ?ticket=, or ?token=)"
		}
		response.ParamError(c, msg)
		return nil, "", false
	}
	if h.auth == nil {
		response.ServerError(c)
		return nil, "", false
	}

	if ticket != "" {
		if h.tickets == nil {
			response.UnauthorizedError(c)
			return nil, "", false
		}
		userID, err := h.tickets.Consume(c.Request.Context(), ticket)
		if err != nil {
			response.UnauthorizedError(c)
			return nil, "", false
		}
		user, err := h.auth.GetUserByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			response.UnauthorizedError(c)
			return nil, "", false
		}
		return user, selectedProto, true
	}

	claims, err := h.auth.ValidateToken(token)
	if err != nil {
		response.UnauthorizedError(c)
		return nil, "", false
	}
	user, err := h.auth.GetUserFromClaims(c.Request.Context(), claims)
	if err != nil || user == nil {
		response.UnauthorizedError(c)
		return nil, "", false
	}
	return user, selectedProto, true
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	user, selectedProto, ok := h.resolveConnectUser(c)
	if !ok {
		return
	}

	if h.checkOrigin != nil && !h.checkOrigin(c.Request) {
		response.Forbidden(c, "origin not allowed")
		return
	}

	opts := &websocket.AcceptOptions{
		// Origin already validated via checkOrigin above.
		InsecureSkipVerify: true,
	}
	if selectedProto != "" {
		opts.Subprotocols = []string{selectedProto}
	}

	conn, err := websocket.Accept(c.Writer, c.Request, opts)
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
