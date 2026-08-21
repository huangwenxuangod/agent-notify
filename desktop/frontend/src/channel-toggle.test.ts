import { expect, test } from "bun:test";
import { applyChannelToggle } from "./channel-toggle";

test("channel toggle preserves an explicit disabled state without an endpoint", () => {
  const config = { Remote: { Ntfy: { Enabled: true, TopicURL: "" } } };

  const next = applyChannelToggle(config, "Ntfy", false);

  expect(next.Remote.Ntfy.Enabled).toBe(false);
  expect(next.Remote.Ntfy.TopicURL).toBe("");
});
