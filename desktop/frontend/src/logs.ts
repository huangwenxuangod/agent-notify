export type LogLevel = "info" | "warning" | "error";

export function classifyLog(line: string): LogLevel {
  const value = line.toLowerCase();
  if (/(error|failed|failure|refused|timeout|unauthorized|forbidden)/.test(value)) {
    return "error";
  }
  if (/(warn|warning|retry|retried|fallback|skipped)/.test(value)) {
    return "warning";
  }
  return "info";
}

export function filterLogs(lines: string[], level: "all" | LogLevel): string[] {
  return level === "all" ? lines : lines.filter((line) => classifyLog(line) === level);
}
