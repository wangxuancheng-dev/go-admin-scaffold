package middleware

import (
	"go-admin-scaffold/pkg/i18n"

	"github.com/gin-gonic/gin"
)

const (
	defaultLocale = "en"
	localeKey     = "locale"
)

// I18n returns a middleware that handles locale selection
func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		i18nInst := i18n.GetInstance()

		// Try to get locale from query parameter
		locale := c.Query(localeKey)

		// If not in query, try header
		if locale == "" {
			locale = c.GetHeader("Accept-Language")
		}

		// If still not found, use default
		if locale == "" {
			locale = i18nInst.GetDefaultLocale()
		}

		// Store locale in context
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
