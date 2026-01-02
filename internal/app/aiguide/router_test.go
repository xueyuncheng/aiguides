package aiguide

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadAvatar(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		url           string
		expectError   bool
		expectData    bool
		expectMIME    string
	}{
		{
			name: "successful download",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/png")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-image-data"))
				}))
			},
			expectError: false,
			expectData:  true,
			expectMIME:  "image/png",
		},
		{
			name: "empty URL",
			setupServer: func() *httptest.Server {
				return nil
			},
			url:         "",
			expectError: true,
			expectData:  false,
		},
		{
			name: "404 not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: true,
			expectData:  false,
		},
		{
			name: "no content type header",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Don't set content type, httptest will set text/plain
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-image-data"))
				}))
			},
			expectError: false,
			expectData:  true,
			expectMIME:  "", // Don't check MIME in this case as httptest adds default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			url := tt.url
			
			if tt.setupServer != nil {
				server = tt.setupServer()
				if server != nil {
					defer server.Close()
					if url == "" {
						url = server.URL
					}
				}
			}

			data, mimeType, err := downloadAvatar(url)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectData && len(data) == 0 {
				t.Errorf("expected data but got none")
			}
			if !tt.expectData && len(data) > 0 {
				t.Errorf("expected no data but got some")
			}

			if tt.expectMIME != "" && !strings.Contains(mimeType, tt.expectMIME) {
				t.Errorf("expected MIME type %s but got %s", tt.expectMIME, mimeType)
			}
		})
	}
}
