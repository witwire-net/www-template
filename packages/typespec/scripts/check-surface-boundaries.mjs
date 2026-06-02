#!/usr/bin/env node

import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const packageRoot = fileURLToPath(new URL('..', import.meta.url));
const productRouteRoot = join(packageRoot, 'src', 'routes', 'v1', 'product');
const adminRouteRoot = join(packageRoot, 'src', 'routes', 'v1', 'admin');
const modelRoot = join(packageRoot, 'src', 'models');
const mainSourcePath = join(packageRoot, 'main.tsp');
const productOpenAPIArtifactPath = join(packageRoot, 'openapi', 'openapi.json');
const adminOpenAPIArtifactPath = join(packageRoot, 'openapi', 'admin.openapi.json');

// Product surface の TypeSpec source が Admin route namespace を import / 参照しないことを表す禁止語彙。
// Admin model は共有 read model として存在できるが、Admin.ApiV1 route namespace は Product artifact へ混入してはならない。
const productSurfaceForbiddenPatterns = [
  {
    pattern: /import\s+["'][^"']*routes\/v1\/admin\/[^"']*["'];?/u,
    reason: 'Product route source must not import Admin route files',
  },
  {
    pattern: /import\s+["'][^"']*v1\/admin\/[^"']*["'];?/u,
    reason: 'Product route source must not import Admin route namespace files',
  },
  {
    pattern: /\bAdmin\.ApiV1\b/u,
    reason: 'Product route source must not reference Admin.ApiV1 namespace',
  },
];

const explicitInputFiles = process.argv.slice(2);

// Step 1: CLI 引数がある場合は test fixture などの明示 path だけを検査し、通常実行では Product route tree 全体を検査する。
const inputFiles = explicitInputFiles.length > 0 ? explicitInputFiles : collectTypeSpecFiles(productRouteRoot);

// Step 2: 各 TypeSpec source を行単位で評価し、違反箇所を path:line 付きで蓄積する。
const violations = inputFiles.flatMap((filePath) => detectProductSurfaceBoundaryViolations(filePath));

// Step 3: 通常実行では生成済み Product/Admin OpenAPI も検査し、service artifact 分離と context refresh path の両 surface 生成を固定する。
if (explicitInputFiles.length === 0) {
  violations.push(...detectTypeSpecSourceStructureViolations());
  violations.push(...detectOpenAPISurfaceBoundaryViolations());
}

// Step 4: 1 件でも違反があれば stderr にすべて出し、contract lint の fail-closed な終了コードにする。
if (violations.length > 0) {
  for (const violation of violations) {
    console.error(violation);
  }
  process.exit(1);
}

/**
 * detectOpenAPISurfaceBoundaryViolations は生成済み Product/Admin OpenAPI の surface 分離を検査する。
 *
 * @returns {string[]} `path:line` 形式に揃えた違反 message。一覧が空なら generated OpenAPI の surface 境界は保たれている。
 * @throws {Error} generated OpenAPI artifact が読めない、または JSON として parse できない場合はそのまま送出する。
 */
function detectOpenAPISurfaceBoundaryViolations() {
  // Step 1: Product/Admin artifact をそれぞれ JSON として読み、path / tag / server / operationId を構造的に検査する。
  const productContract = readOpenAPIContract(productOpenAPIArtifactPath);
  const adminContract = readOpenAPIContract(adminOpenAPIArtifactPath);

  // Step 2: surface ごとの禁止 operation と必須 path を検査し、同じ relative refresh path を両 service に分離生成できているか確認する。
  return [
    ...detectForbiddenOpenAPITokens(productOpenAPIArtifactPath, productContract, adminSurfaceTokens(), '[API-CONTRACT-BE-S001] Product artifact must not expose Admin operations'),
    ...detectForbiddenOpenAPITokens(adminOpenAPIArtifactPath, adminContract, productSurfaceTokens(), '[API-CONTRACT-BE-S002] Admin artifact must not expose Product operations'),
    ...detectContextRefreshPath(productOpenAPIArtifactPath, productContract, 'Product'),
    ...detectContextRefreshPath(adminOpenAPIArtifactPath, adminContract, 'Admin'),
    ...detectForbiddenAdminPath(productOpenAPIArtifactPath, productContract),
    ...detectForbiddenAdminPath(adminOpenAPIArtifactPath, adminContract),
    ...detectSeparatedServers(productContract, adminContract),
    ...detectCredentialModeResponseShapes(productOpenAPIArtifactPath, productContract, productCredentialModeResponseContracts()),
    ...detectCredentialModeResponseShapes(adminOpenAPIArtifactPath, adminContract, adminCredentialModeResponseContracts()),
    ...detectSubjectFieldContracts(productOpenAPIArtifactPath, productContract, productSubjectResponseContracts()),
    ...detectSubjectFieldContracts(adminOpenAPIArtifactPath, adminContract, adminSubjectResponseContracts()),
    ...detectCookieClearCommandShape(productOpenAPIArtifactPath, productContract, 'CookieClearCommand'),
    ...detectCookieClearCommandShape(adminOpenAPIArtifactPath, adminContract, 'WWWTemplate.CookieClearCommand'),
  ];
}

/**
 * detectTypeSpecSourceStructureViolations は TypeSpec source の concept module / route split を検査する。
 *
 * @returns {string[]} `path:line` 形式に揃えた違反 message。一覧が空なら source organization は期待どおりである。
 */
function detectTypeSpecSourceStructureViolations() {
  // Step 1: main.tsp が common import として concept modules だけを読み、surface 固有 catch-all model を読まないことを固定する。
  const violations = [];
  const mainSource = readFileSync(mainSourcePath, 'utf8');
  const mainImportContracts = [
    ['[API-CONTRACT-BE-S017]', './src/models/auth/primitives.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/auth/webauthn.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/auth/sessions.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/auth/refresh.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/auth/logout.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/auth/recovery.tsp'],
    ['[API-CONTRACT-BE-S018]', './src/models/accounts/read_models.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/operators/profiles.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/operators/setup.tsp'],
    ['[API-CONTRACT-BE-S017]', './src/models/operators/authorization.tsp'],
    ['[API-CONTRACT-BE-S014]', './src/routes/v1/product/auth.tsp'],
    ['[API-CONTRACT-BE-S014]', './src/routes/v1/admin/auth.tsp'],
  ];
  for (const [scenarioId, importPath] of mainImportContracts) {
    if (!mainSource.includes(`import "${importPath}";`)) {
      violations.push(`${relative(process.cwd(), mainSourcePath)}:1: ${scenarioId} main.tsp must import ${importPath}`);
    }
  }

  // Step 2: 旧 catch-all model と旧 route root が残ると二重 owner になるため、存在そのものを拒否する。
  for (const [scenarioId, filePath] of [
    ['[API-CONTRACT-BE-S017]', join(modelRoot, 'admin.tsp')],
    ['[API-CONTRACT-BE-S017]', join(modelRoot, 'auth.tsp')],
    ['[API-CONTRACT-BE-S018]', join(modelRoot, 'account_settings.tsp')],
    ['[API-CONTRACT-BE-S014]', join(packageRoot, 'src', 'routes', 'admin-v1', 'auth.tsp')],
    ['[API-CONTRACT-BE-S014]', join(packageRoot, 'src', 'routes', 'admin-v1', 'accounts.tsp')],
    ['[API-CONTRACT-BE-S014]', join(packageRoot, 'src', 'routes', 'admin-v1', '_namespace.tsp')],
    ['[API-CONTRACT-BE-S014]', join(packageRoot, 'src', 'routes', 'v1', 'auth.tsp')],
  ]) {
    if (existsSync(filePath)) {
      violations.push(`${relative(process.cwd(), filePath)}:1: ${scenarioId} legacy catch-all or unsplit route source must not remain`);
    }
  }

  // Step 3: Product/Admin route root が存在し、それぞれの route source が route-only service boundary を表すことを固定する。
  for (const [scenarioId, directoryPath] of [
    ['[API-CONTRACT-BE-S014]', productRouteRoot],
    ['[API-CONTRACT-BE-S014]', adminRouteRoot],
  ]) {
    if (!existsSync(directoryPath)) {
      violations.push(`${relative(process.cwd(), directoryPath)}:1: ${scenarioId} route root must exist`);
    }
  }

  // Step 4: TypeSpec source 全体から forbidden discriminator / legacy token / CSRF contract fields を排除し、service artifact boundary で文脈を決める設計を守る。
  const sourceFiles = collectTypeSpecFiles(join(packageRoot, 'src'));
  for (const filePath of sourceFiles) {
    violations.push(...detectForbiddenSourceTokens(filePath));
  }

  // Step 5: shared auth envelope と account concept の owner file を検査し、route DTO や surface model への再定義を拒否する。
  violations.push(...detectConceptOwnerViolations(sourceFiles));
  violations.push(...detectLegacyRouteSourceViolations());

  return violations;
}

/**
 * detectConceptOwnerViolations は shared auth/account concept が決められた owner file 以外で再定義されていないか検査する。
 *
 * @param {string[]} sourceFiles 検査対象 TypeSpec source file 一覧。
 * @returns {string[]} 違反 message。一覧が空なら concept owner は一意である。
 */
function detectConceptOwnerViolations(sourceFiles) {
  // Step 1: field / model / enum ごとの canonical owner を宣言し、S013/S018 の単一定義条件を機械的に固定する。
  const ownerContracts = [
    ...authEnvelopeOwnerContracts(),
    ...accountOwnerContracts(),
    ...operatorOwnerContracts(),
  ];

  // Step 2: TypeSpec source を実行行だけに正規化し、comment 中の説明文で誤検知しないようにする。
  const indexedSources = sourceFiles.map((filePath) => ({
    filePath,
    lines: readFileSync(filePath, 'utf8').split(/\r?\n/u).map(stripLineComment),
  }));

  // Step 3: 各 concept の定義行が owner file だけに存在し、かつ必ず 1 件以上存在することを検査する。
  return ownerContracts.flatMap((contract) => {
    const matches = findConceptMatches(indexedSources, contract);
    if (matches.length === 0) {
      return [`${relative(process.cwd(), contract.ownerPath)}:1: ${contract.scenarioId} missing canonical definition for ${contract.name}`];
    }
    return matches
      .filter((match) => match.filePath !== contract.ownerPath)
      .map((match) => `${relative(process.cwd(), match.filePath)}:${match.lineNumber}: ${contract.scenarioId} ${contract.name} must be defined only in ${relative(process.cwd(), contract.ownerPath)}`);
  });
}

/**
 * authEnvelopeOwnerContracts は shared auth envelope と primitive の owner 契約を返す。
 *
 * @returns {Array<{name: string, scenarioId: string, ownerPath: string, pattern: RegExp}>} owner contract 一覧。
 */
function authEnvelopeOwnerContracts() {
  // Step 1: Product/Admin 両 artifact が参照する response field は shared envelope owner file でだけ定義する。
  const sessionsPath = join(modelRoot, 'auth', 'sessions.tsp');
  const primitivesPath = join(modelRoot, 'auth', 'primitives.tsp');
  const refreshPath = join(modelRoot, 'auth', 'refresh.tsp');
  return [
    typeOwner('CookieAuthEnvelope', '[API-CONTRACT-BE-S013]', sessionsPath),
    typeOwner('BearerAuthEnvelope', '[API-CONTRACT-BE-S013]', sessionsPath),
    typeOwner('CookieClearCommand', '[API-CONTRACT-BE-S013]', primitivesPath),
    typeOwner('ContextIndexUpdateHint', '[API-CONTRACT-BE-S013]', primitivesPath),
    typeOwner('AuthFailureResponse', '[API-CONTRACT-BE-S013]', primitivesPath),
    typeOwner('CredentialMode', '[API-CONTRACT-BE-S013]', primitivesPath),
    typeOwner('BearerContextRefreshRequest', '[API-CONTRACT-BE-S019]', refreshPath),
  ];
}

/**
 * accountOwnerContracts は account read/create concept の owner 契約を返す。
 *
 * @returns {Array<{name: string, scenarioId: string, ownerPath: string, pattern: RegExp}>} owner contract 一覧。
 */
function accountOwnerContracts() {
  // Step 1: Admin route も Product account concept を同じ owner から参照し、surface 接頭辞 DTO の再定義を拒否する。
  const accountPath = join(modelRoot, 'accounts', 'read_models.tsp');
  return [
    typeOwner('AccountSummary', '[API-CONTRACT-BE-S018]', accountPath),
    typeOwner('CreateAccountRequest', '[API-CONTRACT-BE-S018]', accountPath),
    typeOwner('AccountListResponse', '[API-CONTRACT-BE-S018]', accountPath),
    typeOwner('AccountDetailResponse', '[API-CONTRACT-BE-S018]', accountPath),
    typeOwner('CreateAccountResponse', '[API-CONTRACT-BE-S018]', accountPath),
  ];
}

/**
 * operatorOwnerContracts は operator profile/setup/authorization concept の owner 契約を返す。
 *
 * @returns {Array<{name: string, scenarioId: string, ownerPath: string, pattern: RegExp}>} owner contract 一覧。
 */
function operatorOwnerContracts() {
  // Step 1: operator profile / setup / authorization の owner file を分け、Admin catch-all model への再統合を防ぐ。
  return [
    typeOwner('AdminOperatorProfile', '[API-CONTRACT-BE-S017]', join(modelRoot, 'operators', 'profiles.tsp')),
    typeOwner('AdminOperatorSessionResponse', '[API-CONTRACT-BE-S013]', join(modelRoot, 'operators', 'setup.tsp')),
    typeOwner('AdminContextRefreshResponse', '[API-CONTRACT-BE-S013]', join(modelRoot, 'operators', 'setup.tsp')),
    typeOwner('AdminAuthorizationBoundary', '[API-CONTRACT-BE-S017]', join(modelRoot, 'operators', 'authorization.tsp')),
  ];
}

/**
 * typeOwner は model / enum / union / scalar 定義の owner contract を生成する。
 *
 * @param {string} typeName type 名。
 * @param {string} scenarioId scenario ID。
 * @param {string} ownerPath owner file path。
 * @returns {{name: string, scenarioId: string, ownerPath: string, pattern: RegExp}} owner contract。
 */
function typeOwner(typeName, scenarioId, ownerPath) {
  // Step 1: TypeSpec の top-level type declaration だけを検査し、参照箇所や description の語彙を owner と誤認しない。
  return { name: typeName, scenarioId, ownerPath, pattern: new RegExp(`^\\s*(model|enum|union|scalar)\\s+${typeName}\\b`, 'u') };
}

/**
 * findConceptMatches は owner contract に一致する source 行を列挙する。
 *
 * @param {Array<{filePath: string, lines: string[]}>} indexedSources source file と実行行。
 * @param {{pattern: RegExp}} contract owner contract。
 * @returns {Array<{filePath: string, lineNumber: number}>} 一致箇所一覧。
 */
function findConceptMatches(indexedSources, contract) {
  // Step 1: 全 source / 全行を走査し、owner 以外の duplicate を path:line で報告できる形にする。
  const matches = [];
  for (const { filePath, lines } of indexedSources) {
    for (const [index, line] of lines.entries()) {
      if (contract.pattern.test(line)) {
        matches.push({ filePath, lineNumber: index + 1 });
      }
    }
  }
  return matches;
}

/**
 * detectLegacyRouteSourceViolations は旧 route tree に tracked source が戻っていないか検査する。
 *
 * @returns {string[]} 違反 message。一覧が空なら旧 route source は存在しない。
 */
function detectLegacyRouteSourceViolations() {
  // Step 1: 空 directory は git 管理されないため許容し、`.tsp` source が残る場合だけ route split 違反として扱う。
  const legacyRoots = [join(packageRoot, 'src', 'routes', 'admin-v1'), join(packageRoot, 'src', 'routes', 'v1')];
  const legacyFiles = legacyRoots.flatMap((rootPath) => collectTypeSpecFilesIfExists(rootPath))
    .filter((filePath) => !filePath.startsWith(`${productRouteRoot}/`) && !filePath.startsWith(`${adminRouteRoot}/`));
  return legacyFiles.map((filePath) => `${relative(process.cwd(), filePath)}:1: [API-CONTRACT-BE-S014] route source must live under routes/v1/product or routes/v1/admin`);
}

/**
 * collectTypeSpecFilesIfExists は directory が存在する場合だけ `.tsp` source を列挙する。
 *
 * @param {string} directory 検査対象 directory。
 * @returns {string[]} TypeSpec file path 一覧。directory がなければ空配列。
 */
function collectTypeSpecFilesIfExists(directory) {
  // Step 1: 移行後の空 / 不在 directory を許容し、source file がある場合だけ詳細検査に回す。
  if (!existsSync(directory)) {
    return [];
  }
  return collectTypeSpecFiles(directory);
}

/**
 * detectForbiddenSourceTokens は TypeSpec source に禁止済みの discriminator / legacy field が残っていないか検査する。
 *
 * @param {string} filePath 検査する TypeSpec source file。
 * @returns {string[]} 違反 message。一覧が空なら禁止 token はない。
 */
function detectForbiddenSourceTokens(filePath) {
  // Step 1: source を行単位で読み、comment ではなく実行 contract に残った legacy field だけを検出する。
  const source = readFileSync(filePath, 'utf8');
  const lines = source.split(/\r?\n/u);
  const forbiddenTokenContracts = [
    { pattern: /\bAuthContextIdentityKind\b/u, scenarioId: '[API-CONTRACT-BE-S016]', reason: 'AuthContextIdentityKind must not be a contract discriminator' },
    { pattern: /\bidentityKind\b/u, scenarioId: '[API-CONTRACT-BE-S016]', reason: 'identityKind must not be a contract discriminator' },
    { pattern: /\bprincipal\.kind\b/u, scenarioId: '[API-CONTRACT-BE-S016]', reason: 'principal.kind must not be required by auth context payloads' },
    { pattern: /\boperatorAccessToken\b/u, scenarioId: '[API-CONTRACT-BE-S015]', reason: 'Admin token field must be accessToken' },
    { pattern: /X-CSRF-Token/u, scenarioId: '[API-CONTRACT-BE-S019]', reason: 'protected routes use Authorization Bearer and must not require CSRF contract headers' },
  ];
  const violations = [];
  for (const [index, line] of lines.entries()) {
    const executableLine = stripLineComment(line);
    for (const { pattern, scenarioId, reason } of forbiddenTokenContracts) {
      if (pattern.test(executableLine)) {
        violations.push(`${relative(process.cwd(), filePath)}:${index + 1}: ${scenarioId} ${reason}`);
      }
    }
  }

  return violations;
}

/**
 * readOpenAPIContract は generated OpenAPI artifact を JSON object として読み込む。
 *
 * @param {string} filePath 読み込む OpenAPI artifact の絶対 path。
 * @returns {{paths?: Record<string, unknown>, tags?: Array<{name?: string}>, servers?: Array<{url?: string}>}} 検査に必要な OpenAPI fields。
 */
function readOpenAPIContract(filePath) {
  // Step 1: artifact 全体を UTF-8 text として読み、JSON parse 後の構造検査に渡す。
  const source = readFileSync(filePath, 'utf8');
  return JSON.parse(source);
}

/**
 * detectForbiddenOpenAPITokens は operationId と tag に別 surface の語彙が混入していないか検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{paths?: Record<string, Record<string, {operationId?: string, tags?: string[]}>>, tags?: Array<{name?: string}>}} contract OpenAPI contract object。
 * @param {{operationIds: Set<string>, tags: Set<string>}} forbiddenTokens 禁止 operationId と tag。
 * @param {string} reason 違反理由として出力する説明。
 * @returns {string[]} 違反 message。一覧が空なら混入はない。
 */
function detectForbiddenOpenAPITokens(artifactPath, contract, forbiddenTokens, reason) {
  // Step 1: top-level tag を先に検査し、operation がなくても分類 metadata だけが混入した状態を拒否する。
  const violations = [];
  for (const tag of contract.tags ?? []) {
    if (forbiddenTokens.tags.has(tag.name)) {
      violations.push(`${relative(process.cwd(), artifactPath)}:1: ${reason}: tag ${tag.name}`);
    }
  }

  // Step 2: path item の HTTP operation を走査し、operationId と operation tag の混入を拒否する。
  for (const [pathKey, pathItem] of Object.entries(contract.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem ?? {})) {
      if (forbiddenTokens.operationIds.has(operation?.operationId)) {
        violations.push(`${relative(process.cwd(), artifactPath)}:1: ${reason}: ${method.toUpperCase()} ${pathKey} operationId ${operation.operationId}`);
      }
      for (const tag of operation?.tags ?? []) {
        if (forbiddenTokens.tags.has(tag)) {
          violations.push(`${relative(process.cwd(), artifactPath)}:1: ${reason}: ${method.toUpperCase()} ${pathKey} tag ${tag}`);
        }
      }
    }
  }

  return violations;
}

/**
 * detectContextRefreshPath は context-scoped refresh path が対象 artifact にだけ存在することを検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{paths?: Record<string, Record<string, unknown>>}} contract OpenAPI contract object。
 * @param {string} surfaceName failure message に含める surface 名。
 * @returns {string[]} 違反 message。一覧が空なら context refresh route は生成されている。
 */
function detectContextRefreshPath(artifactPath, contract, surfaceName) {
  // Step 1: Product/Admin の同一 relative path が各 artifact に生成され、post operation を持つことを検査する。
  const contextRefreshPath = '/api/v1/auth/contexts/{authContextId}/refresh';
  const pathItem = contract.paths?.[contextRefreshPath];
  if (pathItem?.post) {
    return [];
  }

  return [`${relative(process.cwd(), artifactPath)}:1: [API-CONTRACT-BE-S010] ${surfaceName} artifact must include POST ${contextRefreshPath}`];
}

/**
 * detectForbiddenAdminPath は `/api/admin/*` が generated OpenAPI に存在しないことを検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{paths?: Record<string, unknown>}} contract OpenAPI contract object。
 * @returns {string[]} 違反 message。一覧が空なら legacy Admin BFF path は存在しない。
 */
function detectForbiddenAdminPath(artifactPath, contract) {
  // Step 1: 全 path key を確認し、Product/Admin とも Admin origin の `/api/v1/*` だけを使う方針を守る。
  return Object.keys(contract.paths ?? {})
    .filter((pathKey) => pathKey.startsWith('/api/admin/'))
    .map((pathKey) => `${relative(process.cwd(), artifactPath)}:1: [API-CONTRACT-BE-S010] /api/admin/* path is forbidden: ${pathKey}`);
}

/**
 * detectSeparatedServers は Product/Admin OpenAPI の server host が別であることを検査する。
 *
 * @param {{servers?: Array<{url?: string}>}} productContract Product OpenAPI contract object。
 * @param {{servers?: Array<{url?: string}>}} adminContract Admin OpenAPI contract object。
 * @returns {string[]} 違反 message。一覧が空なら server origin は分離されている。
 */
function detectSeparatedServers(productContract, adminContract) {
  // Step 1: server URL を URL として parse し、相対 URL や host 不在を surface 分離違反として扱う。
  const productHost = readOpenAPIServerHost(productOpenAPIArtifactPath, productContract);
  const adminHost = readOpenAPIServerHost(adminOpenAPIArtifactPath, adminContract);
  if (productHost !== adminHost) {
    return [];
  }

  return [`${relative(process.cwd(), productOpenAPIArtifactPath)}:1: [API-CONTRACT-BE-S003] Product and Admin OpenAPI servers must use distinct hosts`];
}

/**
 * detectCredentialModeResponseShapes は Cookie/Bearer mode の response DTO が secret と browser hint を正しく分離していることを検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{components?: {schemas?: Record<string, {properties?: Record<string, unknown>, required?: string[]}>}}} contract OpenAPI contract object。
 * @param {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} responseContracts 検査する response schema 契約。
 * @returns {string[]} 違反 message。一覧が空なら credential mode ごとの DTO 分離は保たれている。
 */
function detectCredentialModeResponseShapes(artifactPath, contract, responseContracts) {
  // Step 1: Product/Admin ごとの schema 契約を順に検査し、Cookie mode と Bearer mode の body shape が混ざらないことを固定する。
  return responseContracts.flatMap((responseContract) =>
    detectSchemaFieldContract(artifactPath, contract, responseContract),
  );
}

/**
 * detectSubjectFieldContracts は Product/Admin の subject field が explicit service field として生成されたか検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{components?: {schemas?: Record<string, {properties?: Record<string, unknown>, required?: string[]}>}}} contract OpenAPI contract object。
 * @param {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} responseContracts 検査する subject schema 契約。
 * @returns {string[]} 違反 message。一覧が空なら subject field は service 明示 field に揃っている。
 */
function detectSubjectFieldContracts(artifactPath, contract, responseContracts) {
  // Step 1: Product は account、Admin は operator を必須 field として検査し、principal wrapper や legacy flat subject を拒否する。
  return responseContracts.flatMap((responseContract) =>
    detectSchemaFieldContract(artifactPath, contract, responseContract),
  );
}

/**
 * detectSchemaFieldContract は OpenAPI component schema の必須 field と禁止 field を検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{components?: {schemas?: Record<string, {properties?: Record<string, unknown>, required?: string[]}>}}} contract OpenAPI contract object。
 * @param {{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}} responseContract schema field contract。
 * @returns {string[]} 違反 message。一覧が空なら schema は期待どおりである。
 */
function detectSchemaFieldContract(artifactPath, contract, responseContract) {
  // Step 1: schema が生成されていない場合は DTO 自体が失われているため、対象 scenario の契約違反として報告する。
  const schema = contract.components?.schemas?.[responseContract.schemaName];
  const displayPath = relative(process.cwd(), artifactPath);
  if (!schema) {
    return [`${displayPath}:1: ${responseContract.scenarioId} missing schema ${responseContract.schemaName}: ${responseContract.reason}`];
  }

  // Step 2: properties と required を両方検査し、field が optional 化された場合も browser / external client の契約破壊として検出する。
  const properties = schema.properties ?? {};
  const requiredFields = new Set(schema.required ?? []);
  const violations = [];
  for (const fieldName of responseContract.requiredFields) {
    if (!Object.prototype.hasOwnProperty.call(properties, fieldName)) {
      violations.push(`${displayPath}:1: ${responseContract.scenarioId} ${responseContract.schemaName} must expose ${fieldName}: ${responseContract.reason}`);
      continue;
    }
    if (!requiredFields.has(fieldName)) {
      violations.push(`${displayPath}:1: ${responseContract.scenarioId} ${responseContract.schemaName}.${fieldName} must be required: ${responseContract.reason}`);
    }
  }

  // Step 3: 禁止 field は presence だけで fail にし、Cookie mode の refresh token leak と Bearer mode の browser command leak を防ぐ。
  for (const fieldName of responseContract.forbiddenFields) {
    if (Object.prototype.hasOwnProperty.call(properties, fieldName)) {
      violations.push(`${displayPath}:1: ${responseContract.scenarioId} ${responseContract.schemaName} must not expose ${fieldName}: ${responseContract.reason}`);
    }
  }

  return violations;
}

/**
 * detectCookieClearCommandShape は Cookie clear command が authContextId と exact path field を表現できることを検査する。
 *
 * @param {string} artifactPath 検査対象 artifact の絶対 path。
 * @param {{components?: {schemas?: Record<string, {properties?: Record<string, unknown>, required?: string[]}>}}} contract OpenAPI contract object。
 * @param {string} schemaName surface ごとの CookieClearCommand schema 名。
 * @returns {string[]} 違反 message。一覧が空なら clear-cookie command は path/authContextId を表現できる。
 */
function detectCookieClearCommandShape(artifactPath, contract, schemaName) {
  // Step 1: S011 の exact Cookie Path と authContextId 表現を component schema 単位で固定する。
  return detectSchemaFieldContract(artifactPath, contract, {
    scenarioId: '[API-CONTRACT-BE-S011]',
    schemaName,
    requiredFields: ['authContextId', 'path', 'maxAge', 'httpOnly', 'secure', 'sameSite'],
    forbiddenFields: ['refreshToken', 'operatorRefreshToken'],
    reason: 'clear-cookie command must carry authContextId and exact refresh Cookie Path without plaintext refresh token',
  });
}

/**
 * productCredentialModeResponseContracts は Product response DTO の Cookie/Bearer 分離契約を返す。
 *
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Product schema 契約一覧。
 */
function productCredentialModeResponseContracts() {
  // Step 1: Product Cookie mode は browser hint / Cookie clear command を持ち refreshToken を持たない。Bearer mode はその逆を固定する。
  return [
    ...cookieModeResponseContracts('ProductCookieAuthSessionResponse'),
    ...cookieModeResponseContracts('ProductCookieContextRefreshResponse'),
    ...bearerModeResponseContracts('ProductBearerAuthSessionResponse', 'refreshToken'),
    ...bearerModeResponseContracts('ProductBearerContextRefreshResponse', 'refreshToken'),
  ];
}

/**
 * adminCredentialModeResponseContracts は Admin response DTO の Cookie/Bearer 分離契約を返す。
 *
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Admin schema 契約一覧。
 */
function adminCredentialModeResponseContracts() {
  // Step 1: Admin は Console Cookie mode と automation Bearer mode の両方を shared envelope で持ち、legacy operatorAccessToken を作らないことを固定する。
  return [
    ...cookieModeResponseContracts('AdminOperatorSessionResponse'),
    ...cookieModeResponseContracts('AdminContextRefreshResponse'),
    ...bearerModeResponseContracts('AdminBearerOperatorSessionResponse', 'refreshToken'),
    ...bearerModeResponseContracts('AdminBearerContextRefreshResponse', 'refreshToken'),
  ];
}

/**
 * productSubjectResponseContracts は Product response DTO の explicit account subject 契約を返す。
 *
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Product subject 契約一覧。
 */
function productSubjectResponseContracts() {
  // Step 1: Product response は account field で subject payload を返し、principal wrapper や flat accountId を response root に戻さない。
  return [
    ...subjectResponseContracts('ProductCookieAuthSessionResponse', 'account', ['principal', 'accountId', 'identityKind']),
    ...subjectResponseContracts('ProductBearerAuthSessionResponse', 'account', ['principal', 'accountId', 'identityKind']),
    ...subjectResponseContracts('ProductCookieContextRefreshResponse', 'account', ['principal', 'accountId', 'identityKind']),
    ...subjectResponseContracts('ProductBearerContextRefreshResponse', 'account', ['principal', 'accountId', 'identityKind']),
  ];
}

/**
 * adminSubjectResponseContracts は Admin response DTO の explicit operator subject 契約を返す。
 *
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Admin subject 契約一覧。
 */
function adminSubjectResponseContracts() {
  // Step 1: Admin response は operator field と shared accessToken を返し、operatorAccessToken / principal wrapper を公開しない。
  return [
    ...subjectResponseContracts('AdminOperatorSessionResponse', 'operator', ['principal', 'operatorId', 'operatorAccessToken', 'identityKind']),
    ...subjectResponseContracts('AdminBearerOperatorSessionResponse', 'operator', ['principal', 'operatorId', 'operatorAccessToken', 'identityKind']),
    ...subjectResponseContracts('AdminContextRefreshResponse', 'operator', ['principal', 'operatorId', 'operatorAccessToken', 'identityKind']),
    ...subjectResponseContracts('AdminBearerContextRefreshResponse', 'operator', ['principal', 'operatorId', 'operatorAccessToken', 'identityKind']),
  ];
}

/**
 * subjectResponseContracts は explicit subject field の共通 schema contract を作る。
 *
 * @param {string} schemaName 検査対象 schema 名。
 * @param {string} subjectField 必須にする subject field 名。
 * @param {string[]} forbiddenFields response root に出してはいけない legacy field 名。
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} subject field contract。
 */
function subjectResponseContracts(schemaName, subjectField, forbiddenFields) {
  // Step 1: generated consumer の可読性を保つため、subject は service-specific field で一段だけ表す。
  return [
    {
      scenarioId: '[API-CONTRACT-BE-S016]',
      schemaName,
      requiredFields: ['accessToken', 'authContextId', 'sessionId', subjectField],
      forbiddenFields,
      reason: 'auth response must expose explicit account/operator subject field and shared token field names',
    },
  ];
}

/**
 * cookieModeResponseContracts は Cookie mode response schema の共通契約を返す。
 *
 * @param {string} schemaName 検査対象 schema 名。
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Cookie mode schema 契約。
 */
function cookieModeResponseContracts(schemaName) {
  // Step 1: Cookie mode は browser 用 context index と Cookie clear command を表現し、平文 refresh token の露出を禁止する。
  return [
    {
      scenarioId: '[API-CONTRACT-BE-S015]',
      schemaName,
      requiredFields: ['credentialMode', 'authContextId', 'sessionId', 'accessToken', 'expiresAt', 'contextIndexUpdateHints', 'clearCookieCommands'],
      forbiddenFields: ['refreshToken', 'operatorRefreshToken', 'operatorAccessToken'],
      reason: 'cookie mode response must expose browser cleanup hints without plaintext refresh token',
    },
  ];
}

/**
 * bearerModeResponseContracts は Bearer mode response schema の共通契約を返す。
 *
 * @param {string} schemaName 検査対象 schema 名。
 * @param {string} refreshTokenField Bearer mode で返す plaintext refresh token field 名。
 * @returns {Array<{scenarioId: string, schemaName: string, requiredFields: string[], forbiddenFields: string[], reason: string}>} Bearer mode schema 契約。
 */
function bearerModeResponseContracts(schemaName, refreshTokenField) {
  // Step 1: Bearer mode は external client 用に refresh token を body で返し、Cookie command や browser index hint を混ぜない。
  return [
    {
      scenarioId: '[API-CONTRACT-BE-S012]',
      schemaName,
      requiredFields: ['credentialMode', 'authContextId', 'sessionId', 'accessToken', 'expiresAt', refreshTokenField],
      forbiddenFields: ['clearCookieCommands', 'contextIndexUpdateHints', 'operatorRefreshToken', 'operatorAccessToken'],
      reason: 'bearer mode response must expose plaintext refresh token and exclude browser Cookie/index commands',
    },
  ];
}

/**
 * readOpenAPIServerHost は OpenAPI servers[0].url から hostname を取り出す。
 *
 * @param {string} artifactPath error message 用 artifact path。
 * @param {{servers?: Array<{url?: string}>}} contract OpenAPI contract object。
 * @returns {string} 小文字化した server hostname。
 */
function readOpenAPIServerHost(artifactPath, contract) {
  // Step 1: OpenAPI server URL は absolute URL を必須にし、origin 分離を検査できない artifact を拒否する。
  const serverURL = contract.servers?.[0]?.url;
  if (!serverURL) {
    throw new Error(`${relative(process.cwd(), artifactPath)} must declare servers[0].url`);
  }
  const parsedURL = new URL(serverURL);
  return parsedURL.hostname.toLowerCase();
}

/**
 * adminSurfaceTokens は Product artifact に混入してはならない Admin operation/tag 語彙を返す。
 *
 * @returns {{operationIds: Set<string>, tags: Set<string>}} Admin surface の禁止語彙。
 */
function adminSurfaceTokens() {
  // Step 1: Admin route namespace の operationId と tag だけを列挙し、Admin model 名の説明文を誤検知しない。
  return {
    operationIds: new Set([
      'createAdminAccount',
      'createAdminOperator',
      'deleteAdminOperatorPasskey',
      'finishAdminInitialSetup',
      'finishAdminOperatorSetup',
      'finishAdminPasskeyAuthentication',
      'getAdminAccount',
      'getCurrentAdminOperator',
      'listAdminAccounts',
      'listAdminOperatorPasskeys',
      'logoutAdminOperator',
      'refreshAdminOperatorSession',
      'startAdminInitialSetup',
      'startAdminOperatorSetup',
      'startAdminPasskeyAuthentication',
    ]),
    tags: new Set(['admin-accounts', 'admin-auth']),
  };
}

/**
 * productSurfaceTokens は Admin artifact に混入してはならない Product operation/tag 語彙を返す。
 *
 * @returns {{operationIds: Set<string>, tags: Set<string>}} Product surface の禁止語彙。
 */
function productSurfaceTokens() {
  // Step 1: Product route namespace の operationId と tag だけを列挙し、shared model の利用は許可する。
  return {
    operationIds: new Set([
      'consumeRecoveryToken',
      'deletePasskey',
      'finishPasskeyAddition',
      'finishPasskeyAuthentication',
      'finishReauthentication',
      'getAccountSettings',
      'getStatus',
      'listPasskeys',
      'listSessions',
      'logout',
      'refreshToken',
      'registerPasskey',
      'requestPasskeyRecovery',
      'revokeOtherSessions',
      'revokeSession',
      'sendDeviceLink',
      'startPasskeyAddition',
      'startPasskeyAuthentication',
      'startPasskeyRegistration',
      'startReauthentication',
      'updateAccountSettings',
    ]),
    tags: new Set(['account-settings', 'app-auth', 'auth', 'status']),
  };
}

/**
 * collectTypeSpecFiles は指定 directory 配下の `.tsp` source を再帰的に列挙する。
 *
 * @param {string} directory 検査対象の directory。Product route namespace の root を渡す。
 * @returns {string[]} 検査対象 TypeSpec file の絶対 path 一覧。directory の副作用はない。
 * @throws {Error} directory が読めない場合は Node.js の filesystem error をそのまま送出する。
 */
function collectTypeSpecFiles(directory) {
  // Step 1: directory entry を type 付きで読み、file と subdirectory を確実に分ける。
  const entries = readdirSync(directory, { withFileTypes: true });
  const files = [];

  // Step 2: subdirectory は再帰し、`.tsp` file だけを検査対象として返す。
  for (const entry of entries) {
    const entryPath = join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectTypeSpecFiles(entryPath));
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.tsp')) {
      files.push(entryPath);
    }
  }

  return files;
}

