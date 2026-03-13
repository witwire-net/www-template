export default function appSecurity(targetVal, _opts, context) {
  if (!targetVal || typeof targetVal !== 'object') {
    return [];
  }

  const results = [];
  for (const [pathKey, pathItem] of Object.entries(targetVal)) {
    if (!pathKey.startsWith('/api/v1/app/')) {
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
