graph TD
    subgraph Android Device
        App[Android App Layer]
        Adapter[Transport Adapter]
        TS_Android[tsnet (Embedded)]
    end

    subgraph Linux Device
        Daemon[Linux Daemon]
        TS_Linux[tsnet (Embedded)]
    end

    App -->|1. Request Identity| Adapter
    Adapter -->|2. Generate Ephemeral Key| TS_Android
    TS_Android -->|3. Return NodeID/Key| Adapter
    Adapter -->|4. Return Identity| App

    Daemon -->|1. Init| TS_Linux
    TS_Linux -->|2. Register with Control Plane| ControlPlane
    ControlPlane -->|3. Return NodeID| Daemon
    Daemon -->|4. Generate QR (Config)| Terminal

    App -->|5. Scan QR| Adapter
    Adapter -->|6. Configure Transport| TS_Android
    TS_Android <-->|7. Handshake (DERP/P2P)| TS_Linux
    TS_Linux -->|8. Establish Tunnel| Daemon
