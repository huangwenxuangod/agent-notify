# Agent Notify Control Plane Design

Date: 2026-08-10

## Goal

Extend Agent Notify from a CLI and direct-hook notifier into a host control plane with a reliable first-run setup flow, unified support for all supported agents, and a macOS/Windows desktop application.

The existing hook path remains the data plane: agent hooks invoke the existing Go handlers, which normalize events and send notifications directly. The new control plane manages host configuration and exposes status; it is never required for notification delivery.

## Scope

Build:

- A Go Host Bridge listening only on `127.0.0.1:45174`.
- Token-protected local HTTP APIs for discovery, configuration, hook installation, logs, events, testing, and autostart.
- A local JSONL event journal at `~/.agent-notify/events.jsonl` with bounded rotation.
- Unified discovery and setup for Claude Code, Codex, WorkBuddy/CodeBuddy, Hermes Agent, OpenClaw, ZCode, Grok, Droid, and OpenCode.
- A Wails v2.13.0 desktop shell with a TypeScript frontend.
- A separate Go tray command using `github.com/energye/systray` v1.0.3.
- User-level macOS `launchd` and Windows Run-key autostart.
- Docker deployment retained as an optional local read-only console and event-query surface.

Do not build:

- A process or screen polling monitor.
- A cloud service, account system, or multi-device sync layer.
- An Electron or Tauri application.
- A second notification/event model.
- A requirement that Docker or the desktop app be running for hooks to notify.
- A Linux desktop tray in this delivery.

## Architecture

```text
Agent native Hook / Plugin
        |
        v
agent-notify hook handler
  |-- direct system/remote notification
  `-- fail-open append to ~/.agent-notify/events.jsonl
        |
        v
Host Bridge (Go, 127.0.0.1:45174)
  |-- hook/config/autostart operations
  |-- event and log reads
  `-- notification test
        |                    |
        v                    v
tray command             Wails v2 + TypeScript
常驻/自启动              setup/status/history UI
```

The Bridge is a reusable Go service. CLI, tray, Wails, and the optional Docker console call the same service methods. Agent hook handlers do not call the Bridge; they keep their direct path and only append the journal as a best-effort side effect.

## Host Bridge

The Bridge creates `~/.agent-notify/bridge.token` on first start with mode `0600`. Mutating HTTP endpoints require `Authorization: Bearer <token>`. Health is loopback-only and may be unauthenticated.

Public endpoints:

```text
GET  /api/health
GET  /api/agents
GET  /api/config
PUT  /api/config
GET  /api/events
GET  /api/logs
POST /api/setup/scan
POST /api/setup/install
POST /api/setup/uninstall
POST /api/notifications/test
GET  /api/autostart
PUT  /api/autostart
```

All host writes use the repository's existing ordered JSON, atomic-file, and file-lock helpers. Hook installation is incremental, preserves user hooks, is idempotent, and records absolute installed paths. A partial setup returns per-agent results rather than rolling back successful independent installations.

## Event Model and Journal

The normalized event names remain:

```text
permission_required
input_required
run_completed
run_failed
session_start
```

`session_start` remains a focus-window side effect and never sends a notification. Every non-start message may append one JSON object to the journal containing timestamp, agent, event, session ID, workspace, title, body, source app, and dispatch result. Journal append failures are logged only and never returned to the agent hook. The journal rotates at 5 MiB and retains one previous file.

## Agent Integrations

Existing Claude Code, Codex, ZCode, Grok, Droid, and OpenCode packages remain the source of truth and are registered through the existing `Integration` interface. WorkBuddy/CodeBuddy uses the CodeBuddy settings schema at `~/.codebuddy/settings.json` and `.codebuddy/settings.json`.

Hermes uses the official shell-hook surface in `~/.hermes/config.yaml` and respects Hermes consent and `~/.hermes/shell-hooks-allowlist.json`; the Bridge never bypasses that trust boundary. Hermes events are mapped only where an official event exists.

