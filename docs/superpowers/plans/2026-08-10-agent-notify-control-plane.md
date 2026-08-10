# Agent Notify Control Plane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fail-open Go Host Bridge, complete all Agent integrations, and ship a Wails v2.13.0 + TypeScript macOS/Windows tray application with user-level autostart.

**Architecture:** Existing Agent hooks remain the data plane and continue calling `agenthooks.Dispatch` directly. A shared Go Bridge service owns host configuration, discovery, installation, event history, and autostart APIs on `127.0.0.1:45174`; a tray process keeps the Bridge alive and opens a Wails desktop window whose TypeScript UI calls the Bridge.

**Tech Stack:** Go 1.25+, standard library `net/http`/`encoding/json`, existing config and hook helpers, Wails v2.13.0, TypeScript frontend, `github.com/energye/systray` v1.0.3, macOS `launchd`, Windows HKCU Run key.

## Global Constraints

- Hook notification delivery must remain direct and must not depend on Bridge, tray, Wails, Docker, or PostgreSQL.
- Bridge binds only to `127.0.0.1:45174`; mutating requests require `~/.agent-notify/bridge.token` with mode `0600`.
- Reuse existing `notify.Message`, `agenthooks.Dispatch`, `agentintegrations.Integration`, ordered JSON, atomic-file, file-lock, config, and state helpers.
- Preserve unrelated user hooks; installation and removal are incremental and idempotent.
- TypeScript is strict and uses plain DOM/CSS; do not add React, Vue, Electron, or Tauri.
- Hermes consent/allowlist and OpenClaw plugin trust remain authoritative; never bypass them.
- macOS and Windows builds run on native CI runners because the tray dependency uses CGO.
- Every non-trivial change follows TDD: failing test, red test run, minimal implementation, green test run.

---

### Task 1: Shared Event Journal

**Files:**
- Create: `internal/state/events.go`
- Create: `internal/state/events_test.go`
- Modify: `internal/agenthooks/dispatch.go`
- Modify: each existing hook handler in `internal/*hooks/handler.go`

**Interfaces:**
- Produce `type EventRecord struct { ID string; Timestamp time.Time; Message notify.Message; Result string }`.
- Produce `NewEventJournal(path string, maxBytes int64) *EventJournal`.
- Produce `func (j *EventJournal) Append(record EventRecord) error` and `func (j *EventJournal) List(limit int) ([]EventRecord, error)`.
- Append must use an exclusive file lock, JSONL encoding, `0600` file mode, and rotate at 5 MiB while retaining one `.1` file.

- [ ] Write failing tests for append/list, concurrent appends, `0600` permissions, 5 MiB rotation, and malformed-line skipping.
- [ ] Run `go test ./internal/state -run EventJournal -v` and confirm the new symbols are undefined.
- [ ] Implement the journal with existing `internal/common/filelock.go` and atomic-file helpers; use `uuid.NewString()` for record IDs and ignore journal errors in hook dispatch after logging.
- [ ] Run `go test ./internal/state ./internal/agenthooks -v` and confirm PASS.
- [ ] Commit `feat: add fail-open event journal`.

### Task 2: Bridge Core and Authentication

**Files:**
- Create: `internal/bridge/service.go`
- Create: `internal/bridge/auth.go`
- Create: `internal/bridge/http.go`
- Create: `internal/bridge/service_test.go`
- Create: `internal/bridge/http_test.go`
- Modify: `internal/config/config.go` to add explicit bridge token and listen-address helpers

**Interfaces:**
- Produce `type Service struct` with `ScanAgents`, `InstallAgents`, `UninstallAgents`, `GetConfig`, `SaveConfig`, `ListEvents`, `ListLogs`, `TestNotification`, `AutostartStatus`, and `SetAutostart` methods.
- Produce `func NewHTTPHandler(svc *Service, token []byte) http.Handler`.
- Produce `func EnsureToken(path string) ([]byte, error)`.
- `GET /api/health` returns `{"status":"ok","version":...}` without auth; all other routes require the bearer token.

- [ ] Write tests for token creation/mode, missing/invalid bearer token returning `401`, health without auth, JSON content types, and unknown route returning `404`.
- [ ] Run `go test ./internal/bridge -v` and confirm red.
- [ ] Implement route decoding with `net/http`, `json.Decoder`, bounded request bodies (1 MiB), and explicit `400/401/404/409/500` responses; do not expose arbitrary filesystem paths or command execution.
- [ ] Run the bridge tests and `go vet ./internal/bridge`.
- [ ] Commit `feat: add authenticated host bridge service`.

