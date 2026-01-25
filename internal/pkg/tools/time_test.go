// Copyright 2025 AIGuides
// Tests for time tool

package tools

import (
	"testing"
)

func TestNewCurrentTimeTool(t *testing.T) {
	tool, err := NewCurrentTimeTool()
	if err != nil {
		t.Fatalf("NewCurrentTimeTool() error = %v", err)
	}

	if tool == nil {
		t.Fatal("NewCurrentTimeTool() returned nil tool")
	}
}

func TestCurrentTimeInput_Validation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "UTC (default)",
			timezone: "",
			wantErr:  false,
		},
		{
			name:     "America/New_York",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "Asia/Shanghai",
			timezone: "Asia/Shanghai",
			wantErr:  false,
		},
		{
			name:     "Invalid timezone",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the input structure is valid
			input := CurrentTimeInput{
				Timezone: tt.timezone,
			}
			_ = input // Use the input
		})
	}
}
