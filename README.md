# Grey-Link

> Invisible, persistent TCP/IP tunnel between Android and Linux devices

## What Is This?

Grey-Link creates a "dumb cable" between your Android phone and Linux desktop. Applications communicate as if directly connected—no VPN permission required, no conflict with your corporate VPN.

## Key Features

- **No VPN Permission** — Uses embedded tsnet, not Android VpnService
- **VPN Coexistence** — Works alongside corporate/privacy VPNs
- **Self-Healing** — Automatically reconnects after network changes (Native Netmon)
- **Secure** — All traffic encrypted with WireGuard
- **Private** — Credentials encrypted at rest on both devices

## Status

✅ **Proof of Concept Complete** — Android app connects and displays Tailscale IP

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Android Device                           │
│  ┌──────────────────┐     ┌────────────────────────────────┐   │
│  │   MainActivity   │────▶│   transport.aar (gomobile)     │   │
│  │  (Kotlin/UI)     │     │   └─ tsnet (userspace WireGuard)│   │
│  │                  │     │   └─ Native Netmon Hook         │   │
│  └──────────────────┘     └────────────────────────────────┘   │
│           │                              │                      │
│           ▼                              ▼                      │
│   ConnectivityManager ──────────▶ UpdateNetworkState()          │
│   (Push network state)           (No netlink syscalls!)         │
└─────────────────────────────────────────────────────────────────┘
                                   │
                              WireGuard
                              (encrypted)
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Linux Desktop                            │
│  ┌──────────────────┐     ┌────────────────────────────────┐   │
│  │  grey-link CLI   │────▶│   grey-link-daemon             │   │
│  │  (user commands) │     │   └─ tsnet                      │   │
│  └──────────────────┘     └────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Native Netmon (Android)

Android restricts `netlink` syscalls, which causes Go's `net.Interfaces()` to crash on modern Android versions. Grey-Link solves this with **Native Netmon**:

1. **ConnectivityManager Callback** — Kotlin code registers a `NetworkCallback` to receive real-time network state updates.
2. **JSON State Push** — Network info (interface name, IPs, MTU) is serialized to JSON and pushed into Go.
3. **Hook Registration** — Go `init()` registers custom interface getters that use the pushed state instead of syscalls.

**Result:** No crashes, full roaming support, local peer discovery works.

See [docs/NATIVE_NETMON.md](docs/NATIVE_NETMON.md) for implementation details.

## Requirements

| Platform | Requirement |
|----------|-------------|
| Linux | Go 1.22+, systemd |
| Android | 8.0+ (API 26) |
| Build | Android NDK 26+, gomobile |

## Development

### Prerequisites

```bash
# Install Go
# Install Android SDK and NDK
export ANDROID_HOME=$HOME/Android/Sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/26.1.10909125

# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

### Build

```bash
# Clone
git clone https://github.com/bimdev1/grey-link.git
cd grey-link

# Build Go transport library for Android
gomobile bind -target=android -androidapi 24 -o android/app/libs/transport.aar ./pkg/transport

# Build Android APK
cd android && ./gradlew assembleDebug
```

### Project Structure

```
grey-link/
├── pkg/transport/          # Go bridge (gomobile bindings)
│   └── transport.go        # UpdateNetworkState, Start, Stop
├── android/                # Android app
│   └── app/src/main/java/  # Kotlin source
│       └── MainActivity.kt # UI + NetworkCallback
├── tailscale-fork/         # Vendored Tailscale with patches
│   └── net/netutil/        # Default interface hook
├── docs/
│   └── NATIVE_NETMON.md    # Implementation walkthrough
└── .developer/             # Design specs and diagrams
```

## Roadmap

- [x] Research: tsnet feasibility
- [x] Research: VPN coexistence
- [x] Research: Native Netmon architecture
- [x] **PoC**: Validate architecture
- [x] **Fix**: Resolve `netlinkrib` crash with Native Netmon
- [ ] **MVP 1.0**: Core tunnel functionality (Linux daemon)
- [ ] **Pairing**: QR code pairing flow
- [ ] **UI**: Polish Android UI
- [ ] **F-Droid**: Initial release

## License

[AGPL-3.0](LICENSE)

---

*Built with [tsnet](https://pkg.go.dev/tailscale.com/tsnet) and [WireGuard](https://www.wireguard.com/).*
