package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"wiz-tui/internal/circadian"
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
	DeviceAliases       map[string]string                    `json:"device_aliases"`
	DeviceRooms         map[string]string                    `json:"device_rooms,omitempty"`
	DeviceGroups        map[string][]string                  `json:"device_groups,omitempty"`
	Presets             map[string]Preset                    `json:"presets,omitempty"`
	CircadianPhases     []circadian.SchedulePhase            `json:"circadian_phases,omitempty"`
	RoomCircadianPhases map[string][]circadian.SchedulePhase `json:"room_circadian_phases,omitempty"`
	LastActiveIP        string                               `json:"last_active_ip,omitempty"`
	RecentIPs           []string                             `json:"recent_ips,omitempty"`
	AutoScan            bool                                 `json:"auto_scan"`
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
		DeviceAliases:       make(map[string]string),
		DeviceRooms:         make(map[string]string),
		DeviceGroups:        make(map[string][]string),
		Presets:             make(map[string]Preset),
		RoomCircadianPhases: make(map[string][]circadian.SchedulePhase),
		RecentIPs:           make([]string, 0),
		AutoScan:            true,
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
	if cfg.DeviceGroups == nil {
		cfg.DeviceGroups = make(map[string][]string)
	}
	if cfg.Presets == nil {
		cfg.Presets = make(map[string]Preset)
	}
	if cfg.RoomCircadianPhases == nil {
		cfg.RoomCircadianPhases = make(map[string][]circadian.SchedulePhase)
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

	groupsCopy := make(map[string][]string)
	for k, members := range m.cfg.DeviceGroups {
		if len(members) > 0 {
			mCopy := make([]string, len(members))
			copy(mCopy, members)
			groupsCopy[k] = mCopy
		}
	}

	presetsCopy := make(map[string]Preset)
	for k, v := range m.cfg.Presets {
		presetsCopy[k] = v
	}

	recentCopy := make([]string, len(m.cfg.RecentIPs))
	copy(recentCopy, m.cfg.RecentIPs)

	var phasesCopy []circadian.SchedulePhase
	if len(m.cfg.CircadianPhases) > 0 {
		phasesCopy = make([]circadian.SchedulePhase, len(m.cfg.CircadianPhases))
		copy(phasesCopy, m.cfg.CircadianPhases)
	}

	roomPhasesCopy := make(map[string][]circadian.SchedulePhase)
	for room, phases := range m.cfg.RoomCircadianPhases {
		if len(phases) > 0 {
			pCopy := make([]circadian.SchedulePhase, len(phases))
			copy(pCopy, phases)
			roomPhasesCopy[room] = pCopy
		}
	}

	return Config{
		DeviceAliases:       aliasesCopy,
		DeviceRooms:         roomsCopy,
		DeviceGroups:        groupsCopy,
		Presets:             presetsCopy,
		CircadianPhases:     phasesCopy,
		RoomCircadianPhases: roomPhasesCopy,
		LastActiveIP:        m.cfg.LastActiveIP,
		RecentIPs:           recentCopy,
		AutoScan:            m.cfg.AutoScan,
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

// ExportToFile writes a copy of the current configuration to the specified destination path.
func (m *Manager) ExportToFile(destPath string) error {
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return fmt.Errorf("export file path cannot be empty")
	}

	m.mu.RLock()
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	m.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("failed to marshal export config: %w", err)
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	return os.WriteFile(destPath, data, 0644)
}

// ImportFromFile reads and merges configuration from the specified file path into current settings.
func (m *Manager) ImportFromFile(srcPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	if srcPath == "" {
		return fmt.Errorf("import file path cannot be empty")
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var imported Config
	if err := json.Unmarshal(data, &imported); err != nil {
		return fmt.Errorf("invalid config JSON format: %w", err)
	}

	m.mu.Lock()
	if imported.DeviceAliases != nil {
		if m.cfg.DeviceAliases == nil {
			m.cfg.DeviceAliases = make(map[string]string)
		}
		for k, v := range imported.DeviceAliases {
			m.cfg.DeviceAliases[k] = v
		}
	}
	if imported.DeviceRooms != nil {
		if m.cfg.DeviceRooms == nil {
			m.cfg.DeviceRooms = make(map[string]string)
		}
		for k, v := range imported.DeviceRooms {
			m.cfg.DeviceRooms[k] = v
		}
	}
	if imported.DeviceGroups != nil {
		if m.cfg.DeviceGroups == nil {
			m.cfg.DeviceGroups = make(map[string][]string)
		}
		for group, members := range imported.DeviceGroups {
			if len(members) > 0 {
				mCopy := make([]string, len(members))
				copy(mCopy, members)
				m.cfg.DeviceGroups[strings.ToLower(strings.TrimSpace(group))] = mCopy
			}
		}
	}
	if imported.Presets != nil {
		if m.cfg.Presets == nil {
			m.cfg.Presets = make(map[string]Preset)
		}
		for k, v := range imported.Presets {
			m.cfg.Presets[k] = v
		}
	}
	if len(imported.CircadianPhases) > 0 {
		m.cfg.CircadianPhases = make([]circadian.SchedulePhase, len(imported.CircadianPhases))
		copy(m.cfg.CircadianPhases, imported.CircadianPhases)
	}
	if imported.RoomCircadianPhases != nil {
		if m.cfg.RoomCircadianPhases == nil {
			m.cfg.RoomCircadianPhases = make(map[string][]circadian.SchedulePhase)
		}
		for room, phases := range imported.RoomCircadianPhases {
			if len(phases) > 0 {
				pCopy := make([]circadian.SchedulePhase, len(phases))
				copy(pCopy, phases)
				m.cfg.RoomCircadianPhases[strings.ToLower(strings.TrimSpace(room))] = pCopy
			}
		}
	}
	for _, ip := range imported.RecentIPs {
		if ip != "" {
			found := false
			for _, existing := range m.cfg.RecentIPs {
				if existing == ip {
					found = true
					break
				}
			}
			if !found {
				m.cfg.RecentIPs = append(m.cfg.RecentIPs, ip)
			}
		}
	}
	if len(m.cfg.RecentIPs) > 10 {
		m.cfg.RecentIPs = m.cfg.RecentIPs[:10]
	}
	m.mu.Unlock()

	return m.Save()
}

// SetGroup assigns a list of device member identifiers to a custom group name and saves to disk.
func (m *Manager) SetGroup(name string, members []string) error {
	groupKey := strings.ToLower(strings.TrimSpace(name))
	if groupKey == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	m.mu.Lock()
	if m.cfg.DeviceGroups == nil {
		m.cfg.DeviceGroups = make(map[string][]string)
	}
	if len(members) == 0 {
		delete(m.cfg.DeviceGroups, groupKey)
	} else {
		unique := make([]string, 0, len(members))
		seen := make(map[string]bool)
		for _, mem := range members {
			mem = strings.TrimSpace(mem)
			if mem != "" && !seen[mem] {
				seen[mem] = true
				unique = append(unique, mem)
			}
		}
		if len(unique) == 0 {
			delete(m.cfg.DeviceGroups, groupKey)
		} else {
			m.cfg.DeviceGroups[groupKey] = unique
		}
	}
	m.mu.Unlock()
	return m.Save()
}

// DeleteGroup removes a custom group by name and saves to disk.
func (m *Manager) DeleteGroup(name string) error {
	groupKey := strings.ToLower(strings.TrimSpace(name))
	m.mu.Lock()
	if m.cfg.DeviceGroups != nil {
		delete(m.cfg.DeviceGroups, groupKey)
	}
	m.mu.Unlock()
	return m.Save()
}

// GetGroup retrieves member identifiers for a custom group, if present.
func (m *Manager) GetGroup(name string) ([]string, bool) {
	groupKey := strings.ToLower(strings.TrimSpace(name))
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cfg.DeviceGroups == nil {
		return nil, false
	}
	members, found := m.cfg.DeviceGroups[groupKey]
	if !found {
		return nil, false
	}
	membersCopy := make([]string, len(members))
	copy(membersCopy, members)
	return membersCopy, true
}

// GetGroups returns a copy of all configured custom device groups.
func (m *Manager) GetGroups() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cfg.DeviceGroups == nil {
		return make(map[string][]string)
	}
	groupsCopy := make(map[string][]string)
	for k, members := range m.cfg.DeviceGroups {
		if len(members) > 0 {
			mCopy := make([]string, len(members))
			copy(mCopy, members)
			groupsCopy[k] = mCopy
		}
	}
	return groupsCopy
}

// SetCircadianPhases updates and persists custom 24-hour circadian schedule phases.
func (m *Manager) SetCircadianPhases(phases []circadian.SchedulePhase) error {
	if len(phases) > 0 {
		if err := circadian.ValidatePhases(phases); err != nil {
			return err
		}
	}
	m.mu.Lock()
	if len(phases) == 0 {
		m.cfg.CircadianPhases = nil
	} else {
		m.cfg.CircadianPhases = make([]circadian.SchedulePhase, len(phases))
		copy(m.cfg.CircadianPhases, phases)
	}
	m.mu.Unlock()
	return m.Save()
}

// GetCircadianPhases retrieves a copy of custom circadian schedule phases, if set.
func (m *Manager) GetCircadianPhases() []circadian.SchedulePhase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.cfg.CircadianPhases) == 0 {
		return nil
	}
	phasesCopy := make([]circadian.SchedulePhase, len(m.cfg.CircadianPhases))
	copy(phasesCopy, m.cfg.CircadianPhases)
	return phasesCopy
}

