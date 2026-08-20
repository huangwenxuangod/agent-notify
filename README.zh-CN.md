<div align="center">

<img src="assist/brand/bell-only-128.png" alt="Agent Notify" width="90">

# Agent Notify

<p align="center"><b>在 Agent 需要你时通知你</b>

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/hellolib/agent-notify.svg)](https://github.com/hellolib/agent-notify/releases)

<p align="center"><a href="README.md">English</a> | <b>简体中文</b></p>

</div>

## 项目简介

Agent Notify 是运行在本机的 AI Agent 通知层。它会发现已安装的 Agent，写入官方 Hook 或插件接入，并在任务需要处理、完成或失败时发送系统通知和已配置的机器人消息。

支持 **Claude Code、Codex、WorkBuddy / CodeBuddy、Hermes Agent、OpenClaw、ZCode、Grok、Droid、OpenCode**。通知渠道支持 **系统通知、飞书、微信、企业微信、钉钉、Bark、ntfy、Slack**。

<p align="center">
  <img src="assist/demo.gif" alt="Agent Notify 演示" width="800">
</p>

## 快速开始

```bash
bunx agent-notify
```

这会启动 CLI 配置向导，只接入你选择的 Agent 和通知渠道。Bun 启动器会把对应平台的 Go Hook Runtime 下载到 `~/.agent-notify/`，并始终通过绝对路径运行。

要求：启动器需要 Bun `>=1.3.14`；只有从源码构建桌面端时才需要 Go `>=1.25`。

## 工作方式

```text
Agent 事件
  -> 本机 Hook / 插件
  -> 系统通知 + 已配置机器人通道
  -> Docker Bridge（桌面端事件历史，尽力写入）
```

本机 Hook 是通知数据面。Docker 或桌面端退出时，通知仍会直接发送；Docker 只是本机控制面，不是通知转发的单点依赖。

## 桌面端与 Docker

桌面端是 macOS 菜单栏 / Windows 系统托盘常驻控制台：启动后会自动发现 Agent、接入缺失的用户级 Hook、为已连接 Agent 启用系统通知，并用于配置远程通道、配置个人微信 iLink、查看事件历史、测试推送和管理开机启动。关闭窗口只会隐藏应用；后台 Bridge 和通知链路会继续运行，直到从托盘选择退出。在 macOS 菜单栏选择「打开 Agent Notify」会重新激活、恢复并显示窗口。

```bash
./deploy.sh up        # 启动 Docker 控制面，端口为 127.0.0.1:45173
./deploy.sh desktop   # 构建、更新并显示桌面端
./deploy.sh status    # 查看容器状态
./deploy.sh logs      # 查看控制面日志
./deploy.sh restart
./deploy.sh down
./deploy.sh upgrade
./deploy.sh uninstall
```

`./deploy.sh up` 不是本机通知链路的硬依赖。Hook 会直接发送系统通知和已配置的远程通知；Docker 主要提供事件历史、重试和桌面状态所需的本地控制面。桌面端会按需在 `127.0.0.1:45176` 启动 Bun 微信 Bridge。`~/.agent-notify/config.yaml` 是配置源。macOS 桌面端仅做 ad-hoc 签名，未做公开分发公证。

WorkBuddy 会在 `codebuddy --serve` 进程中缓存 Hook 配置，安装或更新后需要重启 WorkBuddy。


## 功能特性
### 支持的通知渠道
|   通知渠道   | 说明 | 绑定方式   |
|:--------|------|--------|
| 🖥️ 系统通知 | 支持 macOS、Linux、Windows 系统通知 | 原生支持   |
| <img src="assist/logo/feishu.png" width="24" align="absmiddle"> 飞书   | 支持一键扫码绑定、支持飞书机器人消息推送 | 二维码扫描  |
| 个人微信（iLink） | 腾讯支持的个人微信机器人，内置 Bun Bridge 扫码登录 | 扫码 + 绑定 |
| 微信（通用） | 兼容用 Webhook 通道 | Webhook |
| <img src="assist/logo/qiyeweixin.png" width="24" align="absmiddle"> 企业微信  | 支持通过企业微信群机器人 Webhook 推送通知消息 | Webhook |
| <img src="assist/logo/dingding.png" width="24" align="absmiddle"> 钉钉  | 支持通过钉钉群机器人 Webhook 推送通知消息 | Webhook |
| <img src="assist/logo/bark.png" width="24" align="absmiddle"> Bark  | 支持通过 Bark Webhook URL 推送到 iOS 设备 | Webhook |
| <img src="assist/logo/ntfy.png" width="24" align="absmiddle"> ntfy  | 通过 ntfy.sh 或自托管 ntfy 服务推送 | Topic |
| <img src="assist/logo/slack.png" width="24" align="absmiddle"> Slack | 通过 Slack Incoming Webhook 推送 | Webhook |
| <img src="assist/logo/discord.png" width="24" align="absmiddle"> Discord | 通过 Discord 频道 Webhook 推送 | 🚧 Webhook |
| <img src="assist/logo/telegram.png" width="24" align="absmiddle"> Telegram | 通过 Telegram Bot API 推送 | 🚧 Bot token |

### Agent 接入

| Agent | 接入方式 | 事件能力 |
|------|------|------|
| Claude Code | 原生 Hook | 授权、等待输入、完成、失败 |
| Codex | 官方 Hook | 授权、完成；需在 Codex 内运行 `/hooks` trust |
| WorkBuddy / CodeBuddy | 兼容 CodeBuddy Hook | 授权、等待输入、完成、失败；改动后重启 |
| Hermes Agent | Gateway Hook 目录 | 开始、完成、失败、授权 |
| OpenClaw | ESM 扩展 | 完成、失败、授权；需在 OpenClaw 中启用扩展 |
| ZCode | 原生 Hook | 授权、完成、失败 |
| Grok | 原生 Hook | 授权/输入分类、完成、失败 |
| Droid | 原生 Hook | 授权/输入分类、完成 |
| OpenCode | JavaScript 插件 | 授权、等待输入、完成、失败 |
| Pi | 官方 TypeScript 扩展 | 完成、中断；全局安装，无需逐项目审核 |

说明：

- Claude Code 通过 `~/.claude/settings.json` 的 hooks 订阅：`PermissionRequest`、`Notification`、`Stop`、`PostToolUseFailure`、`SessionStart`。
- Codex 通过 `~/.codex/hooks.json` 订阅 `PermissionRequest`、`Stop`（映射到 `permission_required` / `run_completed`）以及 `SessionStart`。`input_required` 与 `run_failed` Codex 目前没有对应 hook，因此暂不支持。Codex 内部控制 payload（例如 `{"exclude":[]}`、建议元数据）会被静默过滤；正常文本和用户主动要求返回的 JSON 会保留通知。
- ZCode 通过 `~/.zcode/cli/config.json` 订阅 `SessionStart`、`PermissionRequest`、`PostToolUseFailure`、`Stop`，映射到 `permission_required`、`run_failed`、`run_completed`。ZCode 没有 `Notification` 事件（因此不支持 `input_required`），且其 hook 配置格式较为严格——无法识别的事件名称会导致整个 hooks 配置被静默丢弃。
- Grok 通过 `~/.grok/hooks/agent-notify.json` 订阅 `SessionStart`、`Notification`、`Stop`、`StopFailure`、`PostToolUseFailure`。Grok 没有独立的 `PermissionRequest` 事件，带 permission/approval 语义的 `Notification` 会映射为 `permission_required`（表中 *）；其它通知映射为 `input_required`。`StopFailure` / `PostToolUseFailure` 映射为 `run_failed`。
- Droid 通过 `~/.factory/hooks.json` 订阅 `SessionStart`、`Notification`、`Stop`，映射为 `session_start` / `permission_required`|`input_required` / `run_completed`。Droid 无失败事件，故不支持 `run_failed`。`session_start` 仅用于点击聚焦的窗口捕获，不作为通知事件。
- OpenCode 使用 JS 插件而非原生 hooks：插件写入 `~/.agent-notify/opencode-plugin.js`（二进制路径烘焙进 JS），路径注册到 `~/.config/opencode/opencode.json`（user）或 `./opencode.json`（project）的 `plugin` 数组。插件订阅 `session.created`→`session_start`、`permission.asked`→`permission_required`、`session.status`(idle)→`input_required`、`session.idle`→`run_completed`、`session.error`→`run_failed`。
- WorkBuddy 使用 `~/.codebuddy/settings.json` 中与 CodeBuddy 一致的 Hook 配置格式。
- Hermes 会写入 `~/.hermes/hooks/agent-notify/HOOK.yaml` 与 `handler.py`，这是它的 Gateway 事件入口。
- OpenClaw 会写入 `~/.openclaw/extensions/agent-notify/` 自包含扩展；是否启用扩展、授予会话权限仍由 OpenClaw 自己控制。
- **`SessionStart` 不产生任何通知。** 它在所有 agent 上被订阅，仅用于在会话启动时捕获终端窗口，为 Linux 的窗口级点击聚焦提供支持（见下方「点击聚焦」一节）；在 macOS/Windows 上该 hook 为空操作。

### 支持的平台

| 平台 | Hook Runtime | 桌面控制台 |
|:---:|:---:|:---:|
| macOS amd64 / arm64 | ✅ | ✅ 源码构建菜单栏应用 |
| Linux amd64 / arm64 | ✅ | — |
| Windows amd64 / arm64 | ✅ | ✅ 源码构建托盘应用 |

### 点击聚焦（Click-to-Focus）

系统通知可点击——点击后会跳回运行 agent 的终端 / 窗口。各平台行为不同：

- **macOS** — 默认应用级（激活 agent 所在的终端/IDE 应用）。若要窗口级（多窗口时也精确跳回那一个），在登录 shell 环境（如 `~/.zshrc`）里设置 `AGENT_NOTIFY_FOCUS_PRECISION=window`；这会用到内置 helper 并需要「辅助功能」权限。不设置则保持应用级。
- **Linux（X11）** — 窗口级。在会话启动时（通过 `SessionStart` hook）捕获精确的终端窗口，点击时跳回，因此能区分单进程多窗口终端（deepin-terminal、GNOME Terminal 等）的兄弟窗口。原生 Wayland 窗口无法定位。
- **Windows** — 通过内置 helper 跳回终端窗口。

> **`AGENT_NOTIFY_FOCUS_PRECISION`** 接受 `window`（窗口级）或 `app`（应用级，默认值）。取值不区分大小写、会去除首尾空白；未设置或无法识别的值都回退为 `app`。该变量**仅对 macOS 生效**——Linux 始终是窗口级，Windows 用自己的 helper。

系统渠道默认开启点击聚焦；目标应用/窗口会从 hook 的环境变量与进程树自动识别。

## 安装说明

```bash
bunx agent-notify
```

首次运行会从 GitHub Releases 下载当前 Bun 包版本对应平台的二进制文件，并安装到：

- macOS / Linux: `~/.agent-notify/agent-notify`
- Windows: `~/.agent-notify/agent-notify.exe`

之后每次运行都会检查本地二进制版本：不存在则自动下载，版本落后则自动更新，否则直接运行。launcher 不会持久修改 PATH，始终用绝对路径执行。

### Windows 部署

Docker 控制面与本机桌面运行时分开运行。在仓库根目录用 PowerShell 执行：

```powershell
.\deploy.ps1 up
.\deploy.ps1 desktop
```

脚本会构建 `agent-notify.exe` 与 `Agent Notify.exe`，写入当前用户的启动项
`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`，并启动桌面端。桌面端默认驻留系统托盘；可通过托盘菜单打开或退出。`down`、`restart`、`status`、`logs`、`upgrade`、`uninstall` 与 `deploy.sh` 对应。

macOS 上交叉编译出 EXE 只能证明构建通过，Toast、托盘、开机启动、终端聚焦和各 Agent Hook 必须在 Windows 10/11 真机验收。

### 个人微信机器人（iLink）与 Webhook 的区别

个人微信是独立的 `wechat-ilink` 通道，不是通用 `wechat` Webhook，也不要求安装 OpenClaw。桌面端会在 `127.0.0.1:45176` 启动本地 Bun Bridge，由 Bridge 完成腾讯 iLink 扫码登录、本地会话保存、长轮询、收件人绑定和 Agent Notify 消息转发。

配置流程：

1. 打开桌面端，选择「个人微信」；
2. 点击「连接微信」并扫描腾讯二维码；
3. 用目标个人微信给机器人发送一条普通消息，完成绑定并刷新会话上下文；
4. 点击「发送测试」或触发一次真实 Agent 事件。

Bridge 会持久化 `get_updates_buf` 和最新 `context_token`，接入腾讯 `notifystart` / `notifystop` 生命周期接口，并在服务端返回 `ret=-2` / `prepare failed` 时进行一次无上下文重试。这能改善空闲后的恢复，但不能承诺永久主动送达；上下文、Token 和服务端有效期由腾讯控制。若状态变为上下文失效，再从微信发送一条消息后重试。不宣传群聊能力。

不要把 iLink token 粘贴到通用 Webhook 输入框。iLink 状态保存在 `~/.agent-notify/wechat-ilink.json`，通道开关保存在 `~/.agent-notify/config.yaml`。

> **注意**: Codex 通过 `~/.codex/hooks.json` 接入官方 hooks 系统，目前仅订阅 `PermissionRequest`、`Stop` 两个事件。首次安装后请在 codex 内运行 `/hooks` 完成 trust 审核。
>
> **Grok**: 写入 `~/.grok/hooks/agent-notify.json`。全局 hooks 始终可信；项目级 hooks（`.grok/hooks/`）需在仓库内运行 `/hooks-trust` 或使用 `--trust`。安装后可在 Grok 中运行 `/hooks`（或 `Ctrl+L`）确认已加载。


## 配置说明

> agent-notify 不需要手动处理配置文件，该章节仅是为了说明配置相关信息。

agent-notify 自身配置位于 `~/.agent-notify/config.yaml`。**新安装默认关闭所有 Agent 与通知渠道**——需运行一次 `bunx agent-notify`（配置向导）启用你需要的 Agent 与渠道。这样可避免只配置了一个 Agent 后，在「查看配置 / 诊断」里把未配置的 Agent 显示为已就绪。已有配置文件不受影响。

Agent 集成配置位置：

- Claude Code: `~/.claude/settings.json`（写入 hooks → 命令 `agent-notify handle-claude-hook`）
- Codex: `~/.codex/hooks.json`（写入 hooks → 命令 `agent-notify handle-codex-hook`，需在 codex 内运行 `/hooks` 完成 trust）
- ZCode: `~/.zcode/cli/config.json`（写入 `hooks.events.<Event>` + `hooks.enabled` → 命令 `agent-notify handle-zcode-hook`；重启 ZCode 使配置生效）
- Grok: `~/.grok/hooks/agent-notify.json`（写入 hooks → 命令 `agent-notify handle-grok-hook`；项目 scope 为 `.grok/hooks/agent-notify.json`）
- Droid: `~/.factory/hooks.json`（写入 hooks → 命令 `agent-notify handle-droid-hook`；项目 scope 为 `.factory/hooks.json`）
- OpenCode: `~/.config/opencode/opencode.json`（写入 `plugin` 数组 → `~/.agent-notify/opencode-plugin.js`，命令 `agent-notify handle-opencode-hook`；项目 scope 为 `./opencode.json`）
- WorkBuddy / CodeBuddy: `~/.codebuddy/settings.json`（写入 `handle-workbuddy-hook`）
- Hermes Agent: `~/.hermes/hooks/agent-notify/`（写入 Gateway Hook 清单与 handler）
- OpenClaw: `~/.openclaw/extensions/agent-notify/`（写入 ESM 扩展）
- Pi: `~/.pi/agent/extensions/agent-notify.ts`（写入自动发现的 TypeScript 扩展；项目级为 `.pi/extensions/agent-notify.ts`）

远程通道统一配置在 `~/.agent-notify/config.yaml` 的 `remote` 下，由全部已启用 Agent 共享；个人微信 iLink 的登录会话和上下文单独保存在 `~/.agent-notify/wechat-ilink.json`。具体哪些事件发送，仍由各 Agent 的事件配置决定。

### 企业微信机器人绑定小技巧

1. **创建单人通知群**：在企业微信中发起群聊（随便拉几个同事），创建成功后**不要在群里发言**，直接将其他人移出，此时该群将变成你的单人通知群；
2. **添加机器人**：「群设置」->「消息推送」->「添加」-> 「自定义消息推送」，命名并保存；
3. **获取 Webhook 地址**：复制生成的地址，格式类似 `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`；
4. **绑定配置**：运行 `bunx agent-notify`，在配置向导中选择启用企业微信渠道，粘贴 Webhook URL 即可；
> 旧版企业微信添加机器人步骤：「群设置」->「群机器人」->「添加机器人」-> 「新建机器人」，命名并保存

## 工作流程

<p align="center">
  <img src="assist/workflow.png" alt="工作流程图" />
</p>

## 效果图

| | |
|:---:|:---:|
| <img src="assist/launch-setting.png" alt="软件配置" width="75%"> | <img src="assist/feishu-bind.png" alt="飞书绑定" width="75%"> |
| **软件配置** | **飞书绑定** |
| <img src="assist/feishu-notify-phone.png" alt="飞书通知" width="55%"> | <img src="assist/wecom-notify.jpg" alt="企业微信通知" width="55%"> |
| **飞书通知** | **企业微信通知** |
| <img src="assist/system-notify.png" alt="系统通知" width="55%"> | |
| **系统通知** | |

## 致谢

感谢 [LINUX DO](https://linux.do/) 社区朋友们的支持与反馈。
