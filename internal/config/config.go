package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Preset stores a reusable lighting configuration snapshot (dimming, color temp, RGB, or scene).
type Preset struct {
	Dimming *int `json:"dimming,omitempty"`
	Temp    *int `json:"temp,omitempty"`
	R       *int `json:"r,omitempty"`
	G       *int `json:"g,omitempty"`
	B       *int `json:"b,omitempty"`
	SceneID *int `json:"scene_id,omitempty"`
	Speed   *int `json:"speed,omitempty"`
}

// Config represents persistent user settings and custom bulb aliases.
type Config struct {
	DeviceAliases map[string]string `json:"device_aliases"`
	DeviceRooms   map[string]string `json:"device_rooms,omitempty"`
	Presets       map[string]Preset `json:"presets,omitempty"`
	LastActiveIP  string            `json:"last_active_ip,omitempty"`
	RecentIPs     []string          `json:"recent_ips,omitempty"`
	AutoScan      bool              `json:"auto_scan"`
}

// Manager manages concurrent-safe loading, saving, and querying of the gowiz config file.
type Manager struct {
	mu       sync.RWMutex
	filePath string
	cfg      Config
}

// DefaultConfig returns standard initial configuration values.
func DefaultConfig() Config {
	return Config{
		DeviceAliases: make(map[string]string),
		DeviceRooms:   make(map[string]string),
		Presets:       make(map[string]Preset),
		RecentIPs:     make([]string, 0),
		AutoScan:      true,
	}
}

// DefaultPath determines the standard user configuration file location (~/.config/gowiz/config.json).
func DefaultPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return ".gowiz.json"
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "gowiz", "config.json")
}

// NewManager constructs a Manager instance targeting the specified file path.
// If path is empty, it uses DefaultPath().
func NewManager(path string) *Manager {
	if path == "" {
		path = DefaultPath()
	}
	return &Manager{
		filePath: path,
		cfg:      DefaultConfig(),
	}
}

// Load reads and unmarshals the JSON configuration file from disk.
// If the file does not exist, it defaults gracefully.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.cfg = DefaultConfig()
			return nil
		}
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if cfg.DeviceAliases == nil {
		cfg.DeviceAliases = make(map[string]string)
	}
	if cfg.DeviceRooms == nil {
		cfg.DeviceRooms = make(map[string]string)
	}
	if cfg.Presets == nil {
		cfg.Presets = make(map[string]Preset)
	}
	m.cfg = cfg
	return nil
}

// Save marshals and writes the configuration to disk, creating parent directories if needed.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// GetConfig returns a deep copy of the current configuration.
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	aliasesCopy := make(map[string]string)
	for k, v := range m.cfg.DeviceAliases {
		aliasesCopy[k] = v
	}

	roomsCopy := make(map[string]string)
	for k, v := range m.cfg.DeviceRooms {
		roomsCopy[k] = v
	}

	presetsCopy := make(map[string]Preset)
	for k, v := range m.cfg.Presets {
		presetsCopy[k] = v
	}

	recentCopy := make([]string, len(m.cfg.RecentIPs))
	copy(recentCopy, m.cfg.RecentIPs)

	return Config{
		DeviceAliases: aliasesCopy,
		DeviceRooms:   roomsCopy,
		Presets:       presetsCopy,
		LastActiveIP:  m.cfg.LastActiveIP,
		RecentIPs:     recentCopy,
		AutoScan:      m.cfg.AutoScan,
	}
}

// FilePath returns the configured file path.
func (m *Manager) FilePath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.filePath
}

// SetAlias assigns a custom friendly name to a bulb IP or MAC address and saves to disk.
func (m *Manager) SetAlias(identifier, alias string) error {
	m.mu.Lock()
	if m.cfg.DeviceAliases == nil {
		m.cfg.DeviceAliases = make(map[string]string)
	}
	if alias == "" {
		delete(m.cfg.DeviceAliases, identifier)
	} else {
		m.cfg.DeviceAliases[identifier] = alias
	}
	m.mu.Unlock()
	return m.Save()
}

// GetAlias retrieves the custom alias for a bulb identifier, if present.
func (m *Manager) GetAlias(identifier string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	alias, found := m.cfg.DeviceAliases[identifier]
	return alias, found
}

// SetLastActiveIP updates and persists the last active device IP address.
func (m *Manager) SetLastActiveIP(ip string) error {
	m.mu.Lock()
	if m.cfg.LastActiveIP == ip {
		m.mu.Unlock()
		return nil
	}
	m.cfg.LastActiveIP = ip
	m.mu.Unlock()
	return m.Save()
}

// GetLastActiveIP retrieves the last active bulb IP address from config.
func (m *Manager) GetLastActiveIP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.LastActiveIP
}

// AddRecentIP adds an IP to the recent targets list (deduplicating and capping at 10) and sets LastActiveIP.
func (m *Manager) AddRecentIP(ip string) error {
	if ip == "" {
		return nil
	}
	m.mu.Lock()
	m.cfg.LastActiveIP = ip

	newList := []string{ip}
	for _, existing := range m.cfg.RecentIPs {
		if existing != ip {
			newList = append(newList, existing)
		}
	}
	if len(newList) > 10 {
		newList = newList[:10]
	}
	m.cfg.RecentIPs = newList
	m.mu.Unlock()

	return m.Save()
}

// GetRecentIPs retrieves a copy of the recent bulb target IP history.
func (m *Manager) GetRecentIPs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	recentCopy := make([]string, len(m.cfg.RecentIPs))
	copy(recentCopy, m.cfg.RecentIPs)
	return recentCopy
}

// SetRoom assigns a room name to a bulb IP or MAC address and saves to disk.
func (m *Manager) SetRoom(identifier, room string) error {
	m.mu.Lock()
	if m.cfg.DeviceRooms == nil {
		m.cfg.DeviceRooms = make(map[string]string)
	}
	if room == "" {
		delete(m.cfg.DeviceRooms, identifier)
	} else {
		m.cfg.DeviceRooms[identifier] = room
	}
	m.mu.Unlock()
	return m.Save()
}

// GetRoom retrieves the room name for a bulb identifier, if present.
func (m *Manager) GetRoom(identifier string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room, found := m.cfg.DeviceRooms[identifier]
	return room, found
}

// SetPreset stores a custom lighting preset by name and saves to disk.
func (m *Manager) SetPreset(name string, preset Preset) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil
	}
	m.mu.Lock()
	if m.cfg.Presets == nil {
		m.cfg.Presets = make(map[string]Preset)
	}
	m.cfg.Presets[name] = preset
	m.mu.Unlock()
	return m.Save()
}

// DeletePreset removes a custom lighting preset by name and saves to disk.
func (m *Manager) DeletePreset(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	m.mu.Lock()
	if m.cfg.Presets != nil {
		delete(m.cfg.Presets, name)
	}
	m.mu.Unlock()
	return m.Save()
}

// GetPreset retrieves a custom lighting preset by name, if present.
func (m *Manager) GetPreset(name string) (Preset, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cfg.Presets == nil {
		return Preset{}, false
	}
	p, found := m.cfg.Presets[strings.ToLower(strings.TrimSpace(name))]
	return p, found
}

// GetPresets returns a copy of all stored custom lighting presets.
func (m *Manager) GetPresets() map[string]Preset {
	m.mu.RLock()
	defer m.mu.RUnlock()
	presetsCopy := make(map[string]Preset)
	for k, v := range m.cfg.Presets {
		presetsCopy[k] = v
	}
	return presetsCopy
}

