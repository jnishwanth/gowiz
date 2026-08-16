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

func TestValidatePhases(t *testing.T) {
	t.Run("Empty phases list", func(t *testing.T) {
		if err := circadian.ValidatePhases(nil); err == nil {
			t.Errorf("expected error for empty phases list, got nil")
		}
	})

	t.Run("Empty phase name", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "", StartHour: 0, EndHour: 12, StartTemp: 2700, EndTemp: 5000, StartDimming: 50, EndDimming: 100},
		}
		if err := circadian.ValidatePhases(phases); err == nil {
			t.Errorf("expected error for empty phase name, got nil")
		}
	})

	t.Run("Invalid start hour", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "Bad Start", StartHour: -1, EndHour: 12, StartTemp: 2700, EndTemp: 5000, StartDimming: 50, EndDimming: 100},
		}
		if err := circadian.ValidatePhases(phases); err == nil {
			t.Errorf("expected error for negative start hour, got nil")
		}
	})

	t.Run("Invalid end hour", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "Bad End", StartHour: 10, EndHour: 8, StartTemp: 2700, EndTemp: 5000, StartDimming: 50, EndDimming: 100},
		}
		if err := circadian.ValidatePhases(phases); err == nil {
			t.Errorf("expected error when end hour <= start hour, got nil")
		}
	})

	t.Run("Invalid color temperature", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "Bad Temp", StartHour: 0, EndHour: 12, StartTemp: 1500, EndTemp: 5000, StartDimming: 50, EndDimming: 100},
		}
		if err := circadian.ValidatePhases(phases); err == nil {
			t.Errorf("expected error for out-of-range color temp, got nil")
		}
	})

	t.Run("Invalid dimming percentage", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "Bad Dimming", StartHour: 0, EndHour: 12, StartTemp: 2700, EndTemp: 5000, StartDimming: 5, EndDimming: 100},
		}
		if err := circadian.ValidatePhases(phases); err == nil {
			t.Errorf("expected error for out-of-range dimming, got nil")
		}
	})

	t.Run("Valid phases slice", func(t *testing.T) {
		phases := []circadian.SchedulePhase{
			{Name: "Custom Morning", StartHour: 5, EndHour: 12, StartTemp: 2200, EndTemp: 6000, StartDimming: 20, EndDimming: 100},
			{Name: "Custom Evening", StartHour: 12, EndHour: 24, StartTemp: 6000, EndTemp: 2200, StartDimming: 100, EndDimming: 20},
		}
		if err := circadian.ValidatePhases(phases); err != nil {
			t.Errorf("unexpected error for valid phases: %v", err)
		}
	})
}

func TestCalculateWithPhases(t *testing.T) {
	customPhases := []circadian.SchedulePhase{
		{Name: "Night Owl Rest", StartHour: 0, EndHour: 8, StartTemp: 2200, EndTemp: 2200, StartDimming: 20, EndDimming: 20},
		{Name: "Shift Work Boost", StartHour: 8, EndHour: 16, StartTemp: 3000, EndTemp: 6500, StartDimming: 50, EndDimming: 100},
		{Name: "Wind Down", StartHour: 16, EndHour: 24, StartTemp: 6500, EndTemp: 2200, StartDimming: 100, EndDimming: 20},
	}

	t.Run("Fallback to default on nil phases", func(t *testing.T) {
		tm := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
		info := circadian.CalculateWithPhases(tm, nil)
		if info.Phase != "Day Daylight" {
			t.Errorf("expected fallback to default phase 'Day Daylight', got %q", info.Phase)
		}
	})

	t.Run("Custom phase calculation", func(t *testing.T) {
		tm := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
		info := circadian.CalculateWithPhases(tm, customPhases)
		if info.Phase != "Shift Work Boost" {
			t.Errorf("expected custom phase 'Shift Work Boost', got %q", info.Phase)
		}
		// At hour 12 (midpoint of 8..16), temp should be (3000+6500)/2 = 4750
		if info.Temp != 4750 {
			t.Errorf("expected temp 4750, got %d", info.Temp)
		}
		// Dimming at midpoint (50+100)/2 = 75
		if info.Dimming != 75 {
			t.Errorf("expected dimming 75, got %d", info.Dimming)
		}
	})
}