OpenClaw uses an official typed plugin hook. The plugin emits normalized hook payloads to the existing Go handler and does not own timers, watchers, sockets, or long-lived clients. Internal OpenClaw hooks are not used as a substitute for typed plugin lifecycle hooks.

Each adapter has its own parser/settings tests and fails open on malformed or unsupported input. Unsupported events are omitted from setup choices instead of being shown as configurable.

## Desktop Application

The desktop shell is Wails v2.13.0. The frontend is the official TypeScript template with strict type checking, plain TypeScript/DOM/CSS, and no React/Vue dependency. Wails bindings call shared Bridge service methods; business logic is not duplicated in the frontend.

The tray is a separate `agent-notify tray` command using `github.com/energye/systray` v1.0.3. It owns the native event loop and provides open-console, rescan, test-notification, autostart, and quit actions. The Wails window is launched on demand. Closing the window does not stop the Bridge or direct hooks.

macOS autostart uses `~/Library/LaunchAgents/com.agentnotify.tray.plist` with user-level `launchctl bootstrap/bootout`. Windows uses `HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run`. Windows packaging uses the WebView2 embedded bootstrapper strategy and displays an actionable message when installation is required.

## Delivery Phases

1. **Bridge and journal**: shared Bridge service, token, API, bounded JSONL journal, CLI entry points, and tests. Existing direct hooks remain usable after this phase.
2. **All agent setup**: complete WorkBuddy wiring and add Hermes/OpenClaw adapters, discovery, configuration defaults, install/uninstall, and trust-step reporting.
3. **Wails TypeScript UI**: setup wizard, agent matrix, notification configuration, events, logs, and test-notification flows.
4. **Tray and autostart**: macOS/Windows tray process, user-level startup registration, platform packaging, and CI matrix.
5. **Optional Docker console**: keep `deploy.sh`; add read-only Bridge status/history access and an event-copy path to PostgreSQL. Docker remains independent from hook delivery.

Each phase is independently usable and can be rolled back by stopping/removing only its new process or startup entry. No phase requires data migration or changes to existing hook files beyond the requested managed entries.

## Verification

- `go test ./...`
- TypeScript typecheck and production build.
- Bridge tests for unauthorized writes, token permissions, idempotent setup, partial failures, atomic writes, journal rotation, and concurrent appends.
- Fixture tests for every Agent parser and settings format, including preservation of unrelated user hooks.
- macOS manual test: tray appears after login, Wails window opens, window close leaves Bridge alive, direct hook notification works with tray stopped.
- Windows manual test: Run-key startup, WebView2 handling, tray actions, and direct hook notification with the desktop process stopped.
- Docker manual test: `./deploy.sh up`, `status`, `upgrade`, `uninstall`; no host config or token is deleted by uninstall.

## Risks and Rollback

The fragile assumption is that Hermes and OpenClaw retain the documented hook/plugin configuration contracts. Discovery performs format/version checks and reports unsupported installations without writing files. Their adapters can be disabled independently.

Wails v2 and the tray dependency are pinned. If either desktop build fails, release the CLI/Bridge and hook binaries independently; no notification data or hook configuration needs rollback. WebView2 absence affects only the Windows UI and does not affect the Bridge or hook handlers.

## Dependency and Credential Inventory

- Wails v2.13.0: desktop window and TypeScript asset bundling; no user credential required.
- energye/systray v1.0.3: native tray loop; no user credential required.
- Node.js/npm: TypeScript frontend build; no runtime service.
- Hermes CLI/config: user-installed Agent integration; existing Hermes consent/allowlist remains authoritative.
- OpenClaw CLI/plugin loader: user-installed Agent integration; no cloud credential is introduced by Agent Notify.
- Existing notification channels: existing user-managed webhook or account credentials remain in `~/.agent-notify/config.yaml` and are not copied into Docker.
