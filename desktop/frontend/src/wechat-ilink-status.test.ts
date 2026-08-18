import { expect, test } from "bun:test";

import { normalizeWechatIlinkStatus } from "./wechat-ilink-status";

test("normalizes the snake_case status returned by the Wails bridge", () => {
  expect(normalizeWechatIlinkStatus({
    logged_in: false,
    bound: false,
    qr_data_url: "data:image/png;base64,qr",
    status: "wait",
    last_delivery_at: "2026-08-18T01:00:00Z",
  })).toEqual({
    LoggedIn: false,
    Bound: false,
    QRDataURL: "data:image/png;base64,qr",
    Status: "wait",
    LastDeliveryAt: "2026-08-18T01:00:00Z",
  });
});
