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
    "sceneId": 3,
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
    "sceneId": 3,
    "speed": 100,
    "dimming": 80
  }
}
```

## WiZ Dynamic Scenes Reference (Official IDs)
WiZ supports built-in dynamic lighting effects activated via `"sceneId"`:
- **1**: Ocean
- **2**: Romance
- **3**: Sunset
- **4**: Party
- **5**: Fireplace
- **6**: Cozy
- **7**: Forest
- **8**: Pastel colors
- **9**: Wake-up
- **10**: Bedtime
- **11**: Warm white (2700K)
- **12**: Daylight (6500K)
- **13**: Cool white (4000K)
- **14**: Night light
- **15**: Focus
- **16**: Relax
- **17**: True colors
- **18**: TV time
- **19**: Plant growth
- **20**: Spring
- **21**: Summer
- **22**: Fall
- **23**: Deep dive
- **24**: Jungle
- **25**: Mojito
- **26**: Club
- **27**: Christmas
- **28**: Halloween
- **29**: Candlelight
- **30**: Golden white
- **31**: Pulse
- **32**: Steampunk
- **33**: Diwali
- **1000**: Rhythm

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
