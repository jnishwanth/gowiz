package wiz

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Device represents a WiZ smart light device state and metadata
type Device struct {
	IP         string
	MAC        string
	Name       string
	Room       string
	Online     bool
	State      bool // Power state (ON/OFF)
	Brightness int  // 10 to 100
	RGB        [3]int
	Temp       int // 2200 to 6500 Kelvin
	SceneID    int
	Speed      int
	Rssi       int
	IsFallback bool
	LastSeen   time.Time
	Selected   bool // For TUI Visual mode multi-selection
}

func NewDevice(ip string) *Device {
	name := fmt.Sprintf("WiZ Light (%s)", ip)
	if ip == FallbackIP {
		name = fmt.Sprintf("WiZ Light (%s) [Fallback]", ip)
	}
	return &Device{
		IP:         ip,
		Name:       name,
		Online:     true,
		State:      true,
		Brightness: 100,
		RGB:        [3]int{255, 255, 255},
		Temp:       2700,
		SceneID:    0,
		Speed:      100,
		IsFallback: ip == FallbackIP,
		LastSeen:   time.Now(),
	}
}

// UpdateFromPilot updates device state from PilotParams
func (d *Device) UpdateFromPilot(p PilotParams) {
	d.LastSeen = time.Now()
	d.Online = true

	if p.State != nil {
		d.State = *p.State
	}
	if p.Dimming != nil {
		d.Brightness = *p.Dimming
	}
	if p.R != nil && p.G != nil && p.B != nil {
		d.RGB = [3]int{*p.R, *p.G, *p.B}
	}
	if p.Temp != nil {
		d.Temp = *p.Temp
	}
	if p.SceneID != nil {
		d.SceneID = *p.SceneID
	}
	if p.Speed != nil {
		d.Speed = *p.Speed
	}
	if p.Rssi != nil {
		d.Rssi = *p.Rssi
	}
	if p.Mac != "" {
		d.MAC = p.Mac
	}
}

// SignalPercentage maps device RSSI (ranging -90 dBm to -30 dBm) to a 0..100 percentage.
func (d *Device) SignalPercentage() int {
	if d.Rssi == 0 {
		return 0
	}
	if d.Rssi >= -30 {
		return 100
	}
	if d.Rssi <= -90 {
		return 0
	}
	return int(float64(d.Rssi+90) / 60.0 * 100.0)
}

// SignalQuality converts RSSI percentage to a human-readable quality rating string.
func (d *Device) SignalQuality() string {
	if d.Rssi == 0 {
		return "Unknown"
	}
	pct := d.SignalPercentage()
	switch {
	case pct >= 80:
		return "Excellent"
	case pct >= 60:
		return "Good"
	case pct >= 40:
		return "Fair"
	case pct >= 20:
		return "Poor"
	default:
		return "Weak"
	}
}

// SignalBar formats RSSI telemetry as a compact Wi-Fi status string.
func (d *Device) SignalBar() string {
	if d.Rssi == 0 {
		return "📶 Online"
	}
	return fmt.Sprintf("📶 %d%% (%d dBm)", d.SignalPercentage(), d.Rssi)
}

// DeviceRegistry provides a thread-safe store for discovered devices
type DeviceRegistry struct {
	mu           sync.RWMutex
	devices      map[string]*Device
	aliases      map[string]string // Persistent alias mapping (IP/MAC -> Custom Name)
	rooms        map[string]string // Persistent room mapping (IP/MAC -> Room Name)
	order        []string          // Ordering of IPs for UI rendering
	activeTarget string            // IP of currently focused bulb
}

func NewDeviceRegistry() *DeviceRegistry {
	reg := &DeviceRegistry{
		devices: make(map[string]*Device),
		aliases: make(map[string]string),
		rooms:   make(map[string]string),
	}
	// Seed with FallbackIP
	reg.AddOrUpdate(NewDevice(FallbackIP))
	reg.activeTarget = FallbackIP
	return reg
}

// ApplyAliases sets multiple bulb alias mappings and updates existing matching devices.
func (r *DeviceRegistry) ApplyAliases(aliases map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
	for k, v := range aliases {
		r.aliases[k] = v
	}

	for _, dev := range r.devices {
		if alias, ok := r.aliases[dev.IP]; ok && alias != "" {
			dev.Name = alias
		} else if dev.MAC != "" {
			if alias, ok := r.aliases[dev.MAC]; ok && alias != "" {
				dev.Name = alias
			}
		}
	}
}

// ApplyRooms sets multiple bulb room mappings and updates existing matching devices.
func (r *DeviceRegistry) ApplyRooms(rooms map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rooms == nil {
		r.rooms = make(map[string]string)
	}
	for k, v := range rooms {
		r.rooms[k] = v
	}

	for _, dev := range r.devices {
		if room, ok := r.rooms[dev.IP]; ok && room != "" {
			dev.Room = room
		} else if dev.MAC != "" {
			if room, ok := r.rooms[dev.MAC]; ok && room != "" {
				dev.Room = room
			}
		}
	}
}

