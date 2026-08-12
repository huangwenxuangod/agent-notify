const test = require('node:test');
const assert = require('node:assert/strict');
const { resolveProxy } = require('../src/proxy');

// proxy-from-env 直接读 process.env，测试只能就地替换后还原。
function withEnv(vars, fn) {
  const saved = {};
  const keys = [
    'https_proxy', 'HTTPS_PROXY', 'http_proxy', 'HTTP_PROXY',
    'all_proxy', 'ALL_PROXY', 'no_proxy', 'NO_PROXY',
  ];
  for (const key of keys) {
    saved[key] = process.env[key];
    delete process.env[key];
  }
  Object.assign(process.env, vars);
  try {
    return fn();
  } finally {
    for (const key of keys) {
      if (saved[key] === undefined) delete process.env[key];
      else process.env[key] = saved[key];
    }
  }
}

const TARGET = 'https://github.com/hellolib/agent-notify/releases/download/v1.0.0/x.tar.gz';

test('returns empty when nothing is configured', () => {
  withEnv({}, () => {
    assert.equal(resolveProxy(TARGET), '');
  });
});

test('uses HTTPS_PROXY', () => {
  withEnv({ HTTPS_PROXY: 'http://corp:8080' }, () => {
    assert.equal(resolveProxy(TARGET), 'http://corp:8080');
  });
});

test('uses the lowercase form too', () => {
  withEnv({ https_proxy: 'http://corp:8080' }, () => {
    assert.equal(resolveProxy(TARGET), 'http://corp:8080');
  });
});

// NO_PROXY 排除 github 的用户此前是直连的，能装上。强制走代理会把一个
// 本来可用的场景改坏，所以这个排除必须被尊重。
test('honours NO_PROXY', () => {
  withEnv({ HTTPS_PROXY: 'http://corp:8080', NO_PROXY: 'github.com' }, () => {
    assert.equal(resolveProxy(TARGET), '');
  });
});

test('honours a wildcard NO_PROXY', () => {
  withEnv({ HTTPS_PROXY: 'http://corp:8080', NO_PROXY: '*' }, () => {
    assert.equal(resolveProxy(TARGET), '');
  });
});

// 真实环境里大小写两种写法常常同时存在（shell profile 设一个、桌面环境设另一个）。
// proxy-from-env 的 getEnv 是 lowercase || uppercase，小写优先——不知道这点
// 会以为设了 NO_PROXY 却不生效。固化下来免得日后重新踩。
test('lowercase env vars take precedence over uppercase', () => {
  withEnv({ https_proxy: 'http://lower:1', HTTPS_PROXY: 'http://upper:2' }, () => {
    assert.equal(resolveProxy(TARGET), 'http://lower:1');
  });

  withEnv({ HTTPS_PROXY: 'http://corp:8080', no_proxy: 'github.com', NO_PROXY: 'nothing.invalid' }, () => {
    assert.equal(resolveProxy(TARGET), '', '小写 no_proxy 应当胜出并排除 github.com');
  });
});

test('an empty proxy value is treated as unset', () => {
  withEnv({ HTTPS_PROXY: '' }, () => {
    assert.equal(resolveProxy(TARGET), '');
  });
});