### Task 3: Bridge CLI and Existing Agent Setup

**Files:**
- Create: `internal/cli/bridge.go`
- Create: `internal/cli/bridge_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/actions.go`
- Modify: `internal/cli/clean_targets.go`
- Modify: `internal/app/setup/service.go`
- Modify: `internal/app/doctor/service.go`
- Modify: `internal/config/config.go`

**Interfaces:**
- Add hidden-free commands `agent-notify bridge`, `agent-notify tray`, and `agent-notify setup --json`.
- Register WorkBuddy in setup and doctor service option lists alongside the six existing integrations.
- Extend `AgentConfig`, `NotifyConfig`, defaults, `Load`, `All`, cleanup, and sender selection for Hermes and OpenClaw IDs.

- [ ] Add command tests that invoke `bridge --port 45174` with an isolated home and verify token creation and graceful context cancellation.
- [ ] Run the focused CLI tests and confirm red for missing commands/fields.
- [ ] Implement the commands using the Bridge service; use `signal.NotifyContext`, `http.Server.Shutdown`, and no prompts in hook handlers.
- [ ] Run `go test ./internal/cli ./internal/app/setup ./internal/app/doctor ./internal/config`.
- [ ] Commit `feat: expose bridge and unified setup commands`.

### Task 4: Hermes Agent Adapter

**Files:**
- Create: `internal/hermeshooks/event.go`
- Create: `internal/hermeshooks/settings.go`
- Create: `internal/hermeshooks/handler.go`
- Create: `internal/hermeshooks/event_test.go`
- Create: `internal/hermeshooks/settings_test.go`
- Create: `internal/agentintegrations/hermes.go`
- Modify: `internal/agenthooks/dispatch.go`
- Modify: `internal/config/config.go`
- Modify: `internal/cli/root.go`

**Interfaces:**
- Use Agent ID `hermes` and command `handle-hermes-hook`.
- Read/write only the documented `hooks:` entries in `~/.hermes/config.yaml`; preserve all other YAML nodes.
- Map official events: `agent:end` to `run_completed`, failed tool/agent outcomes to `run_failed`, approval events only when present in the payload, and lifecycle start to `session_start`.

- [ ] Add fixtures for each supported Hermes event, malformed input, and an existing config containing unrelated hooks and an allowlist.
- [ ] Run `go test ./internal/hermeshooks ./internal/agentintegrations -run Hermes -v` and confirm red.
- [ ] Implement YAML edits with `yaml.v3`, exact command markers, and an installation result that states Hermes consent/allowlist confirmation is still required.
- [ ] Run focused tests and `go test ./...`.
- [ ] Commit `feat: add Hermes hook integration`.

### Task 5: OpenClaw Typed Plugin Adapter

**Files:**
- Create: `internal/openclawhooks/event.go`
- Create: `internal/openclawhooks/plugin.go`
- Create: `internal/openclawhooks/handler.go`
- Create: `internal/openclawhooks/event_test.go`
- Create: `internal/agentintegrations/openclaw.go`
- Modify: `internal/agenthooks/dispatch.go`
- Modify: `internal/config/config.go`
- Modify: `internal/cli/root.go`

**Interfaces:**
- Use Agent ID `openclaw` and command `handle-openclaw-hook`.
- Install an official plugin package under `~/.openclaw/extensions/agent-notify` and register documented typed hooks only.
- Map `gateway_start`/`gateway_stop`, agent completion, model/tool failures, and lifecycle events to normalized messages; never use internal hooks for a long-lived watcher.

- [ ] Add fixtures for plugin payloads, plugin manifest generation, idempotent install, uninstall preserving other extensions, and unsupported event rejection.
- [ ] Run focused tests and confirm red.
- [ ] Implement the minimal plugin emitter that writes one JSON payload to the existing Go handler, with no timers, sockets, or network clients.
- [ ] Run `go test ./internal/openclawhooks ./internal/agentintegrations ./...`.
- [ ] Commit `feat: add OpenClaw typed plugin integration`.

### Task 6: Wails TypeScript Desktop UI

**Files:**
- Create: `cmd/agent-notify-desktop/main.go`
- Create: `desktop/frontend/index.html`
- Create: `desktop/frontend/src/main.ts`
- Create: `desktop/frontend/src/api.ts`
- Create: `desktop/frontend/src/styles.css`
- Create: `desktop/wails.json`
- Create: `desktop/package.json`
- Create: `desktop/tsconfig.json`
- Create: `desktop/frontend/src/api.test.ts`
- Modify: `go.mod` and `go.sum`

