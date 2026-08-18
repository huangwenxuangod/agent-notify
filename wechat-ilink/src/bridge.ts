import { chmod, mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import QRCode from "qrcode";

type Session = { botToken: string; baseUrl: string };
type Binding = { userId: string; contextToken?: string; contextUpdatedAt?: string };
type PendingLogin = { qrcode: string; qrUrl: string; qrDataURL: string };
type PersistedState = { session?: Session; binding?: Binding; pending?: PendingLogin; syncBuf?: string; sessionExpired?: boolean; lastDeliveryAt?: string; lastDeliveryError?: string };
type FetchLike = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export type BridgeOptions = { stateDir: string; fetchImpl?: FetchLike; startMonitor?: boolean };

export type IlinkBridge = {
  handle(request: Request): Promise<Response>;
  setSession(session: Session): Promise<void>;
  bind(binding: Binding): Promise<void>;
  stop(): Promise<void>;
};

const stateFile = "wechat-ilink.json";
// Keep the wire metadata aligned with Tencent's supported iLink client.
const channelVersion = "2.4.6";
const ilinkAppID = "bot";
const ilinkAppClientVersion = "132102"; // 0x020406 (2.4.6)
const sendTimeoutMs = 15_000;
const staleSessionErrorCode = -14;

export async function createBridge(options: BridgeOptions): Promise<IlinkBridge> {
  const fetchImpl = options.fetchImpl ?? fetch;
  await mkdir(options.stateDir, { recursive: true });
  const path = join(options.stateDir, stateFile);
  let state = await loadState(path);
  let sendTail: Promise<void> = Promise.resolve();
  let nextSendAt = 0;
  let monitorRunning = false;
  let lifecycleStarted = false;
  let lifecycleReady: Promise<void> | undefined;

  const save = async () => {
    await writeFile(path, JSON.stringify(state, null, 2), "utf8");
    if (process.platform !== "win32") await chmod(path, 0o600);
  };

  const withSendCooldown = async <T>(work: () => Promise<T>): Promise<T> => {
    const previous = sendTail;
    let release!: () => void;
    sendTail = new Promise<void>((resolve) => { release = resolve; });
    await previous;
    try {
      const wait = nextSendAt - Date.now();
      if (wait > 0) await new Promise((resolve) => setTimeout(resolve, wait));
      return await work();
    } finally {
      // Tencent rejects bursts from the same iLink session. Keep this global
      // because multiple hook processes can call the local bridge concurrently.
      nextSendAt = Date.now() + 2000;
      release();
    }
  };

  const markSessionExpired = async () => {
    state.sessionExpired = true;
    state.lastDeliveryError = "微信会话已过期，请重新连接";
    await save();
  };

  const notifyLifecycle = async (session: Session, endpoint: "notifystart" | "notifystop") => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);
    try {
      const response = await fetchImpl(`${session.baseUrl.replace(/\/$/, "")}/ilink/bot/msg/${endpoint}`, {
        method: "POST",
        headers: ilinkHeaders(session.botToken),
        body: JSON.stringify({ base_info: { channel_version: channelVersion } }),
        signal: controller.signal,
      });
      if (!response.ok) throw new Error(`iLink ${endpoint} HTTP ${response.status}`);
      const result = await response.json().catch(() => ({ ret: 0 })) as { ret?: number; errcode?: number; errmsg?: string };
      if ((result.ret && result.ret !== 0) || (result.errcode && result.errcode !== 0)) {
        throw new Error(result.errmsg ?? `iLink ${endpoint} ret=${result.ret ?? result.errcode}`);
      }
    } finally {
      clearTimeout(timer);
    }
  };

  const monitor = async () => {
    if (monitorRunning || !state.session) return;
    monitorRunning = true;
    const sessionAtStart = state.session;
    try {
      lifecycleReady = notifyLifecycle(sessionAtStart, "notifystart");
      try {
        await lifecycleReady;
        lifecycleStarted = true;
      } catch (error) {
        state.lastDeliveryError = `iLink 通知监听启动失败：${describeNetworkError(error)}`;
        await save();
      }
      let cursor = state.syncBuf ?? "";
      while (monitorRunning && state.session === sessionAtStart) {
        try {
          const session = state.session;
          if (!session) break;
          const response = await fetchImpl(`${session.baseUrl.replace(/\/$/, "")}/ilink/bot/getupdates`, {
            method: "POST",
            headers: ilinkHeaders(session.botToken),
            body: JSON.stringify({ get_updates_buf: cursor, base_info: { channel_version: channelVersion } }),
          });
          if (!response.ok) throw new Error(`getupdates HTTP ${response.status}`);
          const payload = await response.json() as { ret?: number; errcode?: number; msgs?: Array<{ message_type?: number; from_user_id?: string; context_token?: string }>; get_updates_buf?: string };
          if (payload.errcode === staleSessionErrorCode || payload.ret === staleSessionErrorCode) {
            await markSessionExpired();
            return;
          }
          if (payload.ret && payload.ret !== 0) throw new Error(`getupdates ret=${payload.ret}`);
          if (payload.get_updates_buf) {
            cursor = payload.get_updates_buf;
            state.syncBuf = cursor;
            await save();
          }
          for (const message of payload.msgs ?? []) {
            if (message.message_type !== 1 || !message.from_user_id) continue;
            if (!state.binding || state.binding.userId === message.from_user_id) {
              const previous = state.binding;
              state.binding = {
                userId: message.from_user_id,
                contextToken: message.context_token ?? previous?.contextToken,
                contextUpdatedAt: message.context_token ? new Date().toISOString() : previous?.contextUpdatedAt,
              };
              await save();
            }
          }
        } catch {
          if (monitorRunning) await new Promise((resolve) => setTimeout(resolve, 3000));
        }
      }
    } finally {
      monitorRunning = false;
    }
  };

  if (options.startMonitor !== false && state.session) void monitor();

  const setSession = async (session: Session) => {
    state = { ...state, session, sessionExpired: false };
    await save();
  };
  const bind = async (binding: Binding) => {
    state = { ...state, binding };
    await save();
  };

  const stop = async () => {
    monitorRunning = false;
    const session = state.session;
    if (!session) return;
    try {
      if (lifecycleReady) await lifecycleReady.catch(() => undefined);
      if (!lifecycleStarted) return;
      await notifyLifecycle(session, "notifystop");
    } catch (error) {
      state.lastDeliveryError = `iLink 通知监听停止失败：${describeNetworkError(error)}`;
      await save();
    } finally {
      lifecycleStarted = false;
    }
  };

  return {
    setSession,
    bind,
    stop,
    async handle(request) {
      const url = new URL(request.url);
      if (request.method === "GET" && url.pathname === "/health") {
        return json({
          logged_in: Boolean(state.session),
          bound: Boolean(state.binding),
          session_expired: Boolean(state.sessionExpired),
          user_id: state.binding?.userId,
          context_updated_at: state.binding?.contextUpdatedAt,
          monitor_cursor_present: Boolean(state.syncBuf),
          last_delivery_at: state.lastDeliveryAt,
          last_delivery_error: state.lastDeliveryError,
          delivery_state: !state.session ? "logged_out" : state.sessionExpired ? "session_expired" : !state.binding ? "unbound" : state.lastDeliveryError && /prepare failed|ret=-2/i.test(state.lastDeliveryError) ? "context_stale" : "ready",
        });
      }
      if (request.method === "GET" && url.pathname === "/login") {
        return json({ status: state.session ? "confirmed" : state.pending ? "wait" : "idle", qr_url: state.pending?.qrUrl, qr_data_url: state.pending?.qrDataURL });
      }
      if (request.method === "POST" && (url.pathname === "/login" || url.pathname === "/reconnect")) {
        if (url.pathname === "/reconnect") {
          monitorRunning = false;
          lifecycleStarted = false;
          state.session = undefined;
          state.binding = undefined;
          state.pending = undefined;
          state.syncBuf = undefined;
          state.sessionExpired = false;
          state.lastDeliveryAt = undefined;
          state.lastDeliveryError = undefined;
          await save();
        }
        const response = await fetchImpl("https://ilinkai.weixin.qq.com/ilink/bot/get_bot_qrcode?bot_type=3");
        if (!response.ok) return json({ error: `iLink QR HTTP ${response.status}` }, 502);
        const payload = await response.json() as { qrcode?: string; qrcode_img_content?: string };
        if (!payload.qrcode || !payload.qrcode_img_content) return json({ error: "iLink QR response is incomplete" }, 502);
        state.pending = {
          qrcode: payload.qrcode,
          qrUrl: payload.qrcode_img_content,
          qrDataURL: await QRCode.toDataURL(payload.qrcode_img_content, { width: 240, margin: 1 }),
        };
        await save();
        return json({ status: "wait", qr_url: payload.qrcode_img_content, qr_data_url: state.pending.qrDataURL });
      }
      if (request.method === "POST" && url.pathname === "/login/poll") {
        if (!state.pending) return json({ status: state.session ? "confirmed" : "idle" });
        const response = await fetchImpl(`https://ilinkai.weixin.qq.com/ilink/bot/get_qrcode_status?qrcode=${encodeURIComponent(state.pending.qrcode)}`);
        if (!response.ok) return json({ error: `iLink QR status HTTP ${response.status}` }, 502);
        const payload = await response.json() as { status?: string; bot_token?: string; baseurl?: string };
        if (payload.status === "confirmed" && payload.bot_token) {
          state.session = { botToken: payload.bot_token, baseUrl: payload.baseurl || "https://ilinkai.weixin.qq.com" };
          state.pending = undefined;
          await save();
          if (options.startMonitor !== false) void monitor();
        }
        return json({ status: payload.status ?? "wait" });
      }
      if (request.method === "POST" && url.pathname === "/bind") {
        const body = await request.json() as Binding;
        if (!body.userId) return json({ error: "userId is required" }, 400);
        await bind(body);
        return new Response(null, { status: 204 });
      }
      if (request.method !== "POST" || url.pathname !== "/send") return json({ error: "not found" }, 404);
      if (!state.session) return json({ error: "WeChat bot is not logged in" }, 401);
      if (state.sessionExpired) return json({ error: "微信会话已过期，请重新连接" }, 401);
      if (!state.binding) return json({ error: "Send one message to the bot to bind this WeChat account" }, 409);

      const message = await request.json() as { title?: string; content?: string };
      const text = [message.title, message.content].filter(Boolean).join("\n");
      const clientID = `agent-notify-${crypto.randomUUID()}`;
      const payloadFor = (contextToken?: string) => JSON.stringify({
        msg: {
          from_user_id: "",
          to_user_id: state.binding!.userId,
          client_id: clientID,
          message_type: 2,
          message_state: 2,
          context_token: contextToken,
          item_list: [{ type: 1, text_item: { text } }],
        },
        base_info: { channel_version: channelVersion, bot_agent: "AgentNotify/1.0" },
      });
      let response: Response;
      try {
        response = await withSendCooldown(async () => {
          const url = `${state.session!.baseUrl.replace(/\/$/, "")}/ilink/bot/sendmessage`;
          const headers = ilinkHeaders(state.session!.botToken);
          const first = await fetchWithRetry(fetchImpl, url, headers, payloadFor(state.binding!.contextToken));
          const firstResult = await first.clone().json().catch(() => ({})) as { ret?: number; errmsg?: string };
          if (isStaleContextFailure(firstResult)) {
            return fetchWithRetry(fetchImpl, url, headers, payloadFor());
          }
          return first;
        });
      } catch (error) {
        state.lastDeliveryError = describeNetworkError(error);
        await save();
        return json({ error: state.lastDeliveryError }, 502);
      }
      if (!response.ok) {
        state.lastDeliveryError = `iLink returned HTTP ${response.status}`;
        await save();
        return json({ error: state.lastDeliveryError }, 502);
      }
      const result = await response.json().catch(() => ({ ret: 0 })) as { ret?: number; errcode?: number; errmsg?: string };
      if (result.errcode === staleSessionErrorCode || result.ret === staleSessionErrorCode) {
        await markSessionExpired();
        return json({ error: state.lastDeliveryError }, 401);
      }
      if ((result.ret && result.ret !== 0) || (result.errcode && result.errcode !== 0)) {
        state.lastDeliveryError = result.errmsg ?? `iLink returned ${result.errcode ?? result.ret}`;
        await save();
        return json({ error: state.lastDeliveryError }, 502);
      }
      state.lastDeliveryAt = new Date().toISOString();
      state.lastDeliveryError = undefined;
      await save();
      return new Response(null, { status: 204 });
    },
  };
}

