package assistant

import (
	"testing"
)

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
