package tools

import (
	"testing"
)

func TestNewGoogleMapsTool(t *testing.T) {
	mapsTool, err := NewGoogleMapsTool()
	if err != nil {
		t.Fatalf("NewGoogleMapsTool returned error: %v", err)
	}

	if mapsTool.Name() != "generate_google_maps" {
		t.Errorf("Expected name 'generate_google_maps', got '%s'", mapsTool.Name())
	}

	if mapsTool.IsLongRunning() {
		t.Error("Expected IsLongRunning to be false")
	}
}

func TestGenerateGoogleMapsURL_SingleLocation(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []Location{
			{Name: "东京塔", Address: "东京都港区"},
		},
		MapTitle: "东京景点",
	}

	output := generateGoogleMapsURL(input)
	if !output.Success {
		t.Fatalf("Expected success, got error: %s", output.Error)
	}

	if output.MapURL == "" {
		t.Fatal("Expected map URL, got empty string")
	}

	t.Logf("Generated map URL: %s", output.MapURL)
	t.Logf("Message: %s", output.Message)
}

func TestGenerateGoogleMapsURL_MultipleLocations(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []Location{
			{Name: "东京塔", Address: "东京都港区"},
			{Name: "浅草寺", Address: "东京都台东区"},
			{Name: "新宿御苑", Address: "东京都新宿区"},
		},
		MapTitle: "东京一日游",
	}

	output := generateGoogleMapsURL(input)
	if !output.Success {
		t.Fatalf("Expected success, got error: %s", output.Error)
	}

	if output.MapURL == "" {
		t.Fatal("Expected map URL, got empty string")
	}

	t.Logf("Generated map URL: %s", output.MapURL)
	t.Logf("Message: %s", output.Message)
}

func TestGenerateGoogleMapsURL_EmptyLocations(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []Location{},
	}

	output := generateGoogleMapsURL(input)
	if output.Success {
		t.Fatal("Expected failure for empty locations")
	}

	if output.Error == "" {
		t.Fatal("Expected error message for empty locations")
	}

	t.Logf("Expected error: %s", output.Error)
}

func TestGenerateGoogleMapsURL_TwoLocations(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []Location{
			{Name: "成田国际机场", Address: "千叶县成田市"},
			{Name: "东京站", Address: "东京都千代田区"},
		},
	}

	output := generateGoogleMapsURL(input)
	if !output.Success {
		t.Fatalf("Expected success, got error: %s", output.Error)
	}

	if output.MapURL == "" {
		t.Fatal("Expected map URL, got empty string")
	}

	t.Logf("Generated map URL: %s", output.MapURL)
}

func TestBuildLocationQuery(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		expected string
	}{
		{
			name:     "Name and Address",
			location: Location{Name: "东京塔", Address: "东京都港区"},
			expected: "东京塔, 东京都港区",
		},
		{
			name:     "Address only",
			location: Location{Address: "东京都新宿区"},
			expected: "东京都新宿区",
		},
		{
			name:     "Name only",
			location: Location{Name: "浅草寺"},
			expected: "浅草寺",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLocationQuery(tt.location)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
