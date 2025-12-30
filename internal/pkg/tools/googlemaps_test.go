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
		Locations: []string{
			"东京塔, 东京都港区",
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
	t.Logf("Message: %s", output.Message)
}

func TestGenerateGoogleMapsURL_MultipleLocations(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []string{
			"东京塔, 东京都港区",
			"浅草寺, 东京都台东区",
			"新宿御苑, 东京都新宿区",
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
	t.Logf("Message: %s", output.Message)
}

func TestGenerateGoogleMapsURL_EmptyLocations(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []string{},
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
		Locations: []string{
			"成田国际机场, 千叶县成田市",
			"东京站, 东京都千代田区",
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

func TestGenerateGoogleMapsURL_EmptyLocationInfo(t *testing.T) {
	input := GoogleMapsInput{
		Locations: []string{
			"",
		},
	}

	output := generateGoogleMapsURL(input)
	if output.Success {
		t.Fatal("Expected failure for empty location information")
	}

	if output.Error == "" {
		t.Fatal("Expected error message for empty location information")
	}

	t.Logf("Expected error: %s", output.Error)
}
