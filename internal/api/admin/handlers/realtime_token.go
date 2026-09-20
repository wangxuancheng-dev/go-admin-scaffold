package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const wsAccessTokenPrefix = "access_token."

// extractRealtimeToken resolves the JWT for WS/SSE connect.
// Preference order (avoids putting secrets in URLs when possible):
//  1. Authorization: Bearer <jwt>
//  2. Sec-WebSocket-Protocol: access_token.<jwt> (or a bare JWT-looking subprotocol)
//  3. ?token=<jwt> (legacy; may appear in access logs / Referer — prefer 1 or 2)
func extractRealtimeToken(c *gin.Context) (token string, selectedSubprotocol string) {
	if auth := c.GetHeader("Authorization"); len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		if t := strings.TrimSpace(auth[7:]); t != "" {
			return t, ""
		}
	}

	if proto := c.GetHeader("Sec-WebSocket-Protocol"); proto != "" {
		for _, p := range strings.Split(proto, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.HasPrefix(p, wsAccessTokenPrefix) {
				t := strings.TrimPrefix(p, wsAccessTokenPrefix)
				if t != "" {
					return t, p
				}
			}
			// Bare JWT as subprotocol (three base64url segments).
			if strings.Count(p, ".") == 2 && !strings.EqualFold(p, "bearer") {
				return p, p
			}
		}
	}

	return strings.TrimSpace(c.Query("token")), ""
}
