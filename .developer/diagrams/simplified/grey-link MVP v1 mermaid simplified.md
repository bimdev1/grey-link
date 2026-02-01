graph TD
    User([User]) -->|Takes Photo| AndroidApp
    AndroidApp -->|Decodes Config| Adapter
    Adapter -->|Connects| TSNet[Embed: tsnet]
    TSNet <-->|WireGuard| LinuxDaemon
    LinuxDaemon -->|Accepts| LocalSocket
    AndroidApp -->|Connects TCP| LocalSocket
