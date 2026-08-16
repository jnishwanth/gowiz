package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config represents persistent user settings and custom bulb aliases.
type Config struct {
	DeviceAliases map[string]string `json:"device_aliases"`
	LastActiveIP  string            `json:"last_active_ip,omitempty"`
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

	return Config{
		DeviceAliases: aliasesCopy,
		LastActiveIP:  m.cfg.LastActiveIP,
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
