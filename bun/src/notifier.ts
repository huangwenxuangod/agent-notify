// terminal-notifier 本地 bundle 的后处理工具。
// macOS 上点击通知跳转依赖 terminal-notifier。它随 agent-notify 的 tar.gz 一起下载
// （release 打包时已打进 darwin 包），install.js 解压 tar.gz 时提取到 INSTALL_DIR。
// 本模块负责 bundle 就绪后的 quarantine 清除与 ad-hoc 签名。
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { INSTALL_DIR } from './constants.ts';

export const NOTIFIER_BUNDLE = 'terminal-notifier.app';

// notifierExePath 返回本地 terminal-notifier 可执行文件绝对路径（不检查存在性）。
export function notifierExePath(): string {
  return path.join(INSTALL_DIR, NOTIFIER_BUNDLE, 'Contents', 'MacOS', 'terminal-notifier');
}

// isNotifierInstalled 检查本地 bundle 是否已就绪。
export function isNotifierInstalled(): boolean {
  return fs.existsSync(notifierExePath());
}

// clearQuarantine 递归清除 bundle 的 com.apple.quarantine 标记。
// 仅 macOS 需要；用 xattr -dr 精确删除 quarantine，不碰 com.apple.provenance（系统保留）。
export function clearQuarantine(bundlePath: string): void {
  if (process.platform !== 'darwin') return;
  try {
    execFileSync('xattr', ['-dr', 'com.apple.quarantine', bundlePath], { stdio: 'ignore' });
  } catch {
    // xattr 不可用或无 quarantine 时忽略，不影响主流程
  }
}

// adHocSign 对 bundle 做 ad-hoc 签名。未签名 bundle 经 Rosetta 在 Codex 等宿主下
// 初始化 NSApplication 偶发崩溃；ad-hoc 签名后 Mac 终端与 Codex 终端均稳定可弹通知。
// 签名失败仅警告，不阻断主流程（未签名时 Mac 终端仍可用）。
export function adHocSign(bundlePath: string): void {
  if (process.platform !== 'darwin') return;
  try {
    execFileSync('codesign', ['-s', '-', '--force', '--deep', bundlePath], { stdio: 'ignore' });
  } catch {
    signViaTemporaryCopy(bundlePath);
  }
}

function signViaTemporaryCopy(bundlePath: string): void {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'agent-notify-sign-'));
  const tmpBundle = path.join(tmpRoot, NOTIFIER_BUNDLE);

  try {
    fs.cpSync(bundlePath, tmpBundle, { recursive: true });
    execFileSync('codesign', ['-s', '-', '--force', '--deep', tmpBundle], { stdio: 'ignore' });
    fs.rmSync(bundlePath, { recursive: true, force: true });
    fs.cpSync(tmpBundle, bundlePath, { recursive: true });
  } catch {
    // codesign 不可用或替换失败时忽略，不影响主流程
  } finally {
    fs.rmSync(tmpRoot, { recursive: true, force: true });
  }
}