// SetRoomCircadianPhases updates and persists custom 24-hour circadian schedule phases for a specific room.
func (m *Manager) SetRoomCircadianPhases(room string, phases []circadian.SchedulePhase) error {
	roomKey := strings.ToLower(strings.TrimSpace(room))
	if roomKey == "" {
		return fmt.Errorf("room name cannot be empty")
	}
	if len(phases) > 0 {
		if err := circadian.ValidatePhases(phases); err != nil {
			return err
		}
	}
	m.mu.Lock()
	if m.cfg.RoomCircadianPhases == nil {
		m.cfg.RoomCircadianPhases = make(map[string][]circadian.SchedulePhase)
	}
	if len(phases) == 0 {
		delete(m.cfg.RoomCircadianPhases, roomKey)
	} else {
		phasesCopy := make([]circadian.SchedulePhase, len(phases))
		copy(phasesCopy, phases)
		m.cfg.RoomCircadianPhases[roomKey] = phasesCopy
	}
	m.mu.Unlock()
	return m.Save()
}

// GetRoomCircadianPhases retrieves custom circadian schedule phases configured specifically for a room.
func (m *Manager) GetRoomCircadianPhases(room string) []circadian.SchedulePhase {
	roomKey := strings.ToLower(strings.TrimSpace(room))
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cfg.RoomCircadianPhases == nil {
		return nil
	}
	phases, found := m.cfg.RoomCircadianPhases[roomKey]
	if !found || len(phases) == 0 {
		return nil
	}
	phasesCopy := make([]circadian.SchedulePhase, len(phases))
	copy(phasesCopy, phases)
	return phasesCopy
}

// GetCircadianPhasesForRoom retrieves room-specific circadian phases if configured, or falls back to global custom phases.
func (m *Manager) GetCircadianPhasesForRoom(room string) []circadian.SchedulePhase {
	roomPhases := m.GetRoomCircadianPhases(room)
	if len(roomPhases) > 0 {
		return roomPhases
	}
	return m.GetCircadianPhases()
}