/**
 * detectProductSurfaceBoundaryViolations は Product TypeSpec source 内の Admin route namespace 混入を検出する。
 *
 * @param {string} filePath 検査する TypeSpec file。通常は Product route 配下の `.tsp` file または test fixture を渡す。
 * @returns {string[]} `path:line` 付きの違反 message。一覧が空なら Product/Admin route 境界は保たれている。
 * @throws {Error} file が読めない場合は Node.js の filesystem error をそのまま送出する。
 */
function detectProductSurfaceBoundaryViolations(filePath) {
  // Step 1: source を UTF-8 text として読み、検査対象 file 以外へ副作用を出さない。
  const source = readFileSync(filePath, 'utf8');
  const lines = source.split(/\r?\n/u);
  const displayPath = relative(process.cwd(), filePath);
  const violations = [];

  // Step 2: 行ごとに禁止 pattern を適用し、どの Admin route namespace 参照が混入したかを説明する。
  for (const [index, line] of lines.entries()) {
    const executableLine = stripLineComment(line);
    for (const { pattern, reason } of productSurfaceForbiddenPatterns) {
      if (pattern.test(executableLine)) {
        violations.push(`${displayPath}:${index + 1}: ${reason}`);
      }
    }
  }

  return violations;
}

/**
 * stripLineComment は TypeSpec の `//` comment を行末から除去する。
 *
 * @param {string} line TypeSpec source の 1 行。
 * @returns {string} comment を除いた検査用 text。文字列 literal 内の `//` は fixture で使わない前提の軽量 guardrail 用処理。
 */
function stripLineComment(line) {
  // Step 1: comment の説明文で Admin.ApiV1 と書いた場合に誤検知しないよう、実行部分だけを残す。
  const commentIndex = line.indexOf('//');
  if (commentIndex === -1) {
    return line;
  }

  return line.slice(0, commentIndex);
}
