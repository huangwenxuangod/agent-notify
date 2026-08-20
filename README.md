<div align="center">

<img src="assist/brand/bell-only-128.png" alt="Agent Notify" width="90">

# Agent Notify

<p align="center"><b>Notifies you when your agent needs you</b>

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/hellolib/agent-notify.svg)](https://github.com/hellolib/agent-notify/releases)

<p align="center"><b>English</b> | <a href="README.zh-CN.md">简体中文</a></p>

</div>

## Overview

Agent Notify is a local notification layer for AI agents. It detects installed agents, installs their official hook or plugin integration, and sends a native notification or a configured bot message when a task needs attention, finishes, or fails.

It supports **Claude Code, Codex, Pi, WorkBuddy / CodeBuddy, Hermes Agent, OpenClaw, ZCode, Grok, Droid, and OpenCode**. Delivery channels include **system notifications, Feishu / Lark, WeChat, WeChat Work, DingTalk, Bark, ntfy, and Slack**.

<p align="center">
  <img src="assist/demo.gif" alt="Agent Notify demo" width="800">
</p>

## Quick Start

```bash
bunx agent-notify
```

This opens the CLI setup wizard and installs only the agents and channels you select. The Bun launcher downloads the matching Go Hook Runtime into `~/.agent-notify/` and executes it by absolute path.

Requirements: Bun `>=1.3.14` for the launcher and Go `>=1.25` only when building the desktop app from source.

## How It Works

```text
Agent event
  -> host Hook / plugin
  -> native system notification + configured bot channels
  -> Docker Bridge (best-effort event history for the desktop app)
```

The host runtime is the delivery path. Notifications continue to work when Docker or the desktop app is stopped. Docker is a local control plane, not a relay that can become a single point of failure.

## Desktop App and Docker

The desktop app is a source-built menu-bar/tray control panel for macOS and Windows. On launch it detects installed agents, installs missing user-level integrations, enables system notifications for connected agents, manages remote channels, configures personal WeChat iLink, shows event history, tests delivery, and controls login startup. Closing the window hides the app; the background bridge and notification path continue running until you choose Quit. Selecting **Open Agent Notify** from the macOS menu bar reactivates, restores, and shows the window.

```bash
./deploy.sh up        # start the Docker control plane on 127.0.0.1:45173
./deploy.sh desktop   # build/reinstall and show the desktop app
./deploy.sh status    # show container status
./deploy.sh logs      # follow control-plane logs
./deploy.sh restart
./deploy.sh down
./deploy.sh upgrade
./deploy.sh uninstall
```

`./deploy.sh up` is optional for the host notification path. Hooks send system and configured remote notifications directly; Docker provides the local control plane used for event history, retries, and desktop status. The app itself starts the Bun WeChat Bridge on `127.0.0.1:45176` when needed. `~/.agent-notify/config.yaml` remains the configuration authority. The macOS app is ad-hoc signed for local use and is not notarized for public distribution.

WorkBuddy caches hooks inside its `codebuddy --serve` process. Restart WorkBuddy after adding or updating its hooks.



## Features

### Supported Channels

| Channel | Description | Setup   |
|:--------|------|---------|
| 🖥️ System Notification | Native notifications on macOS, Linux, and Windows | Default |
| <img src="assist/logo/feishu.png" width="24" align="absmiddle"> Feishu / Lark | One-click QR-code binding; push via Feishu bot messages | QR scan |
| Personal WeChat (iLink) | Tencent-supported QR login through the built-in local Bun bridge | QR scan + bind |
| WeChat (generic) | Compatibility webhook channel | Webhook |
| <img src="assist/logo/qiyeweixin.png" width="24" align="absmiddle"> WeChat Work | Push notifications via a WeChat Work group bot webhook | Webhook |
| <img src="assist/logo/dingding.png" width="24" align="absmiddle"> DingTalk | Push notifications via a DingTalk group bot webhook | Webhook |
| <img src="assist/logo/bark.png" width="24" align="absmiddle"> Bark | Push to iOS devices via a Bark webhook URL | Webhook |
| <img src="assist/logo/ntfy.png" width="24" align="absmiddle"> ntfy | Push via ntfy.sh or self-hosted ntfy server | Topic |
| <img src="assist/logo/slack.png" width="24" align="absmiddle"> Slack | Push via Slack Incoming Webhook | Webhook |
| <img src="assist/logo/discord.png" width="24" align="absmiddle"> Discord | Push via Discord channel webhook | 🚧 Webhook |
| <img src="assist/logo/telegram.png" width="24" align="absmiddle"> Telegram | Push via Telegram Bot API | 🚧 Bot token |

### Agent Integrations

| Agent | Integration | Events |
|------|------|------|
| Claude Code | Native hooks | Permission, input, completion, failure |
| Codex | Official hooks | Permission and completion; run `/hooks` to trust it |
| WorkBuddy / CodeBuddy | CodeBuddy-compatible hooks | Permission, input, completion, failure; restart after changes |
| Hermes Agent | Gateway hook directory | Start, completion, failure, approval |
| OpenClaw | ESM extension | Completion, failure, approval; enable the extension in OpenClaw |
| ZCode | Native hooks | Permission, completion, failure |
| Grok | Native hooks | Permission/input classification, completion, failure |
| Droid | Native hooks | Permission/input classification and completion |
| OpenCode | JavaScript plugin | Permission, input, completion, failure |
| Pi | Official TypeScript extension | Completion and interruption; installed globally with no per-project approval |

Notes:

- Claude Code subscribes via hooks in `~/.claude/settings.json`: `PermissionRequest`, `PermissionDenied`, `Notification`, `Stop`, `StopFailure`, `PostToolUseFailure`, and `SessionStart`. Permission denials and API/turn failures are normalized as actionable notifications with the original reason.
- Codex subscribes via `~/.codex/hooks.json`: `PermissionRequest` and `Stop` (mapped to `permission_required` / `run_completed`), plus `SessionStart`. Codex Desktop UI turns are additionally read from `~/.codex/sessions/*.jsonl`, including structured task errors such as rate limits; the CLI hook API itself has no dedicated failure event. Internal control payloads such as `{"exclude":[]}` and suggestion metadata are suppressed, while normal text and user-requested JSON results are retained.
- ZCode subscribes via `~/.zcode/cli/config.json`: `SessionStart`, `PermissionRequest`, `PostToolUseFailure`, and `Stop`, mapped to `permission_required`, `run_failed`, and `run_completed`. ZCode has no `Notification` event (so no `input_required`), and its hook schema is strict — an unknown event name will cause the whole hooks config to be silently dropped.
- Grok subscribes via `~/.grok/hooks/agent-notify.json`: `SessionStart`, `Notification`, `Stop`, `StopFailure`, and `PostToolUseFailure`. There is no dedicated `PermissionRequest` event; `Notification`s with permission/approval semantics map to `permission_required` (marked *), others map to `input_required`. `StopFailure` / `PostToolUseFailure` map to `run_failed`.
- Droid subscribes via `~/.factory/hooks.json`: `SessionStart`, `Notification`, `Stop`, mapped to `session_start` / `permission_required`|`input_required` / `run_completed`. Droid has no failure event, so `run_failed` is not supported. `session_start` is only used for click-to-focus window capture, not as a notification event.
- OpenCode uses a JS plugin instead of native hooks: the plugin is written to `~/.agent-notify/opencode-plugin.js` (binary path baked into JS), and its path is registered in `~/.config/opencode/opencode.json` (user) or `./opencode.json` (project) `plugin` array. The plugin subscribes to `session.created`→`session_start`, `permission.asked`→`permission_required`, `session.status`(idle)→`input_required`, `session.idle`→`run_completed`, `session.error`→`run_failed`.
- WorkBuddy uses the CodeBuddy settings schema at `~/.codebuddy/settings.json`.
- Hermes writes `~/.hermes/hooks/agent-notify/HOOK.yaml` and `handler.py`; this is its Gateway event surface.
- OpenClaw writes a self-contained extension to `~/.openclaw/extensions/agent-notify/`. OpenClaw still controls whether that extension is enabled and granted conversation access.
- Pi writes a native TypeScript extension to `~/.pi/agent/extensions/agent-notify.ts` (or `.pi/extensions/agent-notify.ts` for project scope). It coalesces `agent_end` results for five seconds so automatic retries produce only the final notification, and reports `session_shutdown(reason=quit)` only when a run is still active.
- **`SessionStart` does not produce a notification.** It is subscribed on every agent solely to capture the terminal window at session start, which powers Linux window-level [Click-to-Focus](#click-to-focus). On macOS/Windows the SessionStart hook is a no-op.

### Supported Platforms

| Platform | Hook Runtime | Desktop control panel |
|:---:|:---:|:---:|
| macOS amd64 / arm64 | ✅ | ✅ Source-built menu-bar app |
| Linux amd64 / arm64 | ✅ | — |
| Windows amd64 / arm64 | ✅ | ✅ Source-built tray app |

### Click-to-Focus

System notifications are clickable — clicking one brings you back to the terminal / window where the agent is running. Behavior differs by platform:

- **macOS** — App-level by default (activates the agent's terminal/IDE app). For window-level focus (return to the exact window even when several are open), set `AGENT_NOTIFY_FOCUS_PRECISION=window` in your login shell environment (e.g. `~/.zshrc`); this uses a bundled helper and requires Accessibility permission. Unset stays app-level.
- **Linux (X11)** — Window-level. The exact terminal window is captured at session start (via the `SessionStart` hook) and re-focused on click, so it distinguishes sibling windows of single-process terminals (deepin-terminal, GNOME Terminal, etc.). Native Wayland windows can't be targeted.
- **Windows** — Returns to the terminal window via a bundled helper.

> **`AGENT_NOTIFY_FOCUS_PRECISION`** accepts `window` (window-level) or `app` (app-level — the default). Values are case-insensitive and whitespace-trimmed; anything unset or unrecognized falls back to `app`. This variable **only affects macOS** — Linux is always window-level, and Windows uses its own helper.

Click-to-focus is enabled by default for the System channel; the target app/window is detected automatically from the hook's environment and process tree.




## Configuration

On first run, the launcher downloads the platform-specific binary matching the current Bun package version from GitHub Releases and installs it to:

- macOS / Linux: `~/.agent-notify/agent-notify`
- Windows: `~/.agent-notify/agent-notify.exe`

On every subsequent run it checks the local binary version: it downloads if missing, updates if outdated, and otherwise runs directly. The launcher never persistently modifies `PATH` — it always executes via an absolute path.

### Windows deployment

Docker control-plane and the native desktop runtime are separate. On Windows, run PowerShell from the repository root:

```powershell
.\deploy.ps1 up
.\deploy.ps1 desktop
```

The script builds `agent-notify.exe` and `Agent Notify.exe`, installs the user-level startup entry under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, and starts the desktop app. The app stays in the tray by default; use the tray menu to open or quit it. `down`, `restart`, `status`, `logs`, `upgrade`, and `uninstall` mirror `deploy.sh`.

Cross-compiling an EXE on macOS only proves that the Windows build is valid. Toasts, tray behavior, startup, terminal focus, and agent hooks require a Windows 10/11 machine for final acceptance.

### Personal WeChat bot (iLink) versus webhook

Personal WeChat is a separate `wechat-ilink` channel. It is not the generic `wechat` webhook channel and does not require an OpenClaw installation. The desktop app starts a local Bun bridge on `127.0.0.1:45176`; the bridge performs Tencent iLink QR login, stores the bot session locally, maintains long-polling, binds the recipient, and forwards Agent Notify messages.

Setup is intentionally short:

1. Open the desktop app and select **Personal WeChat**.
2. Click **Connect** and scan the displayed Tencent QR code.
3. Send one ordinary message from the target personal WeChat account to the bot. This binds the recipient and refreshes the conversation context.
4. Use **Send test** or trigger a real Agent event.

The bridge persists `get_updates_buf` and the latest `context_token`, calls Tencent's `notifystart`/`notifystop` lifecycle endpoints, and performs one no-context retry when the server reports `ret=-2` / `prepare failed`. This improves recovery after idle periods but cannot guarantee permanent proactive delivery: Tencent controls token, context, and service-side expiry. If the context becomes stale, send another message from WeChat before testing again. Group-chat support is not advertised.

Do not paste an iLink token into the generic webhook field. The iLink state is stored in `~/.agent-notify/wechat-ilink.json`; the shared channel switch is stored in `~/.agent-notify/config.yaml`.

> **Note**: Codex integrates through the official hooks system in `~/.codex/hooks.json` and currently subscribes only to `PermissionRequest` and `Stop`. After first install, run `/hooks` inside Codex to complete the trust review.
>
> **Grok**: Writes `~/.grok/hooks/agent-notify.json`. Global hooks are always trusted; project hooks (`.grok/hooks/`) require `/hooks-trust` or `--trust`. After install, run `/hooks` (or `Ctrl+L`) inside Grok to confirm they loaded.


> You don't need to edit config files by hand — this section is for reference only.

Agent Notify's own config lives at `~/.agent-notify/config.yaml`. **New installs start with all agents and channels disabled** — run `bunx agent-notify` (setup wizard) once to enable the agents and channels you want. This avoids showing unconfigured agents as ready in view/doctor after a partial setup. Existing config files are left unchanged.

Agent integration config locations:

- Claude Code: `~/.claude/settings.json` (writes hooks → command `agent-notify handle-claude-hook`)
- Codex: `~/.codex/hooks.json` (writes hooks → command `agent-notify handle-codex-hook`; run `/hooks` inside Codex to complete trust)
- ZCode: `~/.zcode/cli/config.json` (writes `hooks.events.<Event>` + `hooks.enabled` → command `agent-notify handle-zcode-hook`; restart ZCode for the config to take effect)
- Grok: `~/.grok/hooks/agent-notify.json` (writes hooks → command `agent-notify handle-grok-hook`; project scope uses `.grok/hooks/agent-notify.json`)
- Droid: `~/.factory/hooks.json` (writes hooks → command `agent-notify handle-droid-hook`; project scope uses `.factory/hooks.json`)
- OpenCode: `~/.config/opencode/opencode.json` (writes `plugin` array → `~/.agent-notify/opencode-plugin.js`, command `agent-notify handle-opencode-hook`; project scope uses `./opencode.json`)
- WorkBuddy / CodeBuddy: `~/.codebuddy/settings.json` (writes `handle-workbuddy-hook`)
- Hermes Agent: `~/.hermes/hooks/agent-notify/` (writes a Gateway hook manifest and handler)
- OpenClaw: `~/.openclaw/extensions/agent-notify/` (writes an ESM extension)
- Pi: `~/.pi/agent/extensions/agent-notify.ts` (writes an auto-discovered TypeScript extension; project scope uses `.pi/extensions/agent-notify.ts`)

Remote channels are configured once under `remote` in `~/.agent-notify/config.yaml` and shared by enabled agents. Agent-level event choices still decide which events are sent.

### WeChat Work Bot Binding Tip

1. **Create a single-person notification group**: start a group chat in WeChat Work (pull in a few colleagues). After it's created, **do not post anything**, then remove the others — the group becomes your personal notification channel.
2. **Add a bot**: "Group Settings" → "Message Push" → "Add" → "Custom Message Push", name it and save.
3. **Get the webhook URL**: copy the generated URL, which looks like `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`.
4. **Bind it**: run `bunx agent-notify`, enable the WeChat Work channel in the setup wizard, and paste the webhook URL.
> Older WeChat Work versions: "Group Settings" → "Group Bots" → "Add Bot" → "New Bot", name it and save.


<p align="center">
  <img src="assist/workflow.png" alt="Workflow diagram" />
</p>

## Screenshots

| | |
|:---:|:---:|
| <img src="assist/launch-setting.png" alt="Setup" width="75%"> | <img src="assist/feishu-bind.png" alt="Feishu binding" width="75%"> |
| **Setup** | **Feishu Binding** |
| <img src="assist/feishu-notify-phone.png" alt="Feishu notification" width="55%"> | <img src="assist/wecom-notify.jpg" alt="WeChat Work notification" width="55%"> |
| **Feishu Notification** | **WeChat Work Notification** |
| <img src="assist/system-notify.png" alt="System notification" width="55%"> | |
| **System Notification** | |


## Acknowledgments

Thanks for the support and feedback from the friends at [LINUX DO](https://linux.do/).
