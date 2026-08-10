---
name: wiz-protocol
description: Expert reference skill for WiZ smart light UDP JSON-RPC protocol methods, scenes, and Bubble Tea TUI implementation patterns.
---

# WiZ Smart Light Protocol & TUI Development Skill

This skill provides an authoritative technical reference for working with the WiZ local UDP protocol and developing terminal interfaces for WiZ smart lights.

## WiZ Local UDP Protocol (Port 38899)

WiZ smart lights communicate locally over unencrypted JSON-RPC 1.0 via UDP port **38899**.

### 1. `setPilot` (Set Device State)
```json
{
  "method": "setPilot",
  "params": {
    "state": true,
    "dimming": 80,
    "r": 255,
    "g": 100,
    "b": 50,
    "temp": 2700,
    "sceneId": 2,
    "speed": 100
  }
}
```

### 2. `getPilot` (Query Device Telemetry)
```json
{
  "method": "getPilot",
  "params": {}
}
```
**Sample Response:**
```json
{
  "method": "getPilot",
  "env": "pro",
  "result": {
    "mac": "a8bb50123456",
    "rssi": -62,
    "state": true,
    "sceneId": 2,
    "speed": 100,
    "dimming": 80
  }
}
```

### 3. `getSystemConfig` (Network Discovery)
Broadcast payload sent to subnet broadcast IP (e.g., `192.168.1.255:38899`):
```json
{
  "method": "getSystemConfig",
  "params": {}
}
```

## WiZ Dynamic Scenes Reference (IDs 1-32)
WiZ supports 32 dynamic lighting effects activated via `"sceneId"`:
- **1**: Ocean
- **2**: Sunset
- **3**: Party
- **4**: Fireplace
- **5**: Cozy
- **6**: Forest
- **7**: Pastel Colors
- **8**: Wake up
- **9**: Bedtime
- **10**: Warm White (2700K)
- **11**: Daylight (6500K)
- **12**: Cool White (4000K)
- **13**: Night Light
- **14**: Focus
- **15**: Relax
- **16**: True colors
- **17**: TV time
- **18**: Plant growth
- **19**: Spring
- **20**: Summer
- **21**: Fall
- **22**: Deep dive
- **23**: Jungle
- **24**: Mojito
- **25**: Club
- **26**: Christmas
- **27**: Halloween
- **28**: Candlelight
- **29**: Golden white
- **30**: Pulse
- **31**: Steampunk
- **32**: Rhythm

## Verification Workflow
Always run:
```bash
./scripts/verify.sh
```
or
```bash
make verify
```
to validate code changes against all 3 Tiers before committing.
