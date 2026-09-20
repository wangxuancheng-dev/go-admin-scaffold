package middleware

import (
	"go-admin-scaffold/pkg/i18n"

	"github.com/gin-gonic/gin"
)

const (
	defaultLocale = "en"
	localeKey     = "locale"
)

// I18n returns middleware that stores the request locale on the gin context.
// Pass the process i18n instance from the composition root (after i18n.Init).
// If inst is nil, defaults to "en" without calling the i18n singleton.
func I18n(inst *i18n.I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.Query(localeKey)
		if locale == "" {
			locale = c.GetHeader("Accept-Language")
		}
		if locale == "" {
			if inst != nil {
				locale = inst.GetDefaultLocale()
			} else {
				locale = defaultLocale
			}
		}
		c.Set(localeKey, locale)
		c.Next()
	}
}

// GetLocale returns the current locale from gin context
func GetLocale(c *gin.Context) string {
	if locale, exists := c.Get(localeKey); exists {
		return locale.(string)
	}
	return defaultLocale
}
