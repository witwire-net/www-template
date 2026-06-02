export default function appSecurity(targetVal, _opts, context) {
  if (!targetVal || typeof targetVal !== 'object') {
    return [];
  }

  const results = [];
  for (const [pathKey, pathItem] of Object.entries(targetVal)) {
    // Product/Admin の login/setup/context refresh は bearer 前の public flow であり、Authorization header を refresh credential として扱わないため security 宣言を要求しない。
    if (isPublicAuthPath(pathKey)) {
      continue;
    }
    // Product status は load balancer / browser から認証なしで確認する public endpoint として維持する。
    if (pathKey === '/api/v1/status') {
      continue;
    }
    if (!pathKey.startsWith('/api/v1/')) {
      continue;
    }

    for (const [method, operation] of Object.entries(pathItem)) {
      if (!operation || typeof operation !== 'object') {
        continue;
      }

      const security = operation.security;
      const hasBearerSecurity =
        Array.isArray(security) &&
        security.some(
          (entry) => entry && Object.prototype.hasOwnProperty.call(entry, 'BearerAuth')
        );

      if (hasBearerSecurity) {
        continue;
      }

      results.push({
        message: `${method.toUpperCase()} ${pathKey} must declare BearerAuth security`,
        path: [...context.path, pathKey, method],
      });
    }
  }

  return results;
}

function isPublicAuthPath(pathKey) {
  // Product/Admin の pre-auth ceremony と context refresh だけを public とし、logout / reauth / current / passkey management は BearerAuth 必須にする。
  const publicAuthPaths = new Set([
    '/api/v1/auth/contexts/{authContextId}/refresh',
    '/api/v1/auth/operator-setup/finish',
    '/api/v1/auth/operator-setup/start',
    '/api/v1/auth/passkey/finish',
    '/api/v1/auth/passkey/register',
    '/api/v1/auth/passkey/register/start',
    '/api/v1/auth/passkey/start',
    '/api/v1/auth/recovery',
    '/api/v1/auth/recovery/consume',
    '/api/v1/auth/setup/finish',
    '/api/v1/auth/setup/start',
  ]);

  // 明示 list だけを public とし、将来の `/api/v1/auth/*` 追加が bearer 必須から漏れないよう fail-close にする。
  return publicAuthPaths.has(pathKey);
}
