# P2P Chat App (Go)

A lightweight, end-to-end encrypted **peer-to-peer (P2P) chat app** written in Go, powered by [pion/webrtc](https://github.com/pion/webrtc).

This project explores building a **simple, reliable, and private chat app** that doesn’t rely on a central server for data. The goal is to keep binaries small, the codebase modular, and the user experience simple — **just run the app and connect**.

> ⚠ **Status**: Very early development. Features, APIs, and structure will change frequently.  
> More details, docs, and examples will be added as the project matures.

---

## **Vision**

- **Direct, private connections**  
  Connect two users directly over the internet, no middleman storing your messages.

- **Fallback when needed**  
  Use STUN and TURN (via [OpenRelay](https://openrelay.metered.ca/)) so that even restrictive networks can connect.

- **Modular & testable**  
  Core logic is designed in small, testable packages for easy unit and integration testing.

- **Small binaries, zero setup**  
  One executable per platform. No Electron, no heavy dependencies.

---

## **Planned Features**

- Text messaging over encrypted WebRTC DataChannels  
- Simple signaling via copy/paste or QR code  
- Automatic NAT traversal with STUN/TURN fallback  
- Clean interfaces for building CLI, TUI, or desktop UI clients  
- Optional lightweight signaling server for a better UX  
- Built-in fakes for deterministic testing and CI

---

## **Repository Structure (planned)**

```

cmd/           # Executables (CLI, GUI in the future)
pkg/           # Core reusable packages
internal/      # App-specific code (UI, helpers)
testutil/      # Testing utilities and fakes

```

---

## **License**

This project is licensed under the **MIT License** – see the [LICENSE](LICENSE) file for details.

---

## **Disclaimer**

This project is experimental and under active development.  
Expect breaking changes until the first stable release.