import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const RAW_BASE = 'https://raw.githubusercontent.com/hellolib/agent-notify/main/';
const BLOB_BASE = 'https://github.com/hellolib/agent-notify/blob/main/';

// rewriteAssetPaths converts repo-relative asset references into absolute URLs
// so images and doc links resolve on the npm package page.
export function rewriteAssetPaths(md: string): string {
  return md
    // markdown ![alt](assist/...) and [text](assist/...)
    .replace(/(\]\()(assist\/)/g, `$1${RAW_BASE}$2`)
    // HTML src="assist/..." / href="assist/..."
    .replace(/((?:src|href)=")(assist\/)/g, `$1${RAW_BASE}$2`)
    // the sibling Chinese README link (markdown + HTML)
    .replace(/(\]\()(README\.zh-CN\.md)/g, `$1${BLOB_BASE}$2`)
    .replace(/((?:src|href)=")(README\.zh-CN\.md)/g, `$1${BLOB_BASE}$2`);
}

export function syncReadme(srcPath: string, destPath: string): void {
  const md = fs.readFileSync(srcPath, 'utf8');
  fs.writeFileSync(destPath, rewriteAssetPaths(md));
}

if (import.meta.main) {
  const scriptDir = path.dirname(fileURLToPath(import.meta.url));
  const src = path.join(scriptDir, '..', '..', 'README.md');
  const dest = path.join(scriptDir, '..', 'README.md');
  syncReadme(src, dest);
  console.log(`Synced README: ${src} -> ${dest}`);
}
