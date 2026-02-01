# Grey-Link

> Invisible, persistent TCP/IP tunnel between Android and Linux devices

## What Is This?

Grey-Link creates a "dumb cable" between your Android phone and Linux desktop. Applications communicate as if directly connected—no VPN permission required, no conflict with your corporate VPN.

## Key Features

- **No VPN Permission** — Uses embedded tsnet, not Android VpnService
- **VPN Coexistence** — Works alongside corporate/privacy VPNs
- **Self-Healing** — Automatically reconnects after network changes
- **Secure** — All traffic encrypted with WireGuard
- **Private** — Credentials encrypted at rest on both devices

## Status

🚧 **Development** — See [implementation plan](./implementation_plan.md)

## Quick Start

### Linux

```bash
# Install
go install github.com/bimdev1/grey-link/cmd/grey-link-daemon@latest
go install github.com/bimdev1/grey-link/cmd/grey-link@latest

# Start daemon
grey-link daemon start

# Generate pairing QR
grey-link pair
```

### Android

1. Install Grey-Link from F-Droid
2. Open app, tap "Scan QR Code"
3. Scan the QR from your Linux terminal
4. Done — tunnel is active

## Architecture

```
Android App ──▶ greylink-transport (Go) ──▶ WireGuard ──▶ Linux Daemon
                     │                                          │
                └── tsnet (userspace) ──────────────────────────┘
```

See [.developer/design-spec.md](.developer/design-spec.md) for full architecture.

## Requirements

| Platform | Requirement |
|----------|-------------|
| Linux | Go 1.22+, systemd |
| Android | 8.0+ (API 26), F-Droid |

## Development

```bash
# Clone
git clone https://github.com/bimdev1/grey-link.git
cd grey-link

# Build Linux binaries
make build-linux

# Build Android library
make build-android

# Run tests
make test
```

## License

[AGPL-3.0](LICENSE)

## Roadmap

- [x] Research: tsnet feasibility ✅
- [x] Research: VPN coexistence ✅
- [ ] **PoC**: Validate architecture
- [ ] **MVP 1.0**: Core tunnel functionality
- [ ] **F-Droid**: Initial release
