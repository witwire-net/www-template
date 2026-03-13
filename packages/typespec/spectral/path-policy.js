export default function pathPolicy(targetVal, _opts, context) {
  if (!targetVal || typeof targetVal !== 'object') {
    return [];
  }

  const results = [];
  const allowedPathPattern = /^\/api\/v1(?:\/app)?\/.+/;

  for (const pathKey of Object.keys(targetVal)) {
    if (allowedPathPattern.test(pathKey)) {
      continue;
    }

    results.push({
      message: `path \`${pathKey}\` is outside the allowed /api/v1/* or /api/v1/app/* policy`,
      path: [...context.path, pathKey],
    });
  }

  return results;
}
