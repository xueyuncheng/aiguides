package tools

import (
	"context"
	"testing"
)

func TestNewAudioTranscribeTool(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)

	audioTool, err := NewAudioTranscribeTool(db, store, nil, t.TempDir())
	if err != nil {
		t.Fatalf("NewAudioTranscribeTool() error = %v", err)
	}
	if audioTool == nil {
		t.Fatal("NewAudioTranscribeTool() returned nil tool")
	}
}

func TestSaveChatAudioAsset(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)

	asset, err := SaveChatAudioAsset(context.Background(), db, store, 11, "session-audio", "note.mp3", []byte("fake-audio"), "audio/mpeg")
	if err != nil {
		t.Fatalf("SaveChatAudioAsset() error = %v", err)
	}
	if asset.ID == 0 {
		t.Fatal("SaveChatAudioAsset() returned zero ID")
	}
	if asset.MimeType != "audio/mpeg" {
		t.Fatalf("asset.MimeType = %q, want %q", asset.MimeType, "audio/mpeg")
	}
}

func TestBuildAudioChunkWindows(t *testing.T) {
	windows := buildAudioChunkWindows(12 * 60 * 1000)
	if len(windows) != 3 {
		t.Fatalf("len(windows) = %d, want 3", len(windows))
	}
	if windows[1].startMs != (defaultAudioChunkDurationMs - defaultAudioChunkOverlapMs) {
		t.Fatalf("windows[1].startMs = %d", windows[1].startMs)
	}
	if windows[1].overlapStartMs == 0 {
		t.Fatal("expected overlapStartMs for second chunk")
	}
}

func TestJoinChunkTranscriptsDeduplicatesOverlap(t *testing.T) {
	joined := joinChunkTranscripts([]string{
		"This is a transcript overlap marker that should be removed across chunk boundaries for the merged result.",
		"overlap marker that should be removed across chunk boundaries for the merged result. And this continues.",
	})
	if joined == "" {
		t.Fatal("joinChunkTranscripts() returned empty string")
	}
	if joined != "This is a transcript overlap marker that should be removed across chunk boundaries for the merged result. And this continues." {
		t.Fatalf("joinChunkTranscripts() = %q", joined)
	}
}
