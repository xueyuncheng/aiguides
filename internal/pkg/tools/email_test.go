package tools

import (
	"testing"

	"github.com/emersion/go-imap/v2"
	"google.golang.org/adk/model"
)

func TestNewEmailQueryTool(t *testing.T) {
	tool, err := NewEmailQueryTool()
	if err != nil {
		t.Fatalf("NewEmailQueryTool() failed: %v", err)
	}

	if tool == nil {
		t.Fatal("NewEmailQueryTool() returned nil tool")
	}
}

func TestNewEmailAgent(t *testing.T) {
	// Test with nil model (should still create the agent structure)
	agent, err := NewEmailAgent(nil)
	if err != nil {
		t.Fatalf("NewEmailAgent() failed: %v", err)
	}

	if agent == nil {
		t.Fatal("NewEmailAgent() returned nil agent")
	}
}

func TestNewEmailAgentWithModel(t *testing.T) {
	// Create a mock model
	var mockModel model.LLM = nil // In production, this would be a real model

	agent, err := NewEmailAgent(mockModel)
	if err != nil {
		t.Fatalf("NewEmailAgent() with model failed: %v", err)
	}

	if agent == nil {
		t.Fatal("NewEmailAgent() with model returned nil agent")
	}
}

func TestParseServerName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"imap.gmail.com:993", "imap.gmail.com"},
		{"imap.gmail.com", "imap.gmail.com"},
		{"localhost:143", "localhost"},
		{"mail.example.com:993", "mail.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseServerName(tt.input)
			if result != tt.expected {
				t.Errorf("parseServerName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     *imap.Address
		expected string
	}{
		{
			name:     "nil address",
			addr:     nil,
			expected: "",
		},
		{
			name: "with name",
			addr: &imap.Address{
				Name:    "John Doe",
				Mailbox: "john",
				Host:    "example.com",
			},
			expected: "John Doe <john@example.com>",
		},
		{
			name: "without name",
			addr: &imap.Address{
				Name:    "",
				Mailbox: "jane",
				Host:    "example.com",
			},
			expected: "jane@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddress(tt.addr)
			if result != tt.expected {
				t.Errorf("formatAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasSeenFlag(t *testing.T) {
	tests := []struct {
		name     string
		flags    []imap.Flag
		expected bool
	}{
		{
			name:     "empty flags",
			flags:    []imap.Flag{},
			expected: false,
		},
		{
			name:     "has seen flag",
			flags:    []imap.Flag{imap.FlagSeen},
			expected: true,
		},
		{
			name:     "has seen flag with others",
			flags:    []imap.Flag{imap.FlagAnswered, imap.FlagSeen, imap.FlagFlagged},
			expected: true,
		},
		{
			name:     "no seen flag",
			flags:    []imap.Flag{imap.FlagAnswered, imap.FlagFlagged},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSeenFlag(tt.flags)
			if result != tt.expected {
				t.Errorf("hasSeenFlag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTextFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		contains string
	}{
		{
			name: "simple email",
			body: `From: sender@example.com
To: recipient@example.com
Subject: Test

This is a test email.
`,
			contains: "This is a test email.",
		},
		{
			name: "email with extra newlines",
			body: `From: sender@example.com
To: recipient@example.com
Subject: Test


Hello


World

`,
			contains: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromBody(tt.body)
			if result == "" {
				t.Error("extractTextFromBody() returned empty string")
			}
		})
	}
}