**Interfaces:**
- `api.ts` exposes typed functions for `/api/health`, `/api/agents`, `/api/config`, `/api/events`, `/api/logs`, setup, test notification, and autostart.
- The Go desktop app binds `OpenBridge`, `Scan`, `Install`, `Uninstall`, `TestNotification`, and `SetAutostart`; bindings call the shared Bridge service, not duplicate logic.

- [ ] Add TypeScript tests for typed API decoding, `401` handling, and partial setup results.
- [ ] Run `cd desktop && npm ci && npm run typecheck && npm test` and confirm the new tests fail before implementation.
- [ ] Implement the Wails v2.13.0 TypeScript template with strict compiler settings, compact setup wizard, agent matrix, event/channel controls, history, logs, and test action.
- [ ] Run `wails doctor`, `wails build`, and the TypeScript tests on macOS; verify the generated bindings compile.
- [ ] Commit `feat: add Wails TypeScript desktop console`.

### Task 7: Tray and Platform Autostart

**Files:**
- Create: `internal/tray/tray.go`
- Create: `internal/tray/tray_test.go`
- Create: `internal/autostart/autostart.go`
- Create: `internal/autostart/autostart_darwin.go`
- Create: `internal/autostart/autostart_windows.go`
- Create: `internal/autostart/autostart_test.go`
- Modify: `internal/cli/bridge.go`
- Modify: `Makefile`
- Modify: `.github/workflows/release.yml`

**Interfaces:**
- `type Manager interface { Status() (Status, error); Enable() error; Disable() error }` with platform implementations.
- macOS writes the exact plist label `com.agentnotify.tray`; Windows writes only HKCU Run value `AgentNotifyTray`.
- Tray menu actions invoke Bridge health, open the Wails binary, run a scan, send a test, and quit.

- [ ] Add platform-neutral tests for command construction and state reporting; add macOS plist and Windows registry tests behind build tags.
- [ ] Run platform-specific tests on native runners and confirm red.
- [ ] Implement `launchctl bootstrap/bootout` and HKCU registry operations without admin privileges; quote executable paths and use hidden Windows process creation.
- [ ] Implement `energye/systray v1.0.3` with one tray loop and explicit cleanup.
- [ ] Run `go test ./...`, native packaging builds, and manual startup/quit checks.
- [ ] Commit `feat: add tray and user autostart`.

### Task 8: Optional Docker Console and Release Verification

**Files:**
- Modify: `deploy/compose.yaml`
- Modify: `deploy/www/index.html`
- Modify: `deploy.sh`
- Modify: `Makefile`
- Modify: `README.zh-CN.md`
- Modify: `README.md`
- Create: `deploy/www/api/health`

**Interfaces:**
- Keep `./deploy.sh up|down|restart|status|logs|upgrade|uninstall` unchanged.
- Docker serves read-only Bridge status/history and optionally imports journal records into PostgreSQL; it never mounts the whole home directory or writes host config.

- [ ] Add a Compose smoke test that checks the uncommon host port `45173`, health response, and preserved host state after uninstall.
- [ ] Run `./deploy.sh up`, `status`, `upgrade`, `logs` (bounded tail), `down`, and `uninstall`.
- [ ] Implement `GET /api/health` and `GET /api/events` as the Docker read-only surface, mount only the journal directory read-only, and keep PostgreSQL disabled until a tested importer exists; do not add a database dependency to the hook or Bridge binaries.
- [ ] Run `go test ./...`, `make build`, `cd desktop && npm ci && npm test`, `git diff --check`, and the Compose smoke test.
- [ ] Commit `feat: document and verify desktop deployment`.

## Final Acceptance

- A fresh user can launch the tray, open the Wails TypeScript window, scan all nine Agent integrations, select supported events, install hooks, configure a channel, and receive a test notification without editing JSON/YAML by hand.
- Existing user hooks remain byte-for-byte semantically intact except for managed Agent Notify entries.
- Stopping Bridge, tray, Wails, Docker, or PostgreSQL does not prevent a configured Agent hook from directly sending notifications.
- macOS and Windows user-level autostart can be enabled, inspected, disabled, and removed without administrator privileges.
- Malformed Agent payloads, missing optional CLIs, missing WebView2, denied Hermes consent, and unsupported OpenClaw events produce actionable diagnostics rather than breaking the Agent.
