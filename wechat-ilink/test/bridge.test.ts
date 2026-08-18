import { afterEach, expect, test } from "bun:test";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { bridgeServerOptions, createBridge } from "../src/bridge";

const roots: string[] = [];

afterEach(async () => {
  await Promise.all(roots.splice(0).map((root) => rm(root, { recursive: true, force: true })));
});

async function bridgeForTest() {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  return createBridge({ stateDir: root, fetchImpl: async () => new Response(JSON.stringify({ ret: 0 })) });
}

test("send rejects an unbound personal WeChat account", async () => {
  const bridge = await bridgeForTest();
	await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ title: "Done", content: "test" }),
  }));
  expect(response.status).toBe(409);
});

test("send records the latest iLink failure for the desktop status", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
	const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async () => new Response(JSON.stringify({ ret: -2, errmsg: "prepare failed" })),
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "context" });

  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ title: "Done", content: "test" }),
  }));
  expect(response.status).toBe(502);

  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toMatchObject({
    logged_in: true,
    bound: true,
    session_expired: false,
    last_delivery_error: "prepare failed",
  });
});

test("send retries once without a stale context token after iLink prepare failure", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  const requests: Array<{ msg: { client_id: string; context_token?: string; to_user_id: string } }> = [];
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (_url, init) => {
      const request = JSON.parse(String(init?.body)) as { msg: { client_id: string; context_token?: string; to_user_id: string } };
      requests.push(request);
      return new Response(JSON.stringify(requests.length === 1
        ? { ret: -2, errmsg: "prepare failed" }
        : { ret: 0 }));
    },
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "stale-context" });

  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ content: "test" }),
  }));

  expect(response.status).toBe(204);
  expect(requests).toHaveLength(2);
  expect(requests[0].msg).toMatchObject({ to_user_id: "owner@im.wechat", context_token: "stale-context" });
  expect(requests[1].msg).toMatchObject({ to_user_id: "owner@im.wechat", client_id: requests[0].msg.client_id });
  expect(requests[1].msg.context_token).toBeUndefined();
});

test("send marks an expired iLink session as requiring reconnect", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
	roots.push(root);
	await mkdir(root, { recursive: true });
	const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async () => new Response(JSON.stringify({ errcode: -14, errmsg: "session timeout" })),
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "context" });

  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ content: "test" }),
  }));
  expect(response.status).toBe(401);
  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toMatchObject({
    logged_in: true,
    bound: true,
    session_expired: true,
    last_delivery_error: "微信会话已过期，请重新连接",
  });
});

test("monitor marks a stale token as requiring reconnect", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
	await writeFile(join(root, "wechat-ilink.json"), JSON.stringify({ session: { botToken: "token", baseUrl: "https://ilink.example" } }));
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async () => new Response(JSON.stringify({ errcode: -14, errmsg: "session timeout" })),
  });
  for (let attempt = 0; attempt < 10; attempt++) {
    const health = await bridge.handle(new Request("http://localhost/health"));
    if ((await health.json() as { session_expired?: boolean }).session_expired) return;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  throw new Error("monitor did not mark the stale session");
});

test("send retries a reset connection with the same client id", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  let calls = 0;
  const clientIDs: string[] = [];
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (_url, init) => {
      calls++;
      clientIDs.push((JSON.parse(String(init?.body)) as { msg: { client_id: string } }).msg.client_id);
      if (calls === 1) throw Object.assign(new Error("socket reset"), { code: "ECONNRESET" });
      return new Response(JSON.stringify({ ret: 0 }));
    },
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "context" });

  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ content: "test" }),
  }));
  expect(response.status).toBe(204);
  expect(calls).toBe(2);
  expect(clientIDs[0]).toBe(clientIDs[1]);
});

test("send posts an iLink text payload for the bound account", async () => {
  let request: unknown;
  let headers: Headers | undefined;
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (_url, init) => {
      request = JSON.parse(String(init?.body));
      headers = new Headers(init?.headers);
      return new Response(JSON.stringify({ ret: 0 }));
    },
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "context" });

  const response = await bridge.handle(new Request("http://localhost/send", {
    method: "POST", body: JSON.stringify({ title: "Done", content: "test" }),
  }));
  expect(response.status).toBe(204);
  expect(request).toEqual({
    msg: {
      from_user_id: "",
      to_user_id: "owner@im.wechat",
      client_id: expect.any(String),
      message_type: 2,
      message_state: 2,
      context_token: "context",
      item_list: [{ type: 1, text_item: { text: "Done\ntest" } }],
    },
    base_info: { channel_version: "2.4.6", bot_agent: "AgentNotify/1.0" },
  });
  expect(headers?.get("iLink-App-Id")).toBe("bot");
  expect(headers?.get("iLink-App-ClientVersion")).toBe("132102");

  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toMatchObject({
    logged_in: true,
    bound: true,
    session_expired: false,
    user_id: "owner@im.wechat",
    last_delivery_at: expect.any(String),
  });
});

test("serializes concurrent sends through one iLink session", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  let active = 0;
  let maxActive = 0;
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async () => {
      active++;
      maxActive = Math.max(maxActive, active);
      await new Promise((resolve) => setTimeout(resolve, 10));
      active--;
      return new Response(JSON.stringify({ ret: 0 }));
    },
  });
  await bridge.setSession({ botToken: "token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "owner@im.wechat", contextToken: "context" });
  await Promise.all([
    bridge.handle(new Request("http://localhost/send", { method: "POST", body: JSON.stringify({ content: "one" }) })),
    bridge.handle(new Request("http://localhost/send", { method: "POST", body: JSON.stringify({ content: "two" }) })),
  ]);
  expect(maxActive).toBe(1);
});

