import { afterEach, expect, test } from "bun:test";
import { mkdir, readFile, rm } from "node:fs/promises";
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
    last_delivery_error: "prepare failed",
  });
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
    base_info: { channel_version: "2.4.6" },
  });
  expect(headers?.get("iLink-App-Id")).toBe("bot");
  expect(headers?.get("iLink-App-ClientVersion")).toBe("132102");

  const health = await bridge.handle(new Request("http://localhost/health"));
  expect(await health.json()).toEqual({
    logged_in: true,
    bound: true,
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
