export default function appSecurity(targetVal, _opts, context) {
  if (!targetVal || typeof targetVal !== 'object') {
    return [];
  }

  const results = [];
  for (const [pathKey, pathItem] of Object.entries(targetVal)) {
    // /api/v1/auth/* is public (no bearer required)
    if (pathKey.startsWith('/api/v1/auth/')) {
      continue;
    }
    // /api/v1/status is a public status endpoint
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
