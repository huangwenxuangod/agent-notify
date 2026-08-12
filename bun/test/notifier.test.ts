const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execSync } = require('node:child_process');
const {
  NOTIFIER_BUNDLE,
  notifierExePath,
  isNotifierInstalled,
  clearQuarantine,
  adHocSign,
} = require('../src/notifier');

const REAL_ZIP = path.join(__dirname, '..', '..', 'thirdparty', 'helper', 'mac', 'terminal-notifier-2.0.0.zip');

test('NOTIFIER_BUNDLE is terminal-notifier.app', () => {
  assert.equal(NOTIFIER_BUNDLE, 'terminal-notifier.app');
});

test('notifierExePath points to ~/.agent-notify bundle', () => {
  const p = notifierExePath();
  assert.match(p, /\.agent-notify.*terminal-notifier\.app\/Contents\/MacOS\/terminal-notifier/);
});

test('clearQuarantine removes quarantine attribute', (t) => {
  if (process.platform !== 'darwin') {
    t.skip('darwin only');
    return;
  }
  const tmp = path.join(os.tmpdir(), `an-xattr-${process.pid}`);
  fs.writeFileSync(tmp, 'x');
  t.after(() => fs.rmSync(tmp, { force: true }));
  execSync(`xattr -w com.apple.quarantine "test" "${tmp}"`);
  assert.match(execSync(`xattr "${tmp}" 2>/dev/null || true`).toString(), /quarantine/);
  clearQuarantine(tmp);
  const after = execSync(`xattr "${tmp}" 2>/dev/null || true`).toString().trim();
  assert.doesNotMatch(after, /quarantine/);
});

test('adHocSign signs the bundle (darwin)', (t) => {
  if (process.platform !== 'darwin') {
    t.skip('darwin only');
    return;
  }
  if (!fs.existsSync(REAL_ZIP)) {
    t.skip('real zip not present');
    return;
  }
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'an-sign-'));
  t.after(() => fs.rmSync(tmpRoot, { recursive: true, force: true }));
  const bundle = path.join(tmpRoot, 'terminal-notifier.app');
  execSync(`unzip -o -q "${REAL_ZIP}" -d "${tmpRoot}"`);
  adHocSign(bundle);
  const r = execSync(`codesign -dv "${bundle}" 2>&1 || true`).toString();
  assert.match(r, /adhoc/, 'bundle should be ad-hoc signed');
});
