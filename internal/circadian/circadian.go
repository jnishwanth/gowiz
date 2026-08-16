package circadian

import (
	"fmt"
	"time"

	"wiz-tui/internal/wiz"
)

// SchedulePhase represents a distinct phase of the 24-hour circadian rhythm.
type SchedulePhase struct {
	Name         string `json:"name"`
	StartHour    int    `json:"startHour"`
	EndHour      int    `json:"endHour"`
	StartTemp    int    `json:"startTemp"`    // Color temp in Kelvin (2200 - 6500)
	EndTemp      int    `json:"endTemp"`      // Color temp in Kelvin (2200 - 6500)
	StartDimming int    `json:"startDimming"` // Dimming percentage (10 - 100)
	EndDimming   int    `json:"endDimming"`   // Dimming percentage (10 - 100)
}

// DefaultPhases defines the standard 24-hour circadian lighting curve.
var DefaultPhases = []SchedulePhase{
	{Name: "Night Rest", StartHour: 0, EndHour: 6, StartTemp: 2200, EndTemp: 2200, StartDimming: 30, EndDimming: 30},
	{Name: "Morning Sunrise", StartHour: 6, EndHour: 10, StartTemp: 2200, EndTemp: 6500, StartDimming: 30, EndDimming: 100},
	{Name: "Day Daylight", StartHour: 10, EndHour: 17, StartTemp: 6500, EndTemp: 5000, StartDimming: 100, EndDimming: 100},
	{Name: "Evening Sunset", StartHour: 17, EndHour: 21, StartTemp: 5000, EndTemp: 2700, StartDimming: 100, EndDimming: 50},
	{Name: "Late Wind-down", StartHour: 21, EndHour: 24, StartTemp: 2700, EndTemp: 2200, StartDimming: 50, EndDimming: 30},
}

// Info holds calculated circadian parameters and descriptive phase information.
type Info struct {
	Phase   string          `json:"phase"`
	Temp    int             `json:"temp"`
	Dimming int             `json:"dimming"`
	Status  string          `json:"status"`
	Params  wiz.PilotParams `json:"params"`
}

// Calculate returns the calculated Info (PilotParams, phase, temp, dimming) for a given time.Time.
func Calculate(t time.Time) Info {
	hour := t.Hour()
	minute := t.Minute()
	decimalTime := float64(hour) + float64(minute)/60.0

	for _, phase := range DefaultPhases {
		start := float64(phase.StartHour)
		end := float64(phase.EndHour)
		if decimalTime >= start && decimalTime < end {
			progress := (decimalTime - start) / (end - start)
			temp := int(float64(phase.StartTemp) + progress*float64(phase.EndTemp-phase.StartTemp))
			dimming := int(float64(phase.StartDimming) + progress*float64(phase.EndDimming-phase.StartDimming))

			temp = wiz.Clamp(temp, 2200, 6500)
			dimming = wiz.Clamp(dimming, 10, 100)

			params := wiz.NewTempParams(temp)
			params.Dimming = &dimming

			status := fmt.Sprintf("%s (%dK, %d%%)", phase.Name, temp, dimming)
			return Info{
				Phase:   phase.Name,
				Temp:    temp,
				Dimming: dimming,
				Status:  status,
				Params:  params,
			}
		}
	}

	params := wiz.NewTempParams(2200)
	dim := 30
	params.Dimming = &dim
	return Info{
		Phase:   "Night Rest",
		Temp:    2200,
		Dimming: 30,
		Status:  "Night Rest (2200K, 30%)",
		Params:  params,
	}
}

// ParseTimeArg parses "HH:MM" or "HH" string into a time.Time on the current day.
func ParseTimeArg(arg string) (time.Time, error) {
	now := time.Now()
	var h, m int
	if n, err := fmt.Sscanf(arg, "%d:%d", &h, &m); err == nil && n == 2 {
		if h < 0 || h > 23 || m < 0 || m > 59 {
			return now, fmt.Errorf("invalid time range %q (expected 00:00 - 23:59)", arg)
		}
		return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location()), nil
	}
	if n, err := fmt.Sscanf(arg, "%d", &h); err == nil && n == 1 {
		if h < 0 || h > 23 {
			return now, fmt.Errorf("invalid hour %q (expected 0 - 23)", arg)
		}
		return time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, now.Location()), nil
	}
	return now, fmt.Errorf("invalid time format %q, expected HH:MM or HH", arg)
}
