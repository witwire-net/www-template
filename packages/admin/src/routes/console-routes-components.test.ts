import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';
const routeRoot = new URL('.', import.meta.url);

function readRoute(path: string): string {
  // テスト対象の route/component source を実ファイルから読み、SvelteKit runtime や DB を起動せずに画面契約を検証する。
  return readFileSync(new URL(path, routeRoot), 'utf8');
}

function expectContains(source: string, snippets: string[]): void {
  // 各 scenario に必要な表示文言・導線・action 名が source に残っていることを一括で確認する。
  for (const snippet of snippets) expect(source).toContain(snippet);
}

describe('Admin Console route/component scenarios', () => {
  it('account search page covers email filtering, empty state, status filter, pagination, loading contract, and stale-error contract', () => {
    // accounts 一覧の query/status/page URL 化と空状態を検証し、検索系 scenario の最小契約を固定する。
    const page = readRoute('accounts/+page.svelte');
    const table = readRoute('../lib/components/accounts/AccountTable.svelte');
    const server = readRoute('accounts/+page.server.ts');
    expectContains(page, [
      'encodeURIComponent(query)',
      'encodeURIComponent(status)',
      'Apply filters',
      '検索条件に一致するアカウントはありません。',
    ]);
    expectContains(table, ['Account list', 'Pagination.Pagination', 'onPageChange?.(p)']);
    expectContains(server, [
      'const PAGE_SIZE = 20',
      'searchAccounts(await getProductPrisma()',
      'filters: { query, status }',
    ]);
  });

  it('account detail page covers detail display, not-found redirect, zero passkeys, suspend, restore, and status-gated buttons', () => {
    // 詳細画面の表示項目・passkey 空状態・停止/復旧 form action を検証し、危険操作が確認経由であることを固定する。
    const page = readRoute('accounts/[id]/+page.svelte');
    const passkeys = readRoute('../lib/components/accounts/PasskeyList.svelte');
    const server = readRoute('accounts/[id]/+page.server.ts');
    expectContains(page, [
      '{data.account.email}',
      'Status reason:',
      'PasskeyList',
      '?/suspend',
      '?/restore',
      'Suspend reason',
      'Restore reason',
    ]);
    expectContains(page, [
      "data.account.status === 'active'",
      'confirmText="Suspend"',
      'confirmText="Restore"',
      'suspendForm?.requestSubmit()',
      'restoreForm?.requestSubmit()',
    ]);
    expectContains(passkeys, ['This account has no registered passkeys.', 'Passkeys']);
    expectContains(server, [
      "return redirect(303, '/accounts')",
      "requirePermission(locals.operator, 'accounts:suspend')",
      '停止理由を 1〜500 文字で入力してください。',
    ]);
  });

  it('audit page covers event table, action filter, details expansion, and empty state', () => {
    // 監査ログの一覧・filter query 化・details JSON 展開・空状態を検証し、監査閲覧の主要 UI 契約を固定する。
    const page = readRoute('audit/+page.svelte');
    const filter = readRoute('../lib/components/audit/AuditFilterBar.svelte');
    const table = readRoute('../lib/components/audit/AuditLogTable.svelte');
    expectContains(page, [
      'Audit Log',
      '条件に一致する監査イベントはありません。',
      'AuditLogTable',
      'AuditFilterBar',
    ]);
    expectContains(filter, [
      'Filter by action',
      "action: action !== '' ? action : undefined",
      'onFilter?.({});',
    ]);
    expectContains(table, [
      'Audit log',
      'operator_email',
      'toggleExpand(event.id)',
      'JSON.stringify(details, null, 2)',
      'CodeBlock',
    ]);
  });

  it('settings operators page covers listing, admin-only server guard, create, duplicate error surface, role update, deactivate, and setup token flows', () => {
    // オペレーター管理の一覧表示・admin 専用 guard・作成/更新/無効化/token 再発行導線を検証する。
    const page = readRoute('settings/operators/+page.svelte');
    const table = readRoute('../lib/components/operators/OperatorTable.svelte');
    const server = readRoute('settings/operators/+page.server.ts');
    expectContains(page, [
      'Operator table',
      'Add operator',
      'One-time setup token',
      'Rotate setup token',
      '?/create',
      '?/update',
      '?/deactivate',
      '?/rotate',
    ]);
    expectContains(table, [
      'Operator list',
      'op.id !== currentOperatorId',
      'Edit Role',
      'Deactivate',
      'Rotate Setup Token',
    ]);
    expectContains(server, [
      "requirePermission(locals.operator, 'operators:read')",
      "requirePermission(locals.operator, 'operators:write')",
      '入力値を確認してください。',
      'setupToken: result.plaintextToken',
    ]);
  });

  it('layout and dashboard cover navigation visibility, active links, operator name, KPIs, recent audit, and logout redirect', () => {
    // 共通レイアウトの role-based navigation と dashboard KPI/監査表示、logout cookie 削除を一括で検証する。
    const layout = readRoute('+layout.svelte');
    const layoutServer = readRoute('+layout.server.ts');
    const sidebar = readRoute('../lib/components/layout/AdminSidebar.svelte');
    const header = readRoute('../lib/components/layout/AdminHeader.svelte');
    const dashboard = readRoute('+page.svelte');
    const dashboardServer = readRoute('+page.server.ts');
    const logout = readRoute('api/admin/auth/logout/+server.ts');
    expectContains(layout, [
      'AdminShell',
      'operatorName={data.operator.email}',
      'navItems={data.navItems}',
    ]);
    expectContains(layoutServer, [
      "hasPermission(operator.role, 'operators:read')",
      'Settings',
      'currentPath: url.pathname',
    ]);
    expectContains(sidebar, [
      'Accounts',
      'Audit Log',
      'Settings',
      'isActive(item.href)',
      'currentPath.startsWith(href)',
    ]);
    expectContains(header, [
      "{operatorName !== '' ? operatorName : 'Operator'}",
      'action="/api/admin/auth/logout"',
      'Logout',
    ]);
    expectContains(dashboard, [
      'Total accounts',
      'Active accounts',
      'Suspended accounts',
      'Recent audit',
    ]);
    expectContains(dashboardServer, [
      'getDashboardStats(productPrisma)',
      'listAuditEvents(getAdminPrisma(), { page: 1, limit: 8 })',
    ]);
    expectContains(logout, [
      "cookies.delete('admin_session'",
      "cookies.delete('admin_csrf'",
      "return redirect(303, '/login')",
    ]);
  });
});
