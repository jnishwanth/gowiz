package effect

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"wiz-tui/internal/wiz"
)

// EffectType specifies the category of dynamic lighting effect.
type EffectType string

const (
	EffectFlash   EffectType = "flash"
	EffectPulse   EffectType = "pulse"
	EffectStrobe  EffectType = "strobe"
	EffectRainbow EffectType = "rainbow"
)

// EffectConfig defines configuration for generating and executing dynamic lighting effects.
type EffectConfig struct {
	Type        EffectType `json:"type"`
	Color       string     `json:"color,omitempty"`
	Count       int        `json:"count,omitempty"`
	IntervalMs  int        `json:"intervalMs,omitempty"`
	MinDimming  int        `json:"minDimming,omitempty"`
	MaxDimming  int        `json:"maxDimming,omitempty"`
	DurationSec int        `json:"durationSec,omitempty"`
}

// Step represents a single lighting state parameter mutation in an effect timeline.
type Step struct {
	Params  wiz.PilotParams `json:"params"`
	DelayMs int             `json:"delayMs"`
}

// NamedColors maps standard color names to RGB values.
var NamedColors = map[string][3]int{
	"red":     {255, 0, 0},
	"green":   {0, 255, 0},
	"blue":    {0, 0, 255},
	"yellow":  {255, 255, 0},
	"cyan":    {0, 255, 255},
	"magenta": {255, 0, 255},
	"purple":  {128, 0, 128},
	"orange":  {255, 165, 0},
	"pink":    {255, 105, 180},
	"white":   {255, 255, 255},
	"alert":   {255, 0, 0},
}

// ParseColor converts a hex color string (#RGB, #RRGGBB, RRGGBB) or color name into RGB components.
func ParseColor(colorStr string) (r, g, b int, err error) {
	colorStr = strings.TrimSpace(strings.ToLower(colorStr))
	if colorStr == "" {
		return 255, 0, 0, nil // Default red
	}

	if rgb, found := NamedColors[colorStr]; found {
		return rgb[0], rgb[1], rgb[2], nil
	}

	hex := strings.TrimPrefix(colorStr, "#")
	if len(hex) == 3 {
		rVal, e1 := strconv.ParseUint(string([]byte{hex[0], hex[0]}), 16, 8)
		gVal, e2 := strconv.ParseUint(string([]byte{hex[1], hex[1]}), 16, 8)
		bVal, e3 := strconv.ParseUint(string([]byte{hex[2], hex[2]}), 16, 8)
		if e1 == nil && e2 == nil && e3 == nil {
			return int(rVal), int(gVal), int(bVal), nil
		}
	} else if len(hex) == 6 {
		rVal, e1 := strconv.ParseUint(hex[0:2], 16, 8)
		gVal, e2 := strconv.ParseUint(hex[2:4], 16, 8)
		bVal, e3 := strconv.ParseUint(hex[4:6], 16, 8)
		if e1 == nil && e2 == nil && e3 == nil {
			return int(rVal), int(gVal), int(bVal), nil
		}
	}

	return 0, 0, 0, fmt.Errorf("invalid color format or name: %q", colorStr)
}

// GenerateFlash builds steps for flashing a target color on and off.
func GenerateFlash(colorStr string, count, intervalMs int) ([]Step, error) {
	r, g, b, err := ParseColor(colorStr)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 3
	}
	if intervalMs <= 0 {
		intervalMs = 400
	}

	steps := make([]Step, 0, count*2)
	onParams := wiz.NewRGBParams(r, g, b)
	offParams := wiz.NewPowerParams(false)

	for i := 0; i < count; i++ {
		steps = append(steps, Step{Params: onParams, DelayMs: intervalMs})
		steps = append(steps, Step{Params: offParams, DelayMs: intervalMs})
	}
	return steps, nil
}

// GenerateStrobe builds high-frequency flash steps for intense alert lighting.
func GenerateStrobe(colorStr string, count, intervalMs int) ([]Step, error) {
	if count <= 0 {
		count = 6
	}
	if intervalMs <= 0 {
		intervalMs = 150
	}
	return GenerateFlash(colorStr, count, intervalMs)
}

