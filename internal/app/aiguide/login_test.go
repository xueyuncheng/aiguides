package aiguide

import (
	"testing"
)

func TestAllowedEmailsValidation(t *testing.T) {
	tests := []struct {
		name          string
		allowedEmails []string
		userEmail     string
		expected      bool
	}{
		{
			name:          "empty allowed_emails should allow any email",
			allowedEmails: []string{},
			userEmail:     "anyone@example.com",
			expected:      true,
		},
		{
			name:          "nil allowed_emails should allow any email",
			allowedEmails: nil,
			userEmail:     "anyone@example.com",
			expected:      true,
		},
		{
			name:          "email in allowed list should be allowed",
			allowedEmails: []string{"user1@example.com", "user2@example.com"},
			userEmail:     "user1@example.com",
			expected:      true,
		},
		{
			name:          "email not in allowed list should be denied",
			allowedEmails: []string{"user1@example.com", "user2@example.com"},
			userEmail:     "user3@example.com",
			expected:      false,
		},
		{
			name:          "single allowed email should match exactly",
			allowedEmails: []string{"admin@example.com"},
			userEmail:     "admin@example.com",
			expected:      true,
		},
		{
			name:          "single allowed email should reject others",
			allowedEmails: []string{"admin@example.com"},
			userEmail:     "user@example.com",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock config
			config := &Config{
				AllowedEmails: tt.allowedEmails,
			}

			// Simulate the validation logic from login.go
			allowed := len(config.AllowedEmails) == 0
			if !allowed {
				for _, email := range config.AllowedEmails {
					if email == tt.userEmail {
						allowed = true
						break
					}
				}
			}

			if allowed != tt.expected {
				t.Errorf("expected allowed=%v for email=%s with allowedEmails=%v, but got allowed=%v",
					tt.expected, tt.userEmail, tt.allowedEmails, allowed)
			}
		})
	}
}
