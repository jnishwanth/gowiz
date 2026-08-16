package circadian_test

import (
	"testing"
	"time"

	"wiz-tui/internal/circadian"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name          string
		hour          int
		minute        int
		expectedPhase string
		minTemp       int
		maxTemp       int
		minDimming    int
		maxDimming    int
	}{
		{
			name:          "Night Rest - Midnight",
			hour:          2,
			minute:        0,
			expectedPhase: "Night Rest",
			minTemp:       2200,
			maxTemp:       2200,
			minDimming:    30,
			maxDimming:    30,
		},
		{
			name:          "Morning Sunrise - 8:00 AM",
			hour:          8,
			minute:        0,
			expectedPhase: "Morning Sunrise",
			minTemp:       4000,
			maxTemp:       4600,
			minDimming:    60,
			maxDimming:    70,
		},
		{
			name:          "Day Daylight - 12:00 PM",
			hour:          12,
			minute:        0,
			expectedPhase: "Day Daylight",
			minTemp:       5500,
			maxTemp:       6500,
			minDimming:    100,
			maxDimming:    100,
		},
		{
			name:          "Evening Sunset - 19:00 PM",
			hour:          19,
			minute:        0,
			expectedPhase: "Evening Sunset",
			minTemp:       3000,
			maxTemp:       4200,
			minDimming:    70,
			maxDimming:    80,
		},
		{
			name:          "Late Wind-down - 22:00 PM",
			hour:          22,
			minute:        0,
			expectedPhase: "Late Wind-down",
			minTemp:       2200,
			maxTemp:       2700,
			minDimming:    30,
			maxDimming:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := time.Date(2026, 8, 17, tt.hour, tt.minute, 0, 0, time.UTC)
			info := circadian.Calculate(tm)

			if info.Phase != tt.expectedPhase {
				t.Errorf("expected phase %q, got %q", tt.expectedPhase, info.Phase)
			}
			if info.Temp < tt.minTemp || info.Temp > tt.maxTemp {
				t.Errorf("expected temp between %d and %d, got %d", tt.minTemp, tt.maxTemp, info.Temp)
			}
			if info.Dimming < tt.minDimming || info.Dimming > tt.maxDimming {
				t.Errorf("expected dimming between %d and %d, got %d", tt.minDimming, tt.maxDimming, info.Dimming)
			}
			if info.Params.Temp == nil || *info.Params.Temp != info.Temp {
				t.Errorf("expected PilotParams Temp to match info.Temp %d", info.Temp)
			}
			if info.Params.Dimming == nil || *info.Params.Dimming != info.Dimming {
				t.Errorf("expected PilotParams Dimming to match info.Dimming %d", info.Dimming)
			}
			if info.Status == "" {
				t.Errorf("expected non-empty Status string")
			}
		})
	}
}

func TestParseTimeArg(t *testing.T) {
	t.Run("Valid HH:MM", func(t *testing.T) {
		tm, err := circadian.ParseTimeArg("14:30")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tm.Hour() != 14 || tm.Minute() != 30 {
			t.Errorf("expected 14:30, got %02d:%02d", tm.Hour(), tm.Minute())
		}
	})

	t.Run("Valid HH", func(t *testing.T) {
		tm, err := circadian.ParseTimeArg("9")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tm.Hour() != 9 || tm.Minute() != 0 {
			t.Errorf("expected 09:00, got %02d:%02d", tm.Hour(), tm.Minute())
		}
	})

	t.Run("Invalid hour range", func(t *testing.T) {
		_, err := circadian.ParseTimeArg("25:00")
		if err == nil {
			t.Errorf("expected error for 25:00, got nil")
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		_, err := circadian.ParseTimeArg("invalid")
		if err == nil {
			t.Errorf("expected error for invalid format, got nil")
		}
	})
}
