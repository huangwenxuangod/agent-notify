import "./styles.css";
import "./channel-guide.css";
import "./app-shell-polish.css";
import {
  BellRing,
  Building2,
  Check,
  ChevronRight,
  CircleCheck,
  Gauge,
  Hash,
  LoaderCircle,
  MessageCircle,
  Monitor,
  Moon,
  MousePointer2,
  Palette,
  Radio,
  RefreshCw,
  Save,
  Send,
  Settings2,
  SlidersHorizontal,
  Smartphone,
  Sparkles,
  Sun,
  Webhook,
  createIcons,
} from "lucide";
import {
  AutostartStatus,
  Channel,
  Config,
  HookRuntimeStatus,
  autostart,
  autoSetup,
  clickToFocus,
	 codexHookStatus,
  config,
  openCodexHookReview,
  saveConfig,
  sendTestChannel,
	 testSystemNotification,
  setAutostart,
  setClickToFocus,
} from "./api";

type View = "channels" | "settings";
type SettingsTab = "general" | "notifications" | "appearance" | "advanced";
type Theme = "system" | "light" | "dark";
type RemoteKey = keyof Config["Remote"];
type Credential = {
  key: "SigningSecret" | "AccessToken";
  label: string;
  placeholder: string;
};
type Provider = {
  key: RemoteKey;
  label: string;
  hint: string;
  inputLabel: string;
  placeholder: string;
  icon: string;
  guide: string;
  docs: string;
  credential?: Credential;
};
type Data = {
  config: Config;
  autostart: AutostartStatus;
  clickToFocus: boolean;
	 codexHook: HookRuntimeStatus;
};

const app = document.querySelector<HTMLElement>("#app")!;
const icons = {
  BellRing,
  Building2,
  Check,
  ChevronRight,
  CircleCheck,
  Gauge,
  Hash,
  LoaderCircle,
  MessageCircle,
  Monitor,
  Moon,
  MousePointer2,
  Palette,
  Radio,
  RefreshCw,
  Save,
  Send,
  Settings2,
  SlidersHorizontal,
  Smartphone,
  Sparkles,
  Sun,
  Webhook,
};
const providers: Provider[] = [
  {
    key: "Ntfy",
    label: "ntfy",
    hint: "ntfy.sh 或自托管主题",
    inputLabel: "主题地址",
    placeholder: "https://ntfy.sh/your-topic",
    icon: "radio",
    guide: "创建或选择一个主题；私有主题可填写访问令牌。",
    docs: "https://docs.ntfy.sh/publish/",
    credential: {
      key: "AccessToken",
      label: "访问令牌（可选）",
      placeholder: "tk_...",
    },
  },
  {
    key: "Bark",
    label: "Bark",
    hint: "iPhone 推送服务",
    inputLabel: "推送地址",
    placeholder: "https://api.day.app/your-key",
    icon: "smartphone",
    guide: "在 Bark App 的服务器列表中复制设备推送地址。",
    docs: "https://bark.day.app/",
  },
  {
    key: "Feishu",
    label: "飞书",
    hint: "群机器人 Webhook",
    inputLabel: "Webhook URL",
    placeholder: "https://open.feishu.cn/open-apis/bot/v2/hook/...",
    icon: "message-circle",
    guide: "在目标群添加自定义机器人，复制 Webhook；若开启加签，同时填入密钥。",
    docs: "https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot",
    credential: {
      key: "SigningSecret",
      label: "加签密钥（可选）",
      placeholder: "机器人安全设置中的 Secret",
    },
  },
  {
    key: "WechatWork",
    label: "企业微信",
    hint: "群机器人 Webhook",
    inputLabel: "Webhook URL",
    placeholder: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...",
    icon: "building-2",
    guide: "在群聊中添加群机器人并复制 Webhook 地址。",
    docs: "https://developer.work.weixin.qq.com/document/path/91770",
  },
  {
    key: "DingTalk",
    label: "钉钉",
    hint: "群机器人 Webhook",
    inputLabel: "Webhook URL",
    placeholder: "https://oapi.dingtalk.com/robot/send?access_token=...",
    icon: "send",
    guide: "在目标群添加自定义机器人；若选择加签安全设置，同时填入密钥。",
    docs: "https://open.dingtalk.com/document/orgapp/custom-bot-to-send-group-chat-messages",
    credential: {
      key: "SigningSecret",
      label: "加签密钥（可选）",
      placeholder: "机器人安全设置中的 Secret",
    },
  },
  {
    key: "Slack",
    label: "Slack",
    hint: "Incoming Webhook",
    inputLabel: "Webhook URL",
    placeholder: "https://hooks.slack.com/services/...",
    icon: "hash",
    guide: "为 Slack App 启用 Incoming Webhooks，并添加到目标频道。",
    docs: "https://api.slack.com/messaging/webhooks",
  },
  {
    key: "Wechat",
    label: "自定义 Webhook",
    hint: "兼容现有通知网关",
    inputLabel: "Webhook URL",
    placeholder: "https://your-gateway.example/notify",
    icon: "webhook",
    guide: "接收端需要兼容 Agent Notify 的 JSON 文本消息格式。",
    docs: "https://github.com/hellolib/agent-notify",
  },
];

