package assistant

import (
	"aiguide/internal/pkg/constant"
	"iter"
	"strings"
	"testing"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type testEvents []*session.Event

func (e testEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, event := range e {
			if !yield(event) {
				return
			}
		}
	}
}

func (e testEvents) Len() int {
	return len(e)
}

func (e testEvents) At(i int) *session.Event {
	return e[i]
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("generateSessionID() should not return empty string")
	}

	if id1 == id2 {
		t.Error("generateSessionID() should generate unique IDs")
	}

	if len(id1) < 10 {
		t.Errorf("generateSessionID() generated ID too short: %s", id1)
	}
}

func TestRandomString(t *testing.T) {
	str1 := randomString(8)
	str2 := randomString(8)

	if len(str1) != 8 {
		t.Errorf("randomString(8) should return 8 characters, got %d", len(str1))
	}

	if len(str2) != 8 {
		t.Errorf("randomString(8) should return 8 characters, got %d", len(str2))
	}

	if str1 == str2 {
		t.Error("randomString() should generate different strings")
	}
}

func TestBuildMessageEventsPreservesPDFFileInHistory(t *testing.T) {
	part := genai.NewPartFromText(strings.Join([]string{
		"<!-- PDF_FILE: {\"name\":\"report.pdf\"} -->",
		"[PDF extracted text]",
		"File: report.pdf",
		"",
		"[Page 1]",
		"Alpha content",
	}, "\n"))

	events := testEvents{&session.Event{
		ID:        "event-1",
		Timestamp: time.Now(),
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{part},
			},
		},
	}}

	messages := buildMessageEvents(events, constant.LocaleEN)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Content != "" {
		t.Fatalf("messages[0].Content = %q, want empty", messages[0].Content)
	}
	if len(messages[0].Files) != 1 {
		t.Fatalf("len(messages[0].Files) = %d, want 1", len(messages[0].Files))
	}
	if messages[0].Files[0].MimeType != pdfMimeType {
		t.Fatalf("messages[0].Files[0].MimeType = %q, want %q", messages[0].Files[0].MimeType, pdfMimeType)
	}
	if messages[0].Files[0].Name != "report.pdf" {
		t.Fatalf("messages[0].Files[0].Name = %q, want %q", messages[0].Files[0].Name, "report.pdf")
	}
}

func TestBuildMessageEventsStripsFileNamesMetadataAfterUserContext(t *testing.T) {
	part := genai.NewPartFromText(strings.Join([]string{
		"<user_context>",
		"memory",
		"</user_context>",
		"<!-- FILE_NAMES: [\"report.pdf\"] -->",
		"请总结这份合同",
	}, "\n"))

	events := testEvents{&session.Event{
		ID:        "event-2",
		Timestamp: time.Now(),
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{part},
			},
		},
	}}

	messages := buildMessageEvents(events, constant.LocaleEN)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Content != "请总结这份合同" {
		t.Fatalf("messages[0].Content = %q, want %q", messages[0].Content, "请总结这份合同")
	}
	if len(messages[0].FileNames) != 1 || messages[0].FileNames[0] != "report.pdf" {
		t.Fatalf("messages[0].FileNames = %#v, want report.pdf", messages[0].FileNames)
	}
	if strings.Contains(messages[0].Content, "FILE_NAMES") {
		t.Fatalf("messages[0].Content = %q, should not contain FILE_NAMES metadata", messages[0].Content)
	}
}

func TestExtractFileNamesMetadataHandlesLeadingBlankLines(t *testing.T) {
	text := "\n\n<!-- FILE_NAMES: [\"report.pdf\"] -->\n请总结这份合同"

	content, fileNames, ok := extractFileNamesMetadata(text)
	if !ok {
		t.Fatal("extractFileNamesMetadata() = false, want true")
	}
	if content != "请总结这份合同" {
		t.Fatalf("content = %q, want %q", content, "请总结这份合同")
	}
	if len(fileNames) != 1 || fileNames[0] != "report.pdf" {
		t.Fatalf("fileNames = %#v, want report.pdf", fileNames)
	}
}
