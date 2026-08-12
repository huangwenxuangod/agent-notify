import path from 'node:path';
import os from 'node:os';

export const REPO_OWNER = 'hellolib';
export const REPO_NAME = 'agent-notify';
export const INSTALL_DIR = path.join(os.homedir(), '.agent-notify');
export const TMP_DIR = path.join(INSTALL_DIR, 'tmp');

export type PlatformTarget = {
  goos: 'darwin' | 'linux' | 'windows';
  goarch: 'amd64' | 'arm64';
  ext: '' | '.exe';
};

export function getInstalledBinaryPath(target: PlatformTarget): string {
  return path.join(INSTALL_DIR, `agent-notify${target.ext}`);
}
