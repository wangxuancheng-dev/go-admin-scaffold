package v1

import (
	"fmt"
	"os"
	"path/filepath"

	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

// I18nHandler serves locale metadata and translation files.
type I18nHandler struct{}

func NewI18nHandler() *I18nHandler {
	return &I18nHandler{}
}

func (h *I18nHandler) GetLocales(c *gin.Context) {
	response.Success(c, []string{"zh", "en"})
}

func (h *I18nHandler) GetTranslations(c *gin.Context) {
	locale := c.Query("locale")
	if locale == "" {
		response.ParamError(c, "locale parameter is required")
		return
	}
	if locale != "zh" && locale != "en" {
		response.ParamError(c, "unsupported locale")
		return
	}

	filePath := filepath.Join("locales", fmt.Sprintf("%s.yml", locale))
	data, err := os.ReadFile(filePath)
	if err != nil {
		response.Error(c, response.CodeNotFound, "translation file not found")
		return
	}

	var translations map[string]interface{}
	if err := yaml.Unmarshal(data, &translations); err != nil {
		response.Error(c, response.CodeServerError, "failed to parse translation file")
		return
	}
	response.Success(c, translations)
}
