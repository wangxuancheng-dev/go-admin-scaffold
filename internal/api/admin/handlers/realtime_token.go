package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	wsAccessTokenPrefix = "access_token."
	wsTicketPrefix      = "ticket."
)

// TokenExtractOptions controls legacy query JWT acceptance.
type TokenExtractOptions struct {
	AllowQueryToken bool
}

// extractRealtimeToken resolves connect credentials for WS/SSE.
// Preference:
//  1. Authorization: Bearer <jwt>
//  2. Sec-WebSocket-Protocol: access_token.<jwt> | ticket.<id> | bare JWT
//  3. ?ticket=<id> (short-lived, OK in URLs)
//  4. ?token=<jwt> only when AllowQueryToken (disabled in production by default)
func extractRealtimeToken(c *gin.Context, opt TokenExtractOptions) (token string, ticket string, selectedSubprotocol string) {
	if auth := c.GetHeader("Authorization"); len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		if t := strings.TrimSpace(auth[7:]); t != "" {
			return t, "", ""
		}
	}

	if proto := c.GetHeader("Sec-WebSocket-Protocol"); proto != "" {
		for _, p := range strings.Split(proto, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.HasPrefix(p, wsTicketPrefix) {
				if t := strings.TrimPrefix(p, wsTicketPrefix); t != "" {
					return "", t, p
				}
			}
			if strings.HasPrefix(p, wsAccessTokenPrefix) {
				if t := strings.TrimPrefix(p, wsAccessTokenPrefix); t != "" {
					return t, "", p
				}
			}
			if strings.Count(p, ".") == 2 && !strings.EqualFold(p, "bearer") {
				return p, "", p
			}
		}
	}

	if t := strings.TrimSpace(c.Query("ticket")); t != "" {
		return "", t, ""
	}

	if opt.AllowQueryToken {
		if t := strings.TrimSpace(c.Query("token")); t != "" {
			return t, "", ""
		}
	}

	return "", "", ""
}
