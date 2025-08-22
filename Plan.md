# P2P Chat App (Go) — Project spec & implementation plan (chat version)

Below is a rewritten, self-contained specification and step-by-step plan for the P2P chat app we discussed. It’s focused on practical decisions, testability, and a clear incremental path so you can start implementing with confidence. No code here — just the plan.

---

## 1 — Project goal & constraints (short)

* **Language:** Go.
* **Primary feature:** real-time text chat between two peers (no server required for data).
* **Connectivity:** P2P first, **OpenRelay (STUN+TURN)** as a guaranteed fallback.
* **UX:** single executable per platform (Windows + Linux). No extra installs for networking.
* **Binary:** keep small (no Chromium/Electron).
* **Testability:** modular packages with small interfaces so unit tests and fakes are easy.

---

## 2 — High-level architecture (how it works)

```
User A UI  <--- signaling (copy/paste or tiny server) --->  User B UI
   |                                                   |
   |          WebRTC DataChannel (encrypted, DTLS)     |
   +-----------------------> Peer <---------------------+
                     (pion/webrtc)
                        |
                        v
                  STUN / TURN (OpenRelay)
```

* Each app creates a WebRTC PeerConnection and a DataChannel (pion/webrtc).
* Signaling (offer/answer exchange) is out-of-band in v1 — copy/paste encoded SDP. Optional tiny signaling server later for UX.
* ICE flow: try direct -> STUN -> TURN. With TURN available the connection will work in restrictive networks.

---

## 3 — Packages & responsibilities (structure)

Design the repo so each package has a single responsibility. This improves testability and clarity.

**Top-level idea**

* `cmd/` — executables (headless CLI and later desktop UI).
* `pkg/` — core packages used by apps.
* `internal/` — concrete UI implementations and app-specific internals.
* `testutil/` — fakes and helpers for tests.

**Key packages (what each does)**

1. `pkg/webrtc`

   * Exposes a small `Peer` interface: create offer, set remote, send bytes, on-message callback, on-open, on-ice-state-change, close.
   * Internally uses pion/webrtc but hides its types behind the interface so tests can use a fake peer.

2. `pkg/signaling`

   * Encode/decode SDPs to a shareable compact string (base64url, optionally compressed).
   * Clipboard/QR helpers (desktop implementations live in `internal/`).

3. `pkg/protocol`

   * Message schemas and (de)serialization helpers (newline-delimited JSON on the datachannel).
   * Types: chat, presence, typing, control. Small, stable fields, versioning support.

4. `pkg/room`

   * Lightweight room/session manager (tokens, join/leave), in-memory initially.
   * Generates ephemeral tokens and validates joins.

5. `pkg/upnp` (optional)

   * Try to request a port mapping on the host router to improve direct-connect chances. Opt-in.

6. `pkg/turn`

   * Helpers for managing TURN credentials (if you later switch to REST-generated short-lived TURN creds). For v1 read static creds from config/env.

7. `pkg/ui`

   * UI interfaces (e.g., `UI.Run()`), with concrete Fyne/Gio/TUI implementations under `internal/`.

8. `pkg/testutil`

   * Fakes for `Peer`, deterministic clocks, token generators to enable deterministic unit and integration tests.

---

## 4 — Message protocol (practical)

* Use newline-delimited JSON messages (NDJSON) over the DataChannel.
* Keep fields tiny and stable:

  * `t` — type (`m`=message, `p`=presence, `tp`=typing, `ctrl`=control)
  * `id` — message id (UUID) for idempotency
  * `from` — display name
  * `ts` — timestamp (unix ms)
  * `text` — message body
* Limit message size (e.g., 4–8 KB) and validate on receive.

---

## 5 — Signaling & connection flow (v1: copy/paste)

1. **Host**: create offer → gather ICE → encode SDP → show/copy string or QR.
2. **Guest**: paste host offer → set remote → create answer → encode → copy back.
3. **Host**: set guest answer → DataChannel opens → exchange protocol messages.

* Show clear ICE state updates in UI (gathering, checking, connected, failed). Show whether a relay (TURN) candidate was chosen.

---

## 6 — ICE / TURN config (OpenRelay)

* Put OpenRelay STUN/TURN endpoints in the ICE servers list. Also include `stun:stun.l.google.com:19302` as an extra STUN.
* For TURN add both UDP and TCP/TLS endpoints (e.g., port 3478 and 443) so connections can succeed behind restrictive firewalls.
* In v1, store credentials in a config file or environment variables; in production prefer short-lived credentials if the TURN provider supports REST tokens.

---

## 7 — Security & privacy

* WebRTC DataChannels are encrypted (DTLS) by default; TURN relays ciphertext. Still:

  * Use join tokens to prevent random joins.
  * Limit message size, rate-limit messages.
  * If using a signaling server later, use HTTPS and short-lived tokens.
  * Sanitize any user-generated content rendered in UI.

---

## 8 — Testing strategy

**Unit tests**

* `pkg/protocol`: marshal/unmarshal, invalid payloads.
* `pkg/room`: token lifecycle, join/leave, expiry tests.
* `pkg/signaling`: encode/decode roundtrips.
* `pkg/webrtc`: unit tests should target logic using a fake `Peer` (from `testutil`) rather than real pion peers.

**Integration tests**

* **Headless integration**: run two in-process peers using a fake/in-memory signaling bridge to test message flow, ordering, reconnection logic.
* **Network integration**: real pion connections using OpenRelay or a self-hosted coturn instance; run manually or in CI if permitted.

**Manual E2E**

* Test across networks: same LAN, different ISPs, mobile hotspot, corporate VPN. Check TURN fallback.
