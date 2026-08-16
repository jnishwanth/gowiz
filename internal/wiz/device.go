package wiz

import (
	"fmt"
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

// DeviceRegistry provides a thread-safe store for discovered devices
type DeviceRegistry struct {
	mu           sync.RWMutex
	devices      map[string]*Device
	aliases      map[string]string // Persistent alias mapping (IP/MAC -> Custom Name)
	order        []string          // Ordering of IPs for UI rendering
	activeTarget string            // IP of currently focused bulb
}

func NewDeviceRegistry() *DeviceRegistry {
	reg := &DeviceRegistry{
		devices: make(map[string]*Device),
		aliases: make(map[string]string),
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

func (r *DeviceRegistry) AddOrUpdate(dev *Device) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}

	if existing, found := r.devices[dev.IP]; found {
		existing.LastSeen = time.Now()
		existing.Online = true
		if dev.MAC != "" {
			existing.MAC = dev.MAC
		}
		if alias, ok := r.aliases[dev.IP]; ok && alias != "" {
			existing.Name = alias
		} else if dev.MAC != "" {
			if alias, ok := r.aliases[dev.MAC]; ok && alias != "" {
				existing.Name = alias
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
