package wiz

import "strings"

type Scene struct {
	ID          int
	Name        string
	Category    string
	Description string
	AccentColor string   // Primary hex color string for UI rendering
	Palette     []string // Animated color sequence mimicking physical bulb output
}

func (s Scene) GetAccentColor(frame int) string {
	if len(s.Palette) > 0 {
		return s.Palette[frame%len(s.Palette)]
	}
	return s.AccentColor
}

var AllScenes = []Scene{
	{ID: 1, Name: "Ocean", Category: "Nature", Description: "Refreshing blue waves", AccentColor: "#0077be", Palette: []string{"#0077be", "#00bfff", "#004080", "#20b2aa"}},
	{ID: 2, Name: "Romance", Category: "Cozy", Description: "Intimate magenta & purple ambiance", AccentColor: "#e91e63", Palette: []string{"#e91e63", "#9c27b0", "#ff4081", "#7b1fa2"}},
	{ID: 3, Name: "Sunset", Category: "Nature", Description: "Warm golden glow", AccentColor: "#ff4500", Palette: []string{"#ff4500", "#ff8c00", "#ff2400", "#d84315"}},
	{ID: 4, Name: "Party", Category: "Dynamic", Description: "Vibrant shifting colors", AccentColor: "#ff007f", Palette: []string{"#ff007f", "#00e5ff", "#d500f9", "#76ff03"}},
	{ID: 5, Name: "Fireplace", Category: "Cozy", Description: "Flickering flame warmth", AccentColor: "#ff6600", Palette: []string{"#ff6600", "#ff3300", "#ffaa00", "#d84315"}},
	{ID: 6, Name: "Cozy", Category: "Cozy", Description: "Soft warm ambiance", AccentColor: "#ffaa33", Palette: []string{"#ffaa33", "#ff8800", "#ffcc66", "#e65100"}},
	{ID: 7, Name: "Forest", Category: "Nature", Description: "Calming woodland green", AccentColor: "#228b22", Palette: []string{"#228b22", "#00e676", "#1b5e20", "#76ff03"}},
	{ID: 8, Name: "Pastel colors", Category: "Dynamic", Description: "Gentle pastel gradients", AccentColor: "#b19cd9", Palette: []string{"#b19cd9", "#ffb3ba", "#baffc9", "#bae1ff"}},
	{ID: 9, Name: "Wake-up", Category: "Rhythm", Description: "Gradual morning sunlight", AccentColor: "#ffeb3b", Palette: []string{"#ff8f00", "#ffc107", "#ffeb3b", "#ffffff"}},
	{ID: 10, Name: "Bedtime", Category: "Rhythm", Description: "Dimming evening tones", AccentColor: "#4a148c", Palette: []string{"#ff8f00", "#e65100", "#4a148c", "#1a237e"}},
	{ID: 11, Name: "Warm white", Category: "White", Description: "2700K classic warmth", AccentColor: "#ffcc66"},
	{ID: 12, Name: "Daylight", Category: "White", Description: "6500K bright focus white", AccentColor: "#e0f7fa"},
	{ID: 13, Name: "Cool white", Category: "White", Description: "4000K balanced crisp white", AccentColor: "#ffffff"},
	{ID: 14, Name: "Night light", Category: "Cozy", Description: "Subtle soft night glow", AccentColor: "#3f51b5"},
	{ID: 15, Name: "Focus", Category: "White", Description: "High productivity lighting", AccentColor: "#81d4fa"},
	{ID: 16, Name: "Relax", Category: "White", Description: "Soothing neutral light", AccentColor: "#ffe0b2"},
	{ID: 17, Name: "True colors", Category: "White", Description: "High CRI natural rendering", AccentColor: "#ffffff"},
	{ID: 18, Name: "TV time", Category: "Cozy", Description: "Ambient home cinema backdrop", AccentColor: "#1a237e", Palette: []string{"#1a237e", "#283593", "#0d47a1", "#311b92"}},
	{ID: 19, Name: "Plant growth", Category: "Special", Description: "Optimal spectrum for flora", AccentColor: "#e91e63", Palette: []string{"#e91e63", "#ab47bc", "#8e24aa", "#f48fb1"}},
	{ID: 20, Name: "Spring", Category: "Seasons", Description: "Lush blooming hues", AccentColor: "#76ff03", Palette: []string{"#76ff03", "#ff4081", "#a7ffeb", "#ffff00"}},
	{ID: 21, Name: "Summer", Category: "Seasons", Description: "Bright sun-drenched radiance", AccentColor: "#ffc107", Palette: []string{"#ffc107", "#ff9800", "#ff5722", "#ffeb3b"}},
	{ID: 22, Name: "Fall", Category: "Seasons", Description: "Crisp autumn foliage amber", AccentColor: "#d84315", Palette: []string{"#d84315", "#ef6c00", "#f57f17", "#bf360c"}},
	{ID: 23, Name: "Deep dive", Category: "Nature", Description: "Mystic deep ocean navy", AccentColor: "#0d47a1", Palette: []string{"#0d47a1", "#01579b", "#006064", "#1a237e"}},
	{ID: 24, Name: "Jungle", Category: "Nature", Description: "Tropical rainforest vibrant green", AccentColor: "#00e676", Palette: []string{"#00e676", "#76ff03", "#1b5e20", "#00c853"}},
	{ID: 25, Name: "Mojito", Category: "Dynamic", Description: "Zesty lime & mint sparkle", AccentColor: "#b2ff59", Palette: []string{"#b2ff59", "#76ff03", "#eeff41", "#64dd17"}},
	{ID: 26, Name: "Club", Category: "Dynamic", Description: "Pulsing neon party rhythm", AccentColor: "#d500f9", Palette: []string{"#d500f9", "#ff007f", "#00e5ff", "#aa00ff"}},
	{ID: 27, Name: "Christmas", Category: "Festive", Description: "Red and green holiday spirit", AccentColor: "#ff1744", Palette: []string{"#ff1744", "#00e676", "#d50000", "#2e7d32"}},
	{ID: 28, Name: "Halloween", Category: "Festive", Description: "Spooky orange & purple glow", AccentColor: "#ff6d00", Palette: []string{"#ff6d00", "#aa00ff", "#dd2c00", "#6200ea"}},
	{ID: 29, Name: "Candlelight", Category: "Cozy", Description: "Intimate flickering candle effect", AccentColor: "#ff8f00", Palette: []string{"#ff8f00", "#ff6f00", "#ffab00", "#e65100"}},
	{ID: 30, Name: "Golden white", Category: "White", Description: "Rich amber incandescent white", AccentColor: "#ffb300"},
	{ID: 31, Name: "Pulse", Category: "Dynamic", Description: "Rhythmic breathing pulse", AccentColor: "#00e5ff", Palette: []string{"#00e5ff", "#00b0ff", "#2979ff", "#651fff"}},
	{ID: 32, Name: "Steampunk", Category: "Special", Description: "Copper & brass vintage warmth", AccentColor: "#bf360c", Palette: []string{"#bf360c", "#d84315", "#ff8f00", "#795548"}},
	{ID: 33, Name: "Diwali", Category: "Festive", Description: "Festive golden lights", AccentColor: "#ffd700", Palette: []string{"#ffd700", "#ff9800", "#ff3d00", "#ffc107"}},
	{ID: 1000, Name: "Rhythm", Category: "Rhythm", Description: "Circadian sync spectrum", AccentColor: "#00b0ff", Palette: []string{"#00b0ff", "#ffeb3b", "#ff9800", "#3f51b5"}},
}

// GetSceneByID returns the scene with matching ID or default
func GetSceneByID(id int) Scene {
	for _, s := range AllScenes {
		if s.ID == id {
			return s
		}
	}
	return AllScenes[0]
}

// GetSceneByName performs case-insensitive name matching
func GetSceneByName(name string) (Scene, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, s := range AllScenes {
		if strings.ToLower(s.Name) == lower {
			return s, true
		}
	}
	// Prefix match support
	for _, s := range AllScenes {
		if strings.HasPrefix(strings.ToLower(s.Name), lower) {
			return s, true
		}
	}
	return Scene{}, false
}

// FilterScenes returns scenes matching a query string
func FilterScenes(query string) []Scene {
	if strings.TrimSpace(query) == "" {
		return AllScenes
	}
	q := strings.ToLower(query)
	var matched []Scene
	for _, s := range AllScenes {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Category), q) ||
			strings.Contains(strings.ToLower(s.Description), q) {
			matched = append(matched, s)
		}
	}
	return matched
}
