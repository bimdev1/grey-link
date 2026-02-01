# Native Netmon Implementation

**Goal:** Resolve the `netlinkrib` permission crash on Android by replacing Linux syscalls with Android's `ConnectivityManager` API.

## Changes Implemented

### 1. Go Side (`pkg/transport/transport.go`)

We replaced the previous "Bunker" (fake loopback) or "Raw" (syscall) approaches with a **Push-Based Native Netmon**.

- **State Injection:** Added `AndroidState` struct and `UpdateNetworkState(jsonStr string)` function. This allows the Android app to push the network state (interfaces, IPs, MTU) directly into Go memory.
- **Netmon Hook:** `init()` calls `netmon.RegisterInterfaceGetter` to use our injected state instead of `net.Interfaces()`.
- **Netutil Hook:** `init()` also assigns `netutil.GetDefaultInterface` to a function that looks up the default route in our injected state.
- **Panic Guards:** Added `defer recover()` to `Start` and `UpdateNetworkState` to prevent native crashes from killing the app.
- **Safe Defaults:** If state hasn't arrived yet, we fall back to a dummy loopback to prevent startup errors.
- **XDG_CACHE_HOME:** Explicitly set to `stateDir/cache` to fix `filch` (Tailscale logger) panic on Android.
- **Result:** `tsnet` and `magicsock` see the real network interfaces (Wi-Fi, Cellular) without making any syscalls.

### 2. Android Side (`MainActivity.kt`)

We implemented `ConnectivityManager.NetworkCallback` to monitor network changes in real-time.

- **Network Callback:** We listen for `onLinkPropertiesChanged`.
- **API Compatibility:** We use `java.net.NetworkInterface` to fetch MTU values because `LinkProperties.getMtu()` is only available on API 29+. This ensures support for older devices (API 26+).
- **Push:** Whenever state changes, we serialize the interface list to JSON:

  ```json
  {
      "Interfaces": [
          { "Name": "wlan0", "MTU": 1500, "Addrs": ["192.168.1.105/24", "fe80::..."] }
      ],
      "DefaultInterface": "wlan0"
  }
  ```

- **Bridge:** We call `Transport.updateNetworkState(json)` via gomobile.
- **UI Feedback:** The app now polls for the assigned Tailscale IP and displays "Connected: [IP]" once established.

## Verification

### Build Verification

1. **Go Bind:** Successfully generated `transport.aar` (Target API 24).
2. **Android Build:** `./gradlew assembleDebug` passed successfully.

### Functional Verification

- **Crash Fix:** `netlinkrib` syscalls are bypassed because `netmon` uses our getter.
- **Roaming:** When Android switches to Cellular, `onLinkPropertiesChanged` fires -> `UpdateNetworkState` is called -> `netmon` sees the new IPs -> `magicsock` rebinds.
- **IP Display:** The UI correctly waits for and displays the Tailscale IP (e.g., `100.117.190.110`).

## Build Instructions

1. **Rebuild AAR:**

    ```bash
    export ANDROID_NDK_HOME=$HOME/Android/Sdk/ndk/26.1.10909125
    gomobile bind -target=android -androidapi 24 -o android/app/libs/transport.aar ./pkg/transport
    ```

2. **Build APK:**

    ```bash
    cd android && ./gradlew clean assembleDebug
    ```
