import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

const packageRoot = new URL('../../', import.meta.url);
const adminRoot = new URL('../', packageRoot);
const repoRoot = new URL('../../../../../', import.meta.url);

function readPackageFile(path: string): string {
  // 実ファイルを読むことで、SvelteKit runtime を起動せずに package cleanup の成果だけを検証する。
  return readFileSync(new URL(path, packageRoot), 'utf8');
}

function readRepoFile(path: string): string {
  // root config は Admin layer 境界を lint で守る入口なので、文字列 contract として確認する。
  return readFileSync(new URL(path, repoRoot), 'utf8');
}

function readAdminFile(path: string): string {
  // split 後の app/domain/api sibling package を同じ Admin surface として読み、物理境界の contract を確認する。
  return readFileSync(new URL(path, adminRoot), 'utf8');
}

function listFiles(path: URL): string[] {
  // 空ディレクトリの有無ではなく、runtime source が残っているかを再帰的に確認する。
  if (!existsSync(path)) return [];

  return readdirSync(path).flatMap((entry) => {
    const child = new URL(entry, path);
    const childStat = statSync(child);
    const childPath = new URL(`${entry}${childStat.isDirectory() ? '/' : ''}`, path);
    if (childStat.isDirectory()) return listFiles(childPath);
    return [childPath.pathname];
  });
}

