# Task 0.4: VPN Coexistence Validation Plan

## Objective

Verify that **Grey-Link** runs and establishes a transport connection alongside an active system-level VPN (e.g., ProtonVPN, WireGuard app, Corporate VPN) on Android.

## Prerequisites

1. **Android Device** (Api 26+).
2. **Grey-Link APK**: `android/app/build/outputs/apk/debug/app-debug.apk` (or install via ADB).
3. **Secondary VPN App**: An app that uses Android's `VpnService` (e.g., Google One VPN, DuckDuckGo Privacy, Tailscale app, etc.).
4. **Grey-Link Auth Key**: A reusable auth key from your Tailscale/Headscale admin console.

## Detailed Steps

### 1. Baseline: Activate System VPN

1. Open your secondary VPN app.
2. **Connect** it.
3. Verify a "Key" icon appears in your status bar.
4. Verify internet connectivity works (browse a site).

### 2. Install Grey-Link

1. Run `adb install -r android/app/build/outputs/apk/debug/app-debug.apk`.
2. Or copy the APK to device and install manually.

### 3. Launch & Connect Grey-Link

1. Open **GreyLink** app.
2. Enter your **Auth Key** in the text field.
3. Tap **Connect**.
4. **Observe**:
    * Status should update to "Starting..." then "Started".
    * **CRITICAL**: Android should **NOT** ask for "VPN Permission".
    * The "Key" icon in the status bar should **NOT** disappear or flicker.

### 4. Verification

1. **Check Logs**: Use Logcat to verify connection.

    ```bash
    adb logcat -s GreyLinkPoC GoLog
    ```

    Look for: `Transport started successfully` and `Tailscale IP: ...`.
2. **Check Coexistence**:
    * Switch back to your secondary VPN app. Verify it is **STILL CONNECTED**.
    * Browse the internet. Traffic should flow through the secondary VPN.
    * (Advanced) Ping the Grey-Link Android node from your Linux laptop over the Tailnet.

## Success Criteria

- [ ] Grey-Link connects successfully.
* [ ] System VPN remains active and uninterrupted.
* [ ] No VPN permission prompt was shown by Grey-Link.
