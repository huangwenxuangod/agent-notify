export function applyChannelToggle<
  K extends string,
  T extends { Remote: Record<K, { Enabled: boolean }> },
>(config: T, key: K, enabled: boolean): T {
  const next = structuredClone(config);
  next.Remote[key].Enabled = enabled;
  return next;
}