let view: View = "channels";
let setting: SettingsTab = "general";
let selected: RemoteKey = "Ntfy";
let data: Data | undefined;
let notice = "";
let theme = (localStorage.getItem("agent-notify-theme") as Theme) || "system";

const escape = (value: string) =>
  value.replace(
    /[&<>'"]/g,
    (char) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[
        char
      ]!,
  );
const endpoint = (channel: Channel) =>
  channel.TopicURL ?? channel.WebhookURL ?? "";
const providerFor = (key: RemoteKey) =>
  providers.find((provider) => provider.key === key)!;
const connected = (channel: Channel) =>
  channel.Enabled && endpoint(channel).length > 0;

function applyTheme() {
  document.documentElement.dataset.theme = theme;
  localStorage.setItem("agent-notify-theme", theme);
}
function icon(name: string) {
  return `<i data-lucide="${name}"></i>`;
}
function render() {
  applyTheme();
  app.innerHTML = `<div class="shell"><header class="topbar"><div class="wordmark">Agent Notify</div><nav class="primary-nav"><button class="nav-icon ${view === "channels" ? "active" : ""}" data-view="channels" title="通知渠道" aria-label="通知渠道">${icon("bell-ring")}</button><button class="nav-icon ${view === "settings" ? "active" : ""}" data-view="settings" title="设置" aria-label="设置">${icon("settings-2")}</button></nav><div class="toolbar"><button class="toolbar-icon" data-action="theme" title="切换主题" aria-label="切换主题">${icon(theme === "dark" ? "moon" : theme === "light" ? "sun" : "monitor")}</button><button class="toolbar-icon" data-action="refresh" title="刷新状态" aria-label="刷新状态">${icon("refresh-cw")}</button></div></header><main>${data ? (view === "channels" ? channelsPage() : settingsPage()) : loading()}</main>${notice ? `<div class="toast">${icon("circle-check")}<span>${escape(notice)}</span></div>` : ""}</div>`;
  createIcons({ icons });
  bind();
}
function loading() {
  return `<section class="loading">${icon("loader-circle")}</section>`;
}
function channelsPage() {
  const provider = providerFor(selected);
  const channel = data!.config.Remote[selected];
  const active = connected(channel);
  const credential = provider.credential;
  const credentialSaved = credential && Boolean(channel[credential.key]);
  return `<section class="workspace channels-workspace"><div class="page-intro"><div><p class="eyebrow">远程通知</p><h1>选择你的送达渠道</h1></div><span class="delivery-count">${providers.filter((provider) => connected(data!.config.Remote[provider.key])).length} 个已配置</span></div><div class="provider-picker">${providers
    .map((item) => {
      const itemChannel = data!.config.Remote[item.key];
      return `<button class="provider ${selected === item.key ? "selected" : ""}" data-provider="${item.key}" title="${item.label}"><span class="provider-icon">${icon(item.icon)}</span><span>${item.label}</span>${connected(itemChannel) ? "<b></b>" : ""}</button>`;
    })
    .join(
      "",
    )}</div><section class="configuration-panel"><div class="provider-summary"><span class="summary-icon">${icon(provider.icon)}</span><div><h2>${provider.label}</h2><p>${provider.hint}</p></div><label class="switch" title="启用 ${provider.label}"><input type="checkbox" data-channel-enabled ${channel.Enabled ? "checked" : ""}><span></span></label></div><div class="setup-guide"><p>${provider.guide}</p><a href="${provider.docs}" target="_blank" rel="noreferrer">打开官方说明 ${icon("chevron-right")}</a></div><div class="field-wrap ${channel.Enabled ? "" : "disabled"}"><label for="endpoint">${provider.inputLabel}</label><input id="endpoint" value="${escape(endpoint(channel))}" placeholder="${provider.placeholder}" ${channel.Enabled ? "" : "disabled"} autocomplete="off" spellcheck="false"></div>${credential ? `<div class="field-wrap credential-field ${channel.Enabled ? "" : "disabled"}"><label for="credential">${credential.label}</label><input id="credential" type="password" value="" placeholder="${credentialSaved ? "已保存，留空保持不变" : credential.placeholder}" ${channel.Enabled ? "" : "disabled"} autocomplete="new-password" spellcheck="false"></div>` : ""}<div class="panel-actions"><span class="connection-state ${active ? "ready" : ""}">${active ? `${icon("circle-check")} 已配置` : "尚未配置"}</span><div><button class="secondary" data-action="test" ${active ? "" : "disabled"}>${icon("send")}发送测试</button><button class="primary" data-action="save">${icon("save")}保存更改</button></div></div></section></section>`;
}
function settingsPage() {
  return `<section class="workspace settings-workspace"><aside class="settings-nav"><button class="settings-item ${setting === "general" ? "active" : ""}" data-setting="general">${icon("settings-2")}通用</button><button class="settings-item ${setting === "notifications" ? "active" : ""}" data-setting="notifications">${icon("bell-ring")}通知</button><button class="settings-item ${setting === "appearance" ? "active" : ""}" data-setting="appearance">${icon("palette")}外观</button><button class="settings-item ${setting === "advanced" ? "active" : ""}" data-setting="advanced">${icon("gauge")}高级</button></aside><section class="settings-content">${setting === "general" ? generalSettings() : setting === "notifications" ? notificationSettings() : setting === "appearance" ? appearanceSettings() : advancedSettings()}</section></section>`;
}
function settingHeader(title: string, description: string) {
  return `<div class="setting-header"><p class="eyebrow">设置</p><h1>${title}</h1><p>${description}</p></div>`;
}
function toggleRow(
  title: string,
  description: string,
  attribute: string,
  checked: boolean,
  disabled = false,
) {
  return `<div class="setting-row"><div><h2>${title}</h2><p>${description}</p></div><label class="switch ${disabled ? "is-disabled" : ""}"><input type="checkbox" ${attribute} ${checked ? "checked" : ""} ${disabled ? "disabled" : ""}><span></span></label></div>`;
}
function generalSettings() {
  return `${settingHeader("通用", "桌面应用的运行方式。")}<div class="setting-list">${toggleRow("登录时启动", "登录 macOS 后自动运行 Agent Notify。", "data-autostart", data!.autostart.Enabled, !data!.autostart.Supported)}<div class="setting-row static"><div><h2>关闭窗口时隐藏</h2><p>应用继续在菜单栏运行，通知链路不会中断。</p></div>${icon("check")}</div></div>`;
}
function notificationSettings() {
  const hook = data!.codexHook;
  const runtimeState = hook.last_event_at ? `最近真实事件：${new Date(hook.last_event_at).toLocaleString()} · ${hook.last_event}` : hook.installed ? "已写入，尚未收到真实 Hook 事件" : "尚未写入 Hook";
  return `${settingHeader("通知", "管理宿主机的本地提醒行为。")}<div class="setting-list">${toggleRow("点击后回到工作应用", "点击系统通知时，尝试激活触发事件的终端或 IDE。", "data-focus", data!.clickToFocus)}<div class="setting-row"><div><h2>Codex Hook</h2><p>${runtimeState}</p></div><button class="secondary" data-action="codex-hook-review">${icon("check")}重新打开审核</button></div><div class="setting-row"><div><h2>系统通知</h2><p>直接由本机发送，不经过 Docker。</p></div><button class="secondary" data-action="test-system">${icon("send")}发送测试</button></div></div>`;
}
function appearanceSettings() {
  return `${settingHeader("外观", "选择 Agent Notify 的显示模式。")}<div class="theme-grid">${(["system", "light", "dark"] as Theme[]).map((item) => `<button class="theme-choice ${theme === item ? "active" : ""}" data-theme="${item}">${icon(item === "system" ? "monitor" : item === "light" ? "sun" : "moon")}<span>${item === "system" ? "跟随系统" : item === "light" ? "浅色" : "深色"}</span>${theme === item ? icon("check") : ""}</button>`).join("")}</div>`;
}
function advancedSettings() {
  const seconds = data!.config.Behavior.DedupeSeconds;
  return `${settingHeader("高级", "控制远程通知的投递节奏。")}<div class="setting-list"><label class="setting-row select-row"><div><h2>重复提醒间隔</h2><p>同一会话的相同通知在这段时间内只发送一次。</p></div><select data-dedupe><option value="10" ${seconds === 10 ? "selected" : ""}>10 秒</option><option value="30" ${seconds === 30 ? "selected" : ""}>30 秒</option><option value="60" ${seconds === 60 ? "selected" : ""}>60 秒</option></select></label></div><div class="advanced-actions"><button class="primary" data-action="save-dedupe">${icon("save")}保存更改</button></div>`;
}
async function refresh() {
  notice = "";
  render();
  try {
    await autoSetup();
    const [nextConfig, nextAutostart, nextClickToFocus, nextCodexHook] = await Promise.all([
      config(),
      autostart(),
      clickToFocus(),
		codexHookStatus(),
    ]);
    data = {
      config: nextConfig,
      autostart: nextAutostart,
      clickToFocus: nextClickToFocus,
		codexHook: nextCodexHook,
    };
  } catch (error) {
    notice = `无法连接 Docker Bridge: ${String(error)}`;
  }
  render();
}
async function saveChannel() {
  if (!data) return;
  const next = structuredClone(data.config);
  const channel = next.Remote[selected];
  const provider = providerFor(selected);
  channel.Enabled =
    app.querySelector<HTMLInputElement>("[data-channel-enabled]")?.checked ??
    false;
  const value =
    app.querySelector<HTMLInputElement>("#endpoint")?.value.trim() ?? "";
  if (selected === "Ntfy") channel.TopicURL = value;
  else channel.WebhookURL = value;
  const credential = app
    .querySelector<HTMLInputElement>("#credential")
    ?.value.trim();
  if (provider.credential && credential)
    channel[provider.credential.key] = credential;
  await saveConfig(next);
  data.config = next;
  notice = "渠道配置已保存";
  render();
}
async function saveDedupe() {
  if (!data) return;
  const next = structuredClone(data.config);
  next.Behavior.DedupeSeconds = Number(
    app.querySelector<HTMLSelectElement>("[data-dedupe]")?.value ?? 10,
  );
  await saveConfig(next);
  data.config = next;
  notice = "通知设置已保存";
  render();
}
function bind() {
	const channelToggle = app.querySelector<HTMLInputElement>(
		"[data-channel-enabled]",
	);
	if (data && endpoint(data.config.Remote[selected]) === "" && channelToggle) {
		channelToggle.checked = true;
	}
	app
		.querySelectorAll<HTMLInputElement>("#endpoint, #credential")
		.forEach((input) => {
			input.disabled = false;
		});
	app.querySelectorAll<HTMLElement>("[data-view]").forEach((element) =>
    element.addEventListener("click", () => {
      view = element.dataset.view as View;
      render();
    }),
  );
  app.querySelectorAll<HTMLElement>("[data-provider]").forEach((element) =>
    element.addEventListener("click", () => {
      selected = element.dataset.provider as RemoteKey;
      render();
    }),
  );
  app.querySelectorAll<HTMLElement>("[data-setting]").forEach((element) =>
    element.addEventListener("click", () => {
      setting = element.dataset.setting as SettingsTab;
      render();
    }),
  );
  app
    .querySelector('[data-action="refresh"]')
    ?.addEventListener("click", refresh);
  app.querySelector('[data-action="theme"]')?.addEventListener("click", () => {
    theme =
      theme === "system" ? "light" : theme === "light" ? "dark" : "system";
    render();
  });
  app.querySelectorAll<HTMLElement>("[data-theme]").forEach((element) =>
    element.addEventListener("click", () => {
      theme = element.dataset.theme as Theme;
      render();
    }),
  );
  app
    .querySelector<HTMLInputElement>("[data-channel-enabled]")
    ?.addEventListener("change", render);
  app.querySelector('[data-action="save"]')?.addEventListener(
    "click",
    () =>
      void saveChannel().catch((error) => {
        notice = `保存失败: ${String(error)}`;
        render();
      }),
  );
  app
    .querySelector('[data-action="test"]')
    ?.addEventListener("click", async () => {
      try {
        await sendTestChannel(
          selected.toLowerCase().replace("wechatwork", "wechat-work"),
        );
        notice = "测试通知已发送";
        render();
      } catch (error) {
        notice = `测试失败: ${String(error)}`;
        render();
      }
    });
  app.querySelector('[data-action="codex-hook-review"]')?.addEventListener(
    "click",
    async () => {
      try {
        await openCodexHookReview();
        notice = "已打开 Codex 审核窗口，输入 /hooks 后信任 Agent Notify";
        render();
      } catch (error) {
        notice = `无法打开 Codex 审核: ${String(error)}`;
        render();
      }
    },
  );
  app.querySelector('[data-action="test-system"]')?.addEventListener(
    "click",
    async () => {
      try {
        await testSystemNotification();
        notice = "系统测试通知已发送";
        render();
      } catch (error) {
        notice = `系统通知测试失败: ${String(error)}`;
        render();
      }
    },
  );
  app.querySelector('[data-action="save-dedupe"]')?.addEventListener(
    "click",
    () =>
      void saveDedupe().catch((error) => {
        notice = `保存失败: ${String(error)}`;
        render();
      }),
  );
  app
    .querySelector<HTMLInputElement>("[data-autostart]")
    ?.addEventListener("change", async (event) => {
      try {
        const enabled = (event.target as HTMLInputElement).checked;
        await setAutostart(enabled);
        if (data) data.autostart.Enabled = enabled;
        notice = "通用设置已保存";
        render();
      } catch (error) {
        notice = `保存失败: ${String(error)}`;
        render();
      }
    });
  app
    .querySelector<HTMLInputElement>("[data-focus]")
    ?.addEventListener("change", async (event) => {
      try {
        const enabled = (event.target as HTMLInputElement).checked;
        await setClickToFocus(enabled);
        if (data) data.clickToFocus = enabled;
        notice = "通知设置已保存";
        render();
      } catch (error) {
        notice = `保存失败: ${String(error)}`;
        render();
      }
    });
}
refresh();
