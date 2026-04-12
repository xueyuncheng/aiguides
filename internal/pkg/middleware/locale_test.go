package middleware

import (
	"aiguide/internal/pkg/constant"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocaleFromLanguageHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "defaults to english",
			header: "",
			want:   constant.LocaleEN,
		},
		{
			name:   "english locale",
			header: "en-US,en;q=0.9",
			want:   constant.LocaleEN,
		},
		{
			name:   "chinese locale",
			header: "zh-CN,zh;q=0.9,en;q=0.8",
			want:   constant.LocaleZH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LocaleFromLanguageHeader(tt.header); got != tt.want {
				t.Fatalf("LocaleFromLanguageHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLocaleMiddlewareSetsContextValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	router := gin.New()
	router.Use(Locale())
	router.GET("/api/test", func(c *gin.Context) {
		locale, ok := c.Get(constant.ContextKeyLocale)
		if !ok {
			c.String(http.StatusInternalServerError, "missing locale")
			return
		}

		c.String(http.StatusOK, locale.(string))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if got := w.Body.String(); got != constant.LocaleZH {
		t.Fatalf("locale middleware returned %q, want %q", got, constant.LocaleZH)
	}
}
