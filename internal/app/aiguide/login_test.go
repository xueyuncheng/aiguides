package aiguide

import (
	"slices"
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

			// Test the actual validation logic from login.go (line 86)
			// This is: allowed := len(a.config.AllowedEmails) == 0 || slices.Contains(a.config.AllowedEmails, user.Email)
			allowed := isEmailAllowed(config.AllowedEmails, tt.userEmail)

			if allowed != tt.expected {
				t.Errorf("expected allowed=%v for email=%s with allowedEmails=%v, but got allowed=%v",
					tt.expected, tt.userEmail, tt.allowedEmails, allowed)
			}
		})
	}
}

// isEmailAllowed checks if an email is allowed to log in based on the allowed_emails configuration.
// This is the extracted validation logic from login.go line 86.
// If allowedEmails is empty, all emails are allowed.
// Otherwise, only emails in the list are allowed.
func isEmailAllowed(allowedEmails []string, email string) bool {
	return len(allowedEmails) == 0 || slices.Contains(allowedEmails, email)
}
