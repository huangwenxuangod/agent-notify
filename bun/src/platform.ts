import type { PlatformTarget } from './constants.ts';

export function getPlatformTarget({
  platform = process.platform,
  arch = process.arch,
}: { platform?: string; arch?: string } = {}): PlatformTarget {
  const goosMap: Record<string, PlatformTarget['goos']> = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };

  const goarchMap: Record<string, PlatformTarget['goarch']> = {
    x64: 'amd64',
    arm64: 'arm64',
  };

  const goos = goosMap[platform];
  const goarch = goarchMap[arch];

  if (!goos || !goarch) {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }

  return {
    goos,
    goarch,
    ext: goos === 'windows' ? '.exe' : '',
  };
}
