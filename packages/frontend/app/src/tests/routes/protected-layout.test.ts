import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

// Protected layout のソースコードパス
const layoutPath = resolve(__dirname, '../../routes/(protected)/+layout.svelte');

describe('[APP-IA-001] Protected layout の PageHeader 廃止', () => {
  it('ProtectedLayout が PageHeader をインポートしていない', () => {
    // ソースコードを直接読み取り、import 文を検証する。
    // toString() はコンパイル済みコードを返すため、tree-shaking されない内部参照が
    // 残る可能性がある。ソースレベルで import がないことを確認する方が確実。
    const source = readFileSync(layoutPath, 'utf-8');

    // PageHeader の import がないことを確認
    expect(source).not.toMatch(/import.*PageHeader/);
    expect(source).not.toMatch(/PageHeader.*from '@www-template\/ui'/);
  });

  it('ProtectedLayout が AppShell を使用している', () => {
    const source = readFileSync(layoutPath, 'utf-8');

    // AppShell がインポートされていることを確認
    expect(source).toMatch(/import AppShell from '\$lib\/layouts\/AppShell\.svelte'/);
  });

  it('ProtectedLayout が AppSidebar を使用している', () => {
    const source = readFileSync(layoutPath, 'utf-8');

    // AppSidebar がインポートされていることを確認
    expect(source).toMatch(/import AppSidebar from '\$lib\/layouts\/AppSidebar\.svelte'/);
  });

  it('ProtectedLayout が AppUserMenu を使用している', () => {
    const source = readFileSync(layoutPath, 'utf-8');

    // AppUserMenu がインポートされていることを確認
    expect(source).toMatch(/import AppUserMenu from '\$lib\/components\/AppUserMenu\.svelte'/);
  });
});