function ilinkHeaders(token: string) {
  const uin = Buffer.from(String(crypto.getRandomValues(new Uint32Array(1))[0])).toString("base64");
  return {
    "Content-Type": "application/json",
    AuthorizationType: "ilink_bot_token",
    "X-WECHAT-UIN": uin,
    "iLink-App-Id": ilinkAppID,
    "iLink-App-ClientVersion": ilinkAppClientVersion,
    Authorization: `Bearer ${token}`,
  };
}

async function fetchWithRetry(fetchImpl: FetchLike, url: string, headers: Record<string, string>, body: string): Promise<Response> {
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), sendTimeoutMs);
    try {
      return await fetchImpl(url, { method: "POST", headers, body, signal: controller.signal });
    } catch (error) {
      lastError = error;
      if (!isTransientNetworkError(error) || attempt === 2) throw error;
      await new Promise((resolve) => setTimeout(resolve, 300 * (attempt + 1)));
    } finally {
      clearTimeout(timer);
    }
  }
  throw lastError;
}

function isTransientNetworkError(error: unknown): boolean {
  const value = `${String(error)} ${(error as { code?: string; cause?: { code?: string } })?.code ?? ""} ${(error as { cause?: unknown })?.cause ?? ""}`;
  return /ECONNRESET|UND_ERR_SOCKET|ETIMEDOUT|UND_ERR_CONNECT_TIMEOUT|EAI_AGAIN|ENETUNREACH|EHOSTUNREACH|AbortError/i.test(value);
}

