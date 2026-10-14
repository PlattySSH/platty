# Changelog

All notable changes to Platty are listed here. Versions follow [Semantic Versioning](https://semver.org).

## 0.1.0 (unreleased)

The first public beta.

### Terminal
- Own VT emulator with truecolor, alternate screen, scrollback, wide characters and reflow on resize.
- Custom glyph rasterizer with color emoji, font fallback and ligatures (JetBrains Mono, Fira Code, Geist Mono).
- Tabs and split panes that come back when Platty reopens; rename, reorder and duplicate tabs.
- Broadcast input to every pane of a tab, and plain-text session logs.
- Search in the scrollback, Ctrl+click links (including OSC 8), OSC 52 clipboard writes and OSC 7 working folder.

### SSH
- Hosts in folders with drag and drop, tags, search and `~/.ssh/config` import and export.
- Password, key, agent and keyboard-interactive authentication; jump hosts, keep-alive and reconnect.
- Host key verification with a warning when a key changes.
- Keychain with ed25519 key generation, import, SSH certificates, and passphrases kept per session unless remembered.
- Agent forwarding and a command to run on connect, per host.
- Local, remote and dynamic (SOCKS5) port forwarding.
- SFTP panel with a transfer queue, folder transfers, remote editing and drag and drop in both directions.
- Server metrics for SSH sessions: CPU, memory, disk, network, load and uptime.

### Security and data
- Vault sealed with XChaCha20-Poly1305; the key is wrapped with DPAPI and/or an Argon2id master password.
- Lock on demand (Ctrl+Shift+L) or after a period of inactivity.
- Encrypted `.platty` export and import that never deletes data or replaces trusted host keys.
- Option to hide IP addresses on screen.

### App
- A first-launch introduction with Sol.
- Search Everywhere (Ctrl+Shift+P), four monochrome themes and an About page.
