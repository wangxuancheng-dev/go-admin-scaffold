package logger

import (
	"strings"
	"time"
)

// resolveLogLocation returns the IANA zone for daily rotation and retention.
// Empty, "local", or "Local" means the process default (time.Local).
// Invalid names fall back to time.Local.
func resolveLogLocation(name string) *time.Location {
	s := strings.TrimSpace(name)
	if s == "" || strings.EqualFold(s, "local") {
		return time.Local
	}
	loc, err := time.LoadLocation(s)
	if err != nil {
		return time.Local
	}
	return loc
}
