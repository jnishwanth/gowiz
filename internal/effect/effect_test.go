package effect

import (
	"context"
	"testing"
	"time"

	"wiz-tui/internal/wiz"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		input   string
		wantR   int
		wantG   int
		wantB   int
		wantErr bool
	}{
		{"", 255, 0, 0, false},
		{"red", 255, 0, 0, false},
		{"GREEN", 0, 255, 0, false},
		{"blue", 0, 0, 255, false},
		{"yellow", 255, 255, 0, false},
		{"#FF0000", 255, 0, 0, false},
		{"#00FF00", 0, 255, 0, false},
		{"0000FF", 0, 0, 255, false},
		{"#F00", 255, 0, 0, false},
		{"invalidcolor", 0, 0, 0, true},
		{"#GHIJKL", 0, 0, 0, true},
	}

	for _, tt := range tests {
		r, g, b, err := ParseColor(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if r != tt.wantR || g != tt.wantG || b != tt.wantB {
				t.Errorf("ParseColor(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.input, r, g, b, tt.wantR, tt.wantG, tt.wantB)
			}
		}
	}
}

func TestGenerateFlashAndStrobe(t *testing.T) {
	steps, err := GenerateFlash("red", 2, 50)
	if err != nil {
		t.Fatalf("GenerateFlash failed: %v", err)
	}
	if len(steps) != 4 {
		t.Errorf("expected 4 steps for count 2, got %d", len(steps))
	}
	if steps[0].Params.R == nil || *steps[0].Params.R != 255 {
		t.Errorf("expected step 0 red component 255, got %v", steps[0].Params.R)
	}

	strobeSteps, err := GenerateStrobe("blue", 3, 20)
	if err != nil {
		t.Fatalf("GenerateStrobe failed: %v", err)
	}
	if len(strobeSteps) != 6 {
		t.Errorf("expected 6 steps for strobe count 3, got %d", len(strobeSteps))
	}
}

func TestGeneratePulseAndRainbow(t *testing.T) {
	pulseSteps := GeneratePulse(20, 80, 2, 30)
	if len(pulseSteps) != 4 {
		t.Errorf("expected 4 pulse steps, got %d", len(pulseSteps))
	}
	if pulseSteps[0].Params.Dimming == nil || *pulseSteps[0].Params.Dimming != 80 {
		t.Errorf("expected max dimming 80, got %v", pulseSteps[0].Params.Dimming)
	}

	rainbowSteps := GenerateRainbow(6, 40)
	if len(rainbowSteps) != 6 {
		t.Errorf("expected 6 rainbow steps, got %d", len(rainbowSteps))
	}
	for i, s := range rainbowSteps {
		if s.Params.R == nil || s.Params.G == nil || s.Params.B == nil {
			t.Errorf("rainbow step %d missing RGB parameters", i)
		}
	}
}

func TestGenerateStepsInvalidType(t *testing.T) {
	_, err := GenerateSteps(EffectConfig{Type: EffectType("unknown")})
	if err == nil {
		t.Error("expected error for unknown effect type, got nil")
	}
}

func TestExecuteEffect(t *testing.T) {
	client := wiz.NewMockClient()
	targetIPs := []string{"192.168.1.50"}

	cfg := EffectConfig{
		Type:       EffectFlash,
		Color:      "red",
		Count:      1,
		IntervalMs: 5,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errs := Execute(ctx, client, targetIPs, cfg)
	for ip, err := range errs {
		if err != nil {
			t.Errorf("Execute error for %s: %v", ip, err)
		}
	}
}