test("login creates a QR session without exposing credentials", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  const bridge = await createBridge({
    stateDir: root,
    startMonitor: false,
    fetchImpl: async () => new Response(JSON.stringify({ qrcode: "qr-id", qrcode_img_content: "https://qr.example/image" })),
  });
  const response = await bridge.handle(new Request("http://localhost/login", { method: "POST" }));
  expect(response.status).toBe(200);
  expect(await response.json()).toEqual({
    status: "wait",
    qr_url: "https://qr.example/image",
    qr_data_url: expect.stringMatching(/^data:image\/png;base64,/),
  });
  const state = await readFile(join(root, "wechat-ilink.json"), "utf8");
  expect(state).not.toContain("bot_token");
});

test("reconnect clears the old session and starts a fresh QR login", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  const bridge = await createBridge({
    stateDir: root,
    startMonitor: false,
    fetchImpl: async () => new Response(JSON.stringify({ qrcode: "new-qr", qrcode_img_content: "https://qr.example/new" })),
  });
  await bridge.setSession({ botToken: "old-token", baseUrl: "https://ilink.example" });
  await bridge.bind({ userId: "old@im.wechat", contextToken: "old-context" });
  const response = await bridge.handle(new Request("http://localhost/reconnect", { method: "POST" }));
  expect(response.status).toBe(200);
  expect(await response.json()).toMatchObject({ status: "wait", qr_url: "https://qr.example/new" });
  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toMatchObject({ logged_in: false, bound: false });
});

test("bridge keeps long iLink polling requests alive beyond Bun default", () => {
  expect(bridgeServerOptions({ handle: async () => new Response() }).idleTimeout).toBe(60);
});

test("monitor persists and resumes the iLink sync cursor", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  await writeFile(join(root, "wechat-ilink.json"), JSON.stringify({
    session: { botToken: "token", baseUrl: "https://ilink.example" },
  }));
  const requests: Array<{ url: string; body: { get_updates_buf?: string } }> = [];
  let calls = 0;
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (input, init) => {
      const url = String(input);
      const body = JSON.parse(String(init?.body)) as { get_updates_buf?: string };
      requests.push({ url, body });
      if (url.endsWith("/notifystart")) return new Response(JSON.stringify({ ret: 0 }));
      calls++;
      if (calls === 1) return new Response(JSON.stringify({ ret: 0, get_updates_buf: "cursor-2", msgs: [] }));
      await new Promise((resolve) => setTimeout(resolve, 100));
      return new Response(JSON.stringify({ ret: 0, get_updates_buf: "cursor-2", msgs: [] }));
    },
  });
  for (let attempt = 0; attempt < 20; attempt++) {
    const state = JSON.parse(await readFile(join(root, "wechat-ilink.json"), "utf8")) as { syncBuf?: string };
    if (state.syncBuf === "cursor-2") break;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  const state = JSON.parse(await readFile(join(root, "wechat-ilink.json"), "utf8")) as { syncBuf?: string };
  expect(state.syncBuf).toBe("cursor-2");
  expect(requests.some((request) => request.body.get_updates_buf === "cursor-2")).toBe(true);
  await bridge.stop();
});

test("monitor persists a fresh context token and exposes its update time", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  await writeFile(join(root, "wechat-ilink.json"), JSON.stringify({
    session: { botToken: "token", baseUrl: "https://ilink.example" },
    binding: { userId: "owner@im.wechat", contextToken: "old-context" },
  }));
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (input) => {
      if (String(input).endsWith("/notifystart")) return new Response(JSON.stringify({ ret: 0 }));
      return new Response(JSON.stringify({
        ret: 0,
        msgs: [{ message_type: 1, from_user_id: "owner@im.wechat", context_token: "fresh-context" }],
      }));
    },
  });
  let state: { binding?: { contextToken?: string; contextUpdatedAt?: string } } = {};
  for (let attempt = 0; attempt < 20; attempt++) {
    state = JSON.parse(await readFile(join(root, "wechat-ilink.json"), "utf8"));
    if (state.binding?.contextToken === "fresh-context") break;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  expect(state.binding?.contextToken).toBe("fresh-context");
  expect(state.binding?.contextUpdatedAt).toEqual(expect.any(String));
  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toMatchObject({ context_updated_at: expect.any(String) });
  await bridge.stop();
});

test("bridge notifies iLink when starting and stopping its monitor", async () => {
  const root = join(tmpdir(), `agent-notify-ilink-${crypto.randomUUID()}`);
  roots.push(root);
  await mkdir(root, { recursive: true });
  await writeFile(join(root, "wechat-ilink.json"), JSON.stringify({
    session: { botToken: "token", baseUrl: "https://ilink.example" },
  }));
  const calls: string[] = [];
  const bridge = await createBridge({
    stateDir: root,
    fetchImpl: async (input) => {
      const url = String(input);
      calls.push(url);
      if (url.endsWith("/notifystart") || url.endsWith("/notifystop")) return new Response(JSON.stringify({ ret: 0 }));
      await new Promise((resolve) => setTimeout(resolve, 100));
      return new Response(JSON.stringify({ ret: 0, msgs: [] }));
    },
  });
  for (let attempt = 0; attempt < 20 && !calls.some((url) => url.endsWith("/notifystart")); attempt++) {
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  expect(calls.some((url) => url.endsWith("/notifystart"))).toBe(true);
  await bridge.stop();
  expect(calls.some((url) => url.endsWith("/notifystop"))).toBe(true);
});