function describeNetworkError(error: unknown): string {
  const value = String(error);
  if (/AbortError/i.test(value)) return "iLink 请求超时";
  const code = (error as { code?: string; cause?: { code?: string } })?.code ?? (error as { cause?: { code?: string } })?.cause?.code;
  return `iLink 网络请求失败${code ? `：${code}` : ""}`;
}

function isStaleContextFailure(result: { ret?: number; errmsg?: string }): boolean {
  if (result.ret !== -2) return false;
  return !result.errmsg || /prepare failed|unknown error/i.test(result.errmsg);
}

async function loadState(path: string): Promise<PersistedState> {
  try {
    return JSON.parse(await readFile(path, "utf8")) as PersistedState;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return {};
    throw error;
  }
}

function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}

export function bridgeServerOptions(bridge: Pick<IlinkBridge, "handle">, port?: number) {
  return { hostname: "127.0.0.1", port, idleTimeout: 60, fetch: bridge.handle };
}

if (import.meta.main) {
  const stateDir = process.env.AGENT_NOTIFY_STATE_DIR ?? `${process.env.HOME ?? process.env.USERPROFILE}/.agent-notify`;
  const port = Number(process.env.AGENT_NOTIFY_ILINK_PORT ?? "45176");
  const bridge = await createBridge({ stateDir });
  const server = Bun.serve(bridgeServerOptions(bridge, port));
  let stopping = false;
  const shutdown = async () => {
    if (stopping) return;
    stopping = true;
    await bridge.stop();
    server.stop(true);
  };
  process.on("SIGTERM", () => void shutdown());
  process.on("SIGINT", () => void shutdown());
  console.log(`Agent Notify WeChat iLink bridge listening on http://127.0.0.1:${port}`);
}
