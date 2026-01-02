package aiguide

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadAvatar(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		url         string
		expectError bool
		expectData  bool
		expectMIME  string
		errorMsg    string
	}{
		{
			name: "successful download",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			name: "successful jpeg download",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/jpeg")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-jpeg-data"))
				}))
			},
			expectError: false,
			expectData:  true,
			expectMIME:  "image/jpeg",
		},
		{
			name: "empty URL",
			setupServer: func() *httptest.Server {
				return nil
			},
			url:         "",
			expectError: true,
			expectData:  false,
			errorMsg:    "empty avatar URL",
		},
		{
			name: "http URL rejected",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/png")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-image-data"))
				}))
			},
			expectError: true,
			expectData:  false,
			errorMsg:    "only HTTPS URLs are allowed",
		},
		{
			name: "404 not found",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: true,
			expectData:  false,
			errorMsg:    "unexpected status code",
		},
		{
			name: "missing content type header",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-image-data"))
				}))
			},
			expectError: true,
			expectData:  false,
			errorMsg:    "invalid MIME type", // httptest sets text/plain by default
		},
		{
			name: "invalid MIME type",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("<html>fake data</html>"))
				}))
			},
			expectError: true,
			expectData:  false,
			errorMsg:    "invalid MIME type",
		},
		{
			name: "content-type with charset",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/png; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("fake-image-data"))
				}))
			},
			expectError: false,
			expectData:  true,
			expectMIME:  "image/png",
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

					// Set up the test with the server's client to bypass TLS verification
					if url == "" {
						url = server.URL
					}

					// Temporarily replace http.DefaultClient for testing
					oldClient := http.DefaultClient
					if server.Client() != nil {
						http.DefaultClient = server.Client()
					}
					defer func() {
						http.DefaultClient = oldClient
					}()
				}
			}

			data, mimeType, err := downloadAvatar(url)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && tt.errorMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q but got %q", tt.errorMsg, err.Error())
				}
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
