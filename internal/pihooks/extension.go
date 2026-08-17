package pihooks

import (
	"encoding/json"
	"fmt"
)

func extensionSource(binary string) string {
	encoded, _ := json.Marshal(binary)
	return fmt.Sprintf(`// %s
// Forward Pi lifecycle events to Agent Notify without blocking Pi.
import { spawn } from "node:child_process";

const agentNotifyBinary = %s;
let sessionID = "";
let activeRun = false;
let pendingEnd;
let pendingTimer;
const completionQuietMs = 5000;

function stringValue(value) {
  return typeof value === "string" ? value : "";
}

function lastAssistantDetails(messages) {
  if (!Array.isArray(messages)) return { message: "", error: "", stopReason: "" };
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const item = messages[index];
    if (item?.role !== "assistant") continue;
    const content = Array.isArray(item.content)
      ? item.content.filter((part) => part?.type === "text").map((part) => stringValue(part.text)).join("\n")
      : stringValue(item.content);
    return { message: content, error: stringValue(item.errorMessage), stopReason: stringValue(item.stopReason) };
  }
  return { message: "", error: "", stopReason: "" };
}

function forward(eventName, event, ctx) {
  const value = event ?? {};
  const context = ctx ?? {};
  const assistant = lastAssistantDetails(value.messages);
  const payload = {
    event: eventName,
    session_id: stringValue(value.sessionId) || stringValue(value.session_id) || sessionID,
    turn_id: stringValue(value.turnId) || stringValue(value.turn_id),
    run_id: stringValue(value.runId) || stringValue(value.run_id),
    cwd: stringValue(context.cwd),
    reason: stringValue(value.reason),
    message: stringValue(value.message) || stringValue(value.lastMessage) || assistant.message,
    error: stringValue(value.errorMessage) || stringValue(value.error) || assistant.error,
    stop_reason: stringValue(value.stopReason) || stringValue(value.stop_reason) || assistant.stopReason,
  };
  const child = spawn(agentNotifyBinary, ["handle-pi-hook"], {
    detached: true,
    stdio: ["pipe", "ignore", "ignore"],
  });
  child.on("error", () => {});
  child.stdin.end(JSON.stringify(payload));
  child.unref();
}

function flushPendingEnd() {
  if (!pendingEnd) return;
  const next = pendingEnd;
  pendingEnd = undefined;
  if (pendingTimer) {
    clearTimeout(pendingTimer);
    pendingTimer = undefined;
  }
  forward("agent_end", next.event, next.ctx);
}

function scheduleAgentEnd(event, ctx) {
  pendingEnd = { event, ctx };
  if (pendingTimer) clearTimeout(pendingTimer);
  pendingTimer = setTimeout(flushPendingEnd, completionQuietMs);
}

export default function agentNotifyPiExtension(pi) {
  pi.on("session_start", (event, ctx) => {
    const value = event ?? {};
    sessionID = stringValue(value.sessionFile) || stringValue(value.session_file) || stringValue(ctx?.cwd) + ":" + process.pid;
	activeRun = false;
    forward("session_start", value, ctx);
  });

  pi.on("agent_start", () => {
    flushPendingEnd();
    activeRun = true;
  });

  // Pi v0.80.x exposes agent_end before its retry decision to extensions. A
  // quiet window keeps only the last terminal result without blocking Pi.
  pi.on("agent_end", (event, ctx) => {
    activeRun = false;
    scheduleAgentEnd(event, ctx);
  });

  // A normal print-mode exit first flushes its completion. A shutdown while a
  // run is active is an interruption; session switches/reloads stay silent.
  pi.on("session_shutdown", (event, ctx) => {
	if (pendingEnd) {
	  flushPendingEnd();
	} else if (activeRun && stringValue(event?.reason).toLowerCase() === "quit") {
      forward("session_shutdown", event, ctx);
    }
  });
}
`, marker, encoded)
}

// BuildExtension returns the complete Pi TypeScript extension source.
func BuildExtension(binary string) string { return extensionSource(binary) }
