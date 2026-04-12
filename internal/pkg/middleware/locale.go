package middleware

import (
	"aiguide/internal/pkg/constant"
	"strings"

	"github.com/gin-gonic/gin"
)

func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(constant.ContextKeyLocale, LocaleFromLanguageHeader(c.GetHeader("Accept-Language")))
		c.Next()
	}
}

func LocaleFromLanguageHeader(header string) string {
	for _, part := range strings.Split(header, ",") {
		language := strings.TrimSpace(part)
		if language == "" {
			continue
		}

		if idx := strings.Index(language, ";"); idx >= 0 {
			language = language[:idx]
		}

		language = strings.ToLower(strings.TrimSpace(language))
		if language == "" {
			continue
		}

		if strings.HasPrefix(language, constant.LocaleZH) {
			return constant.LocaleZH
		}

		return constant.LocaleEN
	}

	return constant.LocaleEN
}
