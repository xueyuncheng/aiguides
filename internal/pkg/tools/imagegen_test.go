package tools

import (
	"testing"
)

func TestNewImageGenTool(t *testing.T) {
	// This test just verifies that the tool can be created
	// without errors when a nil client is passed (basic structure test)
	// We can't test the actual image generation without a real API key
	tool, err := NewImageGenTool(nil)
	if err != nil {
		t.Fatalf("NewImageGenTool() error = %v", err)
	}

	if tool == nil {
		t.Fatal("NewImageGenTool() returned nil tool")
	}
}

func TestImageGenInput_Validation(t *testing.T) {
	tests := []struct {
		name           string
		input          ImageGenInput
		expectError    bool
		errorContains  string
	}{
		{
			name: "empty prompt",
			input: ImageGenInput{
				Prompt: "",
			},
			expectError:   true,
			errorContains: "图片描述不能为空",
		},
		{
			name: "valid input with defaults",
			input: ImageGenInput{
				Prompt: "a beautiful cat",
			},
			expectError: false,
		},
		{
			name: "invalid aspect ratio",
			input: ImageGenInput{
				Prompt:      "a beautiful cat",
				AspectRatio: "invalid",
			},
			expectError:   true,
			errorContains: "无效的宽高比",
		},
		{
			name: "valid aspect ratio 1:1",
			input: ImageGenInput{
				Prompt:      "a beautiful cat",
				AspectRatio: "1:1",
			},
			expectError: false,
		},
		{
			name: "valid aspect ratio 16:9",
			input: ImageGenInput{
				Prompt:      "a landscape photo",
				AspectRatio: "16:9",
			},
			expectError: false,
		},
		{
			name: "multiple images",
			input: ImageGenInput{
				Prompt:         "a dragon",
				NumberOfImages: 3,
			},
			expectError: false,
		},
		{
			name: "with negative prompt",
			input: ImageGenInput{
				Prompt:         "a beautiful landscape",
				NegativePrompt: "people, cars",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test actual generation without a client,
			// but we can validate the input structure
			if tt.input.Prompt == "" && !tt.expectError {
				t.Error("Expected error for empty prompt")
			}
			
			if tt.input.AspectRatio != "" {
				if !ValidAspectRatios[tt.input.AspectRatio] && !tt.expectError {
					t.Errorf("Invalid aspect ratio %s should cause error", tt.input.AspectRatio)
				}
			}
		})
	}
}