func (r *DeviceRegistry) AddOrUpdate(dev *Device) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
	if r.rooms == nil {
		r.rooms = make(map[string]string)
	}

	if existing, found := r.devices[dev.IP]; found {
		existing.LastSeen = time.Now()
		existing.Online = true
		if dev.MAC != "" {
			existing.MAC = dev.MAC
		}
		if dev.Rssi != 0 {
			existing.Rssi = dev.Rssi
		}
		if alias, ok := r.aliases[dev.IP]; ok && alias != "" {
			existing.Name = alias
		} else if dev.MAC != "" {
			if alias, ok := r.aliases[dev.MAC]; ok && alias != "" {
				existing.Name = alias
			}
		}
		if room, ok := r.rooms[dev.IP]; ok && room != "" {
			existing.Room = room
		} else if dev.MAC != "" {
			if room, ok := r.rooms[dev.MAC]; ok && room != "" {
				existing.Room = room
			}
		}
	} else {
		if alias, ok := r.aliases[dev.IP]; ok && alias != "" {
			dev.Name = alias
		} else if dev.MAC != "" {
			if alias, ok := r.aliases[dev.MAC]; ok && alias != "" {
				dev.Name = alias
			}
		}
		if room, ok := r.rooms[dev.IP]; ok && room != "" {
			dev.Room = room
		} else if dev.MAC != "" {
			if room, ok := r.rooms[dev.MAC]; ok && room != "" {
				dev.Room = room
			}
		}
		r.devices[dev.IP] = dev
		r.order = append(r.order, dev.IP)
	}

	if r.activeTarget == "" {
		r.activeTarget = dev.IP
	}
}

func (r *DeviceRegistry) Get(ip string) (*Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dev, found := r.devices[ip]
	return dev, found
}

func (r *DeviceRegistry) GetActive() (*Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.activeTarget == "" {
		return nil, false
	}
	dev, found := r.devices[r.activeTarget]
	return dev, found
}

func (r *DeviceRegistry) SetActive(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := r.devices[ip]; found {
		r.activeTarget = ip
		return true
	}
	return false
}

func (r *DeviceRegistry) SetName(ip string, name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
	r.aliases[ip] = name

	if dev, found := r.devices[ip]; found {
		dev.Name = name
		return true
	}
	return false
}

func (r *DeviceRegistry) List() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Device
	for _, ip := range r.order {
		if dev, found := r.devices[ip]; found {
			result = append(result, dev)
		}
	}
	return result
}

func (r *DeviceRegistry) GetSelectedOrActive() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var selected []*Device
	for _, ip := range r.order {
		if dev, found := r.devices[ip]; found && dev.Selected {
			selected = append(selected, dev)
		}
	}
	if len(selected) > 0 {
		return selected
	}

	if active, found := r.devices[r.activeTarget]; found {
		return []*Device{active}
	}
	return nil
}

func (r *DeviceRegistry) ToggleSelection(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if dev, found := r.devices[ip]; found {
		dev.Selected = !dev.Selected
	}
}

func (r *DeviceRegistry) ClearSelections() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dev := range r.devices {
		dev.Selected = false
	}
}

func (r *DeviceRegistry) SelectAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dev := range r.devices {
		dev.Selected = true
	}
}

func (r *DeviceRegistry) SetRoom(ip string, room string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rooms == nil {
		r.rooms = make(map[string]string)
	}
	r.rooms[ip] = room

	if dev, found := r.devices[ip]; found {
		dev.Room = room
		return true
	}
	return false
}

func (r *DeviceRegistry) GetDevicesByRoom(room string) []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Device
	for _, ip := range r.order {
		if dev, found := r.devices[ip]; found {
			if strings.EqualFold(dev.Room, room) {
				result = append(result, dev)
			}
		}
	}
	return result
}

// GetDevicesByGroup returns devices matching members in a custom group map.
func (r *DeviceRegistry) GetDevicesByGroup(group string, groups map[string][]string) []*Device {
	groupKey := strings.ToLower(strings.TrimSpace(group))
	if len(groups) == 0 {
		return nil
	}
	members, found := groups[groupKey]
	if !found || len(members) == 0 {
		return nil
	}

	memberMap := make(map[string]bool)
	for _, m := range members {
		memberMap[strings.ToLower(strings.TrimSpace(m))] = true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Device
	for _, ip := range r.order {
		if dev, found := r.devices[ip]; found {
			matched := memberMap[strings.ToLower(dev.IP)] ||
				(dev.MAC != "" && memberMap[strings.ToLower(dev.MAC)]) ||
				(dev.Name != "" && memberMap[strings.ToLower(dev.Name)])
			if matched {
				result = append(result, dev)
			}
		}
	}
	return result
}

// GetDevicesBySelector resolves devices by wildcard target ("all", "*", "everyone", "broadcast"), custom group, room name, or direct device IP/MAC/Name match.
func (r *DeviceRegistry) GetDevicesBySelector(target string, groups map[string][]string) []*Device {
	targetLower := strings.ToLower(strings.TrimSpace(target))
	if targetLower == "" {
		return nil
	}
	if targetLower == "all" || targetLower == "*" || targetLower == "everyone" || targetLower == "broadcast" {
		return r.List()
	}
	if devs := r.GetDevicesByGroup(targetLower, groups); len(devs) > 0 {
		return devs
	}
	if devs := r.GetDevicesByRoom(targetLower); len(devs) > 0 {
		return devs
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ip := range r.order {
		if dev, found := r.devices[ip]; found {
			if strings.EqualFold(dev.IP, targetLower) ||
				(dev.MAC != "" && strings.EqualFold(dev.MAC, targetLower)) ||
				(dev.Name != "" && strings.EqualFold(dev.Name, targetLower)) {
				return []*Device{dev}
			}
		}
	}
	return nil
}

// GetDevicesByGroupOrRoom resolves devices by selector (all/wildcard, custom group, room name, or individual device).
func (r *DeviceRegistry) GetDevicesByGroupOrRoom(target string, groups map[string][]string) []*Device {
	return r.GetDevicesBySelector(target, groups)
}

