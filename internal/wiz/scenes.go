package wiz

import "strings"

type Scene struct {
	ID          int
	Name        string
	Category    string
	Description string
	AccentColor string // Hex color string for UI rendering
}

var AllScenes = []Scene{
	{ID: 1, Name: "Ocean", Category: "Nature", Description: "Refreshing blue waves", AccentColor: "#0077be"},
	{ID: 2, Name: "Sunset", Category: "Nature", Description: "Warm golden glow", AccentColor: "#ff4500"},
	{ID: 3, Name: "Party", Category: "Dynamic", Description: "Vibrant shifting colors", AccentColor: "#ff007f"},
	{ID: 4, Name: "Fireplace", Category: "Cozy", Description: "Flickering flame warmth", AccentColor: "#ff6600"},
	{ID: 5, Name: "Cozy", Category: "Cozy", Description: "Soft warm ambiance", AccentColor: "#ffaa33"},
	{ID: 6, Name: "Forest", Category: "Nature", Description: "Calming woodland green", AccentColor: "#228b22"},
	{ID: 7, Name: "Pastel Colors", Category: "Dynamic", Description: "Gentle pastel gradients", AccentColor: "#b19cd9"},
	{ID: 8, Name: "Wake up", Category: "Rhythm", Description: "Gradual morning sunlight", AccentColor: "#ffeb3b"},
	{ID: 9, Name: "Bedtime", Category: "Rhythm", Description: "Dimming evening tones", AccentColor: "#4a148c"},
	{ID: 10, Name: "Warm White", Category: "White", Description: "2700K classic warmth", AccentColor: "#ffcc66"},
	{ID: 11, Name: "Daylight", Category: "White", Description: "6500K bright focus white", AccentColor: "#e0f7fa"},
	{ID: 12, Name: "Cool White", Category: "White", Description: "4000K balanced crisp white", AccentColor: "#ffffff"},
	{ID: 13, Name: "Night Light", Category: "Cozy", Description: "Subtle soft night glow", AccentColor: "#3f51b5"},
	{ID: 14, Name: "Focus", Category: "White", Description: "High productivity lighting", AccentColor: "#81d4fa"},
	{ID: 15, Name: "Relax", Category: "White", Description: "Soothing neutral light", AccentColor: "#ffe0b2"},
	{ID: 16, Name: "True colors", Category: "White", Description: "High CRI natural rendering", AccentColor: "#ffffff"},
	{ID: 17, Name: "TV time", Category: "Cozy", Description: "Ambient home cinema backdrop", AccentColor: "#1a237e"},
	{ID: 18, Name: "Plant growth", Category: "Special", Description: "Optimal spectrum for flora", AccentColor: "#e91e63"},
	{ID: 19, Name: "Spring", Category: "Seasons", Description: "Lush blooming hues", AccentColor: "#76ff03"},
	{ID: 20, Name: "Summer", Category: "Seasons", Description: "Bright sun-drenched radiance", AccentColor: "#ffc107"},
	{ID: 21, Name: "Fall", Category: "Seasons", Description: "Crisp autumn foliage amber", AccentColor: "#d84315"},
	{ID: 22, Name: "Deep dive", Category: "Nature", Description: "Mystic deep ocean navy", AccentColor: "#0d47a1"},
	{ID: 23, Name: "Jungle", Category: "Nature", Description: "Tropical rainforest vibrant green", AccentColor: "#00e676"},
	{ID: 24, Name: "Mojito", Category: "Dynamic", Description: "Zesty lime & mint sparkle", AccentColor: "#b2ff59"},
	{ID: 25, Name: "Club", Category: "Dynamic", Description: "Pulsing neon party rhythm", AccentColor: "#d500f9"},
	{ID: 26, Name: "Christmas", Category: "Festive", Description: "Red and green holiday spirit", AccentColor: "#ff1744"},
	{ID: 27, Name: "Halloween", Category: "Festive", Description: "Spooky orange & purple glow", AccentColor: "#aa00ff"},
	{ID: 28, Name: "Candlelight", Category: "Cozy", Description: "Intimate flickering candle effect", AccentColor: "#ff8f00"},
	{ID: 29, Name: "Golden white", Category: "White", Description: "Rich amber incandescent white", AccentColor: "#ffb300"},
	{ID: 30, Name: "Pulse", Category: "Dynamic", Description: "Rhythmic breathing pulse", AccentColor: "#00e5ff"},
	{ID: 31, Name: "Steampunk", Category: "Special", Description: "Copper & brass vintage warmth", AccentColor: "#bf360c"},
	{ID: 32, Name: "Rhythm", Category: "Rhythm", Description: "Circadian sync spectrum", AccentColor: "#00b0ff"},
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
