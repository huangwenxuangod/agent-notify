import { REPO_OWNER, REPO_NAME, type PlatformTarget } from './constants.ts';

export function buildTag(version: string): string {
  return version.startsWith('v') ? version : `v${version}`;
}

export function buildAssetName(version: string, target: PlatformTarget): string {
  const tag = buildTag(version);
  return `agent-notify-${tag}-${target.goos}-${target.goarch}.tar.gz`;
}

export function buildDownloadUrl(version: string, assetName: string): string {
  const tag = buildTag(version);
  return `https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${assetName}`;
}
