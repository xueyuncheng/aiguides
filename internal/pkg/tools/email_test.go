package tools

import (
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
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

func TestNewSendEmailTool(t *testing.T) {
	tool, err := NewSendEmailTool()
	if err != nil {
		t.Fatalf("NewSendEmailTool() failed: %v", err)
	}

	if tool == nil {
		t.Fatal("NewSendEmailTool() returned nil tool")
	}
}

func TestValidateSendEmailInput(t *testing.T) {
	tests := []struct {
		name    string
		input   SendEmailInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: SendEmailInput{
				To:      []string{"Alice <alice@example.com>", "bob@example.com"},
				Subject: "Status update",
				Body:    "All good",
			},
		},
		{
			name: "missing recipients",
			input: SendEmailInput{
				Subject: "Status update",
				Body:    "All good",
			},
		},
		{
			name: "invalid recipient",
			input: SendEmailInput{
				To:      []string{"invalid-email"},
				Subject: "Status update",
				Body:    "All good",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSendEmailInput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("validateSendEmailInput() error = nil, want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("validateSendEmailInput() error = %v", err)
			}
			wantToLen := len(tt.input.To)
			if len(got.To) != wantToLen {
				t.Fatalf("len(got.To) = %d, want %d", len(got.To), wantToLen)
			}
		})
	}
}

func TestPopulateDefaultRecipient(t *testing.T) {
	input := &SendEmailInput{Subject: "Status update", Body: "All good"}
	defaulted, err := populateDefaultRecipient(input, "sender@example.com")
	if err != nil {
		t.Fatalf("populateDefaultRecipient() error = %v", err)
	}
	if !defaulted {
		t.Fatal("populateDefaultRecipient() defaulted = false, want true")
	}
	if len(input.To) != 1 || input.To[0] != "sender@example.com" {
		t.Fatalf("populateDefaultRecipient() To = %v, want [sender@example.com]", input.To)
	}
}

func TestPopulateDefaultRecipientKeepsExistingTo(t *testing.T) {
	input := &SendEmailInput{To: []string{"target@example.com"}, Subject: "Status update", Body: "All good"}
	defaulted, err := populateDefaultRecipient(input, "sender@example.com")
	if err != nil {
		t.Fatalf("populateDefaultRecipient() error = %v", err)
	}
	if defaulted {
		t.Fatal("populateDefaultRecipient() defaulted = true, want false")
	}
	if len(input.To) != 1 || input.To[0] != "target@example.com" {
		t.Fatalf("populateDefaultRecipient() To = %v, want [target@example.com]", input.To)
	}
}

func TestBuildEmailMessage(t *testing.T) {
	message, err := buildEmailMessage("sender@example.com", "测试发件人", SendEmailInput{
		To:      []string{"to@example.com"},
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		Subject: "项目进展",
		Body:    "第一行\n第二行",
	}, []string{"to@example.com", "cc@example.com", "bcc@example.com"})
	if err != nil {
		t.Fatalf("buildEmailMessage() error = %v", err)
	}

	content := string(message)
	checks := []string{
		"From: =?UTF-8?",
		"To: to@example.com",
		"Cc: cc@example.com",
		"Subject: =?UTF-8?",
		"Content-Type: text/plain; charset=UTF-8",
		"第一行\r\n第二行",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Fatalf("buildEmailMessage() missing %q in %q", check, content)
		}
	}

	if strings.Contains(content, "Bcc:") {
		t.Fatalf("buildEmailMessage() should not include Bcc header: %q", content)
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
