import { describe, expect, test } from "bun:test";
import { classifyLog, filterLogs } from "./logs";

describe("log classification", () => {
  test("marks failures as errors", () => {
    expect(classifyLog("dispatch error event=run_completed")).toBe("error");
    expect(classifyLog("bridge event record failed: connection refused")).toBe("error");
  });

  test("marks retries as warnings", () => {
    expect(classifyLog("retried 1 remote notification(s)")).toBe("warning");
    expect(classifyLog("fallback to local delivery")).toBe("warning");
  });

  test("keeps ordinary runtime lines informational", () => {
    expect(classifyLog("Agent Notify bridge listening")).toBe("info");
  });

  test("filters without changing source order", () => {
    const lines = ["ok", "retrying", "send failed"];
    expect(filterLogs(lines, "warning")).toEqual(["retrying"]);
    expect(filterLogs(lines, "all")).toEqual(lines);
  });
});
