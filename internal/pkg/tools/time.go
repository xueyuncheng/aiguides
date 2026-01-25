// Copyright 2025 AIGuides
// Time tool for providing current date and time information to agents

package tools

import (
	"fmt"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// CurrentTimeInput defines the input for the current_time tool
type CurrentTimeInput struct {
	Timezone string `json:"timezone,omitempty" jsonschema:"Optional timezone (e.g., 'America/New_York', 'Asia/Shanghai'). Defaults to UTC if not specified."`
}

// CurrentTimeOutput defines the output of the current_time tool
type CurrentTimeOutput struct {
	DateTime     string `json:"datetime"`      // Full datetime string (e.g., "2026-01-25 15:30:45")
	Date         string `json:"date"`          // Date only (e.g., "2026-01-25")
	Time         string `json:"time"`          // Time only (e.g., "15:30:45")
	Year         int    `json:"year"`          // Year (e.g., 2026)
	Month        string `json:"month"`         // Month name (e.g., "January")
	MonthNumber  int    `json:"month_number"`  // Month number (1-12)
	Day          int    `json:"day"`           // Day of month (1-31)
	Weekday      string `json:"weekday"`       // Day of week (e.g., "Saturday")
	Hour         int    `json:"hour"`          // Hour (0-23)
	Minute       int    `json:"minute"`        // Minute (0-59)
	Second       int    `json:"second"`        // Second (0-59)
	Timezone     string `json:"timezone"`      // Timezone (e.g., "UTC")
	TimezoneAbbr string `json:"timezone_abbr"` // Timezone abbreviation (e.g., "UTC", "EST")
	UnixTime     int64  `json:"unix_time"`     // Unix timestamp
}

// NewCurrentTimeTool creates a tool that provides current date and time information
func NewCurrentTimeTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name: "current_time",
		Description: `Get the current date and time. Use this tool when you need to:
- Know the current date/time for context
- Determine if information might be outdated
- Format search queries with current dates
- Provide time-aware responses

This tool gives you the exact current datetime so you can make informed decisions about whether to use web search for time-sensitive queries.`,
	}

	handler := func(ctx tool.Context, input CurrentTimeInput) (*CurrentTimeOutput, error) {
		// Get current time
		now := time.Now()

		// Apply timezone if specified
		if input.Timezone != "" {
			loc, err := time.LoadLocation(input.Timezone)
			if err != nil {
				return nil, fmt.Errorf("invalid timezone '%s': %w", input.Timezone, err)
			}
			now = now.In(loc)
		} else {
			// Default to UTC
			now = now.UTC()
		}

		// Get timezone info
		zone, offset := now.Zone()
		_ = offset // offset in seconds, not used for now but available

		return &CurrentTimeOutput{
			DateTime:     now.Format("2006-01-02 15:04:05"),
			Date:         now.Format("2006-01-02"),
			Time:         now.Format("15:04:05"),
			Year:         now.Year(),
			Month:        now.Month().String(),
			MonthNumber:  int(now.Month()),
			Day:          now.Day(),
			Weekday:      now.Weekday().String(),
			Hour:         now.Hour(),
			Minute:       now.Minute(),
			Second:       now.Second(),
			Timezone:     input.Timezone,
			TimezoneAbbr: zone,
			UnixTime:     now.Unix(),
		}, nil
	}

	return functiontool.New(config, handler)
}