// GeneratePulse builds dimming pulse steps oscillating between min and max brightness.
func GeneratePulse(minDim, maxDim, count, intervalMs int) []Step {
	if count <= 0 {
		count = 3
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	minDim = wiz.Clamp(minDim, 10, 100)
	maxDim = wiz.Clamp(maxDim, 10, 100)
	if minDim >= maxDim {
		minDim = 10
		maxDim = 100
	}

	steps := make([]Step, 0, count*2)
	highParams := wiz.NewDimmingParams(maxDim)
	lowParams := wiz.NewDimmingParams(minDim)

	for i := 0; i < count; i++ {
		steps = append(steps, Step{Params: highParams, DelayMs: intervalMs})
		steps = append(steps, Step{Params: lowParams, DelayMs: intervalMs})
	}
	return steps
}

// GenerateRainbow builds a spectrum sweep across RGB hues in N steps.
func GenerateRainbow(stepsCount, intervalMs int) []Step {
	if stepsCount <= 0 {
		stepsCount = 6
	}
	if intervalMs <= 0 {
		intervalMs = 600
	}

	steps := make([]Step, 0, stepsCount)
	for i := 0; i < stepsCount; i++ {
		hue := (float64(i) / float64(stepsCount)) * 360.0
		r, g, b := hslToRGB(hue, 1.0, 0.5)
		steps = append(steps, Step{
			Params:  wiz.NewRGBParams(r, g, b),
			DelayMs: intervalMs,
		})
	}
	return steps
}

// GenerateSteps constructs effect timeline steps based on an EffectConfig.
func GenerateSteps(cfg EffectConfig) ([]Step, error) {
	switch cfg.Type {
	case EffectFlash:
		return GenerateFlash(cfg.Color, cfg.Count, cfg.IntervalMs)
	case EffectStrobe:
		return GenerateStrobe(cfg.Color, cfg.Count, cfg.IntervalMs)
	case EffectPulse:
		minDim := cfg.MinDimming
		if minDim <= 0 {
			minDim = 10
		}
		maxDim := cfg.MaxDimming
		if maxDim <= 0 {
			maxDim = 100
		}
		return GeneratePulse(minDim, maxDim, cfg.Count, cfg.IntervalMs), nil
	case EffectRainbow:
		return GenerateRainbow(cfg.Count, cfg.IntervalMs), nil
	default:
		return nil, fmt.Errorf("unsupported effect type: %q", cfg.Type)
	}
}

// Execute Effect executes an effect timeline across target IP addresses.
func Execute(ctx context.Context, client wiz.Client, targetIPs []string, cfg EffectConfig) map[string]error {
	steps, err := GenerateSteps(cfg)
	if err != nil {
		errMap := make(map[string]error)
		for _, ip := range targetIPs {
			errMap[ip] = err
		}
		return errMap
	}

	lastErrs := make(map[string]error)
	for _, step := range steps {
		select {
		case <-ctx.Done():
			return lastErrs
		default:
		}

		errs := client.SendBatchCommand(ctx, targetIPs, step.Params)
		for ip, e := range errs {
			if e != nil {
				lastErrs[ip] = e
			}
		}

		if step.DelayMs > 0 {
			select {
			case <-ctx.Done():
				return lastErrs
			case <-time.After(time.Duration(step.DelayMs) * time.Millisecond):
			}
		}
	}
	return lastErrs
}

// hslToRGB converts Hue (0-360), Saturation (0-1), Lightness (0-1) to integer R, G, B (0-255).
func hslToRGB(h, s, l float64) (r, g, b int) {
	c := (1.0 - math.Abs(2.0*l-1.0)) * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := l - c/2.0

	var rF, gF, bF float64
	switch {
	case 0 <= h && h < 60:
		rF, gF, bF = c, x, 0
	case 60 <= h && h < 120:
		rF, gF, bF = x, c, 0
	case 120 <= h && h < 180:
		rF, gF, bF = 0, c, x
	case 180 <= h && h < 240:
		rF, gF, bF = 0, x, c
	case 240 <= h && h < 300:
		rF, gF, bF = x, 0, c
	default:
		rF, gF, bF = c, 0, x
	}

	r = int(math.Round((rF + m) * 255.0))
	g = int(math.Round((gF + m) * 255.0))
	b = int(math.Round((bF + m) * 255.0))
	return wiz.Clamp(r, 0, 255), wiz.Clamp(g, 0, 255), wiz.Clamp(b, 0, 255)
}
