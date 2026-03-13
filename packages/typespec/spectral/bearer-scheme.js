export default function bearerScheme(targetVal, _opts, context) {
  const scheme = targetVal?.components?.securitySchemes?.BearerAuth;
  if (scheme && scheme.type === 'http' && String(scheme.scheme).toLowerCase() === 'bearer') {
    return [];
  }

  return [
    {
      message: 'components.securitySchemes.BearerAuth must exist with type=http and scheme=bearer',
      path: [...context.path, 'components', 'securitySchemes', 'BearerAuth'],
    },
  ];
}
