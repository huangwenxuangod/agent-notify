import { getProxyForUrl } from 'proxy-from-env';

// proxy-from-env 读取的标准变量。它对「压根没配代理」和「配了但被 NO_PROXY
// 排除」都返回空串,靠这份清单把两者区分开。
const STANDARD_PROXY_ENV = [
  'https_proxy', 'HTTPS_PROXY',
  'http_proxy', 'HTTP_PROXY',
  'all_proxy', 'ALL_PROXY',
];

function hasStandardProxyEnv(env: NodeJS.ProcessEnv): boolean {
  return STANDARD_PROXY_ENV.some((key) => Boolean(env[key]));
}

// resolveProxy 给出下载 url 时应当使用的代理地址,没有则返回 ''。
//
// 交给 proxy-from-env 处理标准环境变量。它同时实现 NO_PROXY 的后缀、
// 通配与端口匹配语义，自己实现这套规则很容易出错。
export function resolveProxy(url: string, env = process.env): string {
  if (!hasStandardProxyEnv(env)) return '';
  return getProxyForUrl(url);
}