describe('Admin static client cleanup', () => {
  it('[ADMIN-CONSOLE-FE-S038] app layer cannot import Admin API client directly', () => {
    // eslint config と route source の両面から、app -> domain -> api の依存方向を固定する。
    const eslintConfig = readRepoFile('eslint.config.js');
    const accountsPage = readPackageFile('src/routes/accounts/+page.svelte');
    const loginPage = readPackageFile('src/routes/login/+page.svelte');

    expect(eslintConfig).toContain('Admin Console app 層では Admin API layer を直接 import');
    expect(eslintConfig).toContain('@www-template/admin-api');
    expect(accountsPage).toContain("from '@www-template/admin-domain'");
    expect(loginPage).toContain("from '@www-template/admin-domain'");
    expect(accountsPage).not.toContain("from '@www-template/admin-api'");
    expect(loginPage).not.toContain("from '@www-template/admin-api'");
  });

  it('server runtime directories and route modules are removed', () => {
    // SvelteKit server runtime / Prisma schema の再追加を早期に検出する。
    const forbiddenPaths = [
      'src/hooks.server.ts',
      'src/routes/+layout.server.ts',
      'src/routes/+page.server.ts',
    ];

    for (const path of forbiddenPaths) {
      expect(existsSync(new URL(path, packageRoot)), path).toBe(false);
    }

    expect(listFiles(new URL('src/lib/server/', packageRoot))).toEqual([]);
    expect(
      listFiles(new URL(['src', 'routes', 'api', 'admin', ''].join('/'), packageRoot))
    ).toEqual([]);
    expect(listFiles(new URL('prisma/', packageRoot))).toEqual([]);
  });

  it('package scripts and dependencies exclude server-only runtime ownership', () => {
    // package manifest から Prisma / Valkey / Node adapter / WebAuthn server の runtime 所有を排除する。
    const manifest = JSON.parse(readPackageFile('package.json')) as {
      scripts: Record<string, string>;
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    const allDependencyNames = [
      ...Object.keys(manifest.dependencies ?? {}),
      ...Object.keys(manifest.devDependencies ?? {}),
    ];

    expect(
      Object.keys(manifest.scripts).some((scriptName) => scriptName.startsWith('prisma:'))
    ).toBe(false);
    expect(allDependencyNames).not.toContain('@prisma/client');
    expect(allDependencyNames).not.toContain('prisma');
    expect(allDependencyNames).not.toContain('ioredis');
    expect(allDependencyNames).not.toContain('@simplewebauthn/server');
    expect(allDependencyNames).not.toContain('@sveltejs/adapter-node');
    expect(allDependencyNames).toContain('@sveltejs/adapter-static');
  });

  it('SvelteKit config is static and lint config rejects server runtime reintroduction', () => {
    // static adapter と lint guard の両方があることで、build と future change の入口を同時に守る。
    const svelteConfig = readPackageFile('svelte.config.js');
    const eslintConfig = readRepoFile('eslint.config.js');

    expect(svelteConfig).toContain("import adapter from '@sveltejs/adapter-static'");
    expect(svelteConfig).toContain("fallback: 'index.html'");
    expect(eslintConfig).toContain('Admin Console は静的 client です');
    expect(eslintConfig).toContain('packages/admin/app/src/routes/**/+server.ts');
    expect(eslintConfig).toContain('packages/admin/app/src/lib/server/**/*.{ts,js,svelte}');
  });

  it('[ADMIN-CONSOLE-FE-S043/S044/S045] account create UI delegates to domain flow', () => {
    // Account 作成 UI は component と domain action に分離し、成功遷移・validation・duplicate 表示を page state で扱う。
    const accountsPage = readPackageFile('src/routes/accounts/+page.svelte');
    const accountForm = readPackageFile('src/lib/components/accounts/AccountCreateForm.svelte');

    expect(accountsPage).toContain('useAdminAccounts');
    expect(accountsPage).toContain('await adminAccounts.actions.submitCreateAccount()');
    expect(accountsPage).toContain('createMessage = $derived');
    expect(accountsPage).toContain("if (error === 'duplicate-email')");
    expect(accountForm).toContain('bind:value={email}');
    expect(accountForm).toContain("disabled={isSubmitting || email.trim() === ''}");
  });

  it('[ADMIN-CONSOLE-FE-S046] Admin surface remains distinct from Product surfaces', () => {
    // deployment docs は後続 task 対象のため、ここでは Admin の dev origin と package surface 分離を固定する。
    const adminViteConfig = readPackageFile('vite.config.ts');
    const rootManifest = readRepoFile('package.json');

    expect(adminViteConfig).toContain('port: 5176');
    expect(rootManifest).toContain('dev:web');
    expect(rootManifest).toContain('dev:app');
    expect(rootManifest).toContain('dev:admin');
  });

  it('[ADMIN-AUTH-FE-S027/S028] login UI uses Admin auth domain and not Product SDK', () => {
    // login route は browser WebAuthn と Admin domain だけを使い、Product auth SDK や package-local BFF を経由しない。
    const loginPage = readPackageFile('src/routes/login/+page.svelte');
    const authDomain = readAdminFile('domain/src/auth.ts');
    const eslintConfig = readRepoFile('eslint.config.js');

    expect(loginPage).toContain("from '@simplewebauthn/browser'");
    expect(loginPage).toContain('useAdminLogin');
    expect(loginPage).toContain('await login.actions.submit(');
    expect(authDomain).toContain('requestStartAdminLogin({ identifier: normalizedIdentifier })');
    expect(authDomain).not.toContain("from '@www-template/api'");
    expect(eslintConfig).toContain('Admin domain 層から Product SDK を import しないでください');
  });

  it('[ADMIN-AUTH-FE-S030/S031] protected routes hide content until backend verification', () => {
    // layout は current operator 検証前に child content を描画せず、role は表示制御だけに限定する。
    const layout = readPackageFile('src/routes/+layout.svelte');
    const sessionHook = readAdminFile('domain/src/hooks/useAdminSession.svelte.ts');
    const operatorsDomain = readAdminFile('domain/src/operators.ts');
    const accountsDomain = readAdminFile('domain/src/accounts.ts');

    expect(layout).toContain('useAdminSession');
    expect(layout).toContain('const routeState = $derived(session.data.state.routeState)');
    expect(layout).toContain("void goto('/login')");
    expect(layout).toContain('{:else if operator !== null}');
    expect(operatorsDomain).toContain('canCreateAccounts');
    expect(operatorsDomain).toContain('backend authorization の代替にはしません');
    expect(accountsDomain).toContain("if (status === 403) return 'forbidden'");
    expect(sessionHook).toContain('subscribeAdminContextIndexChanges');
    expect(sessionHook).toContain('void actions.verifyCurrentRoute(currentPath)');
  });

  it('[ADMIN-AUTH-FE-S032] Admin HTML is served with no-store semantics', () => {
    // Admin route shell は no-store、fingerprinted immutable assets だけ長期 cache 可能にする。
    const headersConfig = readPackageFile('static/_headers');
    const nginxConfig = readPackageFile('nginx.conf');

    expect(headersConfig).toContain('/*');
    expect(headersConfig).toContain('Cache-Control: no-store');
    expect(headersConfig).toContain('/_app/immutable/*');
    expect(headersConfig).toContain('Cache-Control: public, max-age=31536000, immutable');
    expect(nginxConfig).toContain('try_files $uri $uri/ /index.html');
    expect(nginxConfig).toContain('location ^~ /api/v1/');
    expect(nginxConfig).toContain('return 404');
  });

  it('[ADMIN-AUTH-FE-S033/S034/S035/S036/S037] auth state stays memory-only and generic', () => {
    // auth domain は accessToken の memory state と Cookie refresh orchestration だけを持ち、expiry reason を UI へ露出しない。
    const authDomain = readAdminFile('domain/src/auth.ts');

    expect(authDomain).toContain('let currentSession: AdminSessionState | null = null');
    expect(authDomain).toContain('requestCurrentAdminOperator(session)');
    expect(authDomain).toContain('requestRefreshAdminSession(authContextId)');
    expect(authDomain).toContain('clearAdminSession()');
    expect(authDomain).not.toContain('localStorage');
    expect(authDomain).not.toContain('sessionStorage');
    expect(authDomain).not.toContain('indexedDB');
    expect(authDomain).not.toContain('refreshToken:');
  });

  it('[ADMIN-AUTH-FE-S038/S039/S040] setup UI uses Admin setup API and hides unavailable forms', () => {
    // 初回 setup UI は `/auth/setup/*` domain action を使い、operator 既存・bootstrap 無効時に secret form を出さない。
    const setupPage = readPackageFile('src/routes/setup/+page.svelte');
    const authDomain = readAdminFile('domain/src/auth.ts');
    const apiClient = readAdminFile('api/src/client.ts');

    expect(setupPage).toContain('useAdminInitialSetup');
    expect(setupPage).toContain('await initialSetup.actions.submit(');
    expect(setupPage).toContain("type SetupAvailability = 'available' | 'operator-exists'");
    expect(setupPage).toContain('initialSetup.data.state.setupAvailability');
    expect(setupPage).toContain('{#if showSetupForm}');
    expect(setupPage).toContain("initialSetup.data.state.setupAvailability === 'operator-exists'");
    expect(apiClient).toContain('getStartAdminInitialSetupUrl()');
    expect(apiClient).toContain('getFinishAdminInitialSetupUrl()');
    expect(authDomain).toContain(
      'requestStartInitialAdminSetup({ email, displayName, bootstrapSecret })'
    );
    expect(authDomain).not.toContain('bootstrapSecret: currentSession');
  });

  it('[ADMIN-I18N-FE-S001] Admin routes use shared client locale state', () => {
    // server load がない静的 SPA では data.locale に依存せず、client locale state を全 route の表示 source にする。
    const routeFiles = [
      'src/routes/+layout.svelte',
      'src/routes/+page.svelte',
      'src/routes/accounts/+page.svelte',
      'src/routes/accounts/[id]/+page.svelte',
      'src/routes/audit/+page.svelte',
      'src/routes/login/+page.svelte',
      'src/routes/operator-setup/+page.svelte',
      'src/routes/passkeys/+page.svelte',
      'src/routes/settings/+page.svelte',
      'src/routes/settings/operators/+page.svelte',
      'src/routes/setup/+page.svelte',
    ];

    for (const routeFile of routeFiles) {
      const source = readPackageFile(routeFile);
      expect(source, routeFile).toContain('createCurrentAdminI18n');
      expect(source, routeFile).not.toContain('data?.locale');
      expect(source, routeFile).not.toContain('pageData.locale');
    }
  });

  it('[ADMIN-I18N-FE-S002] settings page has no English fallback copy', () => {
    // settings は最初に壊れていた画面なので、英語 fallback object へ戻らないことを source contract として固定する。
    const settingsPage = readPackageFile('src/routes/settings/+page.svelte');

    expect(settingsPage).toContain("i18n.t('settings.title')");
    expect(settingsPage).toContain('useAdminSettings');
    expect(settingsPage).toContain('settings.actions.saveLocale()');
    expect(settingsPage).not.toContain("?? 'Settings'");
    expect(settingsPage).not.toContain("?? 'Language'");
    expect(settingsPage).not.toContain("?? 'Save'");
    expect(settingsPage).not.toContain('preventDefault()');
  });

  it('[ADMIN-CONSOLE-FE-S047] operator create UI delegates to Admin domain flow', () => {
    // operator 作成 UI は no-op form ではなく domain action へ委譲し、setup token 平文を画面に出さない。
    const operatorsPage = readPackageFile('src/routes/settings/operators/+page.svelte');
    const operatorsDomain = readAdminFile('domain/src/operators.ts');
    const apiClient = readAdminFile('api/src/client.ts');

    expect(operatorsPage).toContain('useAdminOperators');
    expect(operatorsPage).toContain('await operators.actions.submitCreateOperator()');
    expect(operatorsPage).not.toContain('preventDefault()');
    expect(operatorsPage).not.toContain('form?.setupToken');
    expect(operatorsDomain).toContain(
      'requestCreateAdminOperator({ email, role: input.role }, session)'
    );
    expect(apiClient).toContain('getCreateAdminOperatorUrl()');
    expect(apiClient).toContain('createAdminRequestInit(session)');
  });
});
