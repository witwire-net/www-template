import { describe, it, expect } from 'vitest';

import { hasPermission, requirePermission } from './guard.js';
import { PERMISSIONS } from './permissions.js';

import type { Permission } from './permissions.js';

describe('hasPermission', () => {
  it('admin は全権限を持つ', () => {
    for (const perm of PERMISSIONS as readonly Permission[]) {
      expect(hasPermission('admin', perm)).toBe(true);
    }
  });

  it('viewer は accounts:suspend を持たない', () => {
    expect(hasPermission('viewer', 'accounts:suspend')).toBe(false);
  });

  it('存在しないロールは常に false を返す', () => {
    expect(hasPermission('unknown_role', 'accounts:read')).toBe(false);
  });

  it('未定義権限は false を返す', () => {
    expect(hasPermission('admin', 'unknown:permission' as Permission)).toBe(false);
  });
});

describe('requirePermission', () => {
  it('権限がある場合はエラーを throw しない', () => {
    const operator = {
      id: '1',
      email: 'a@example.com',
      role: 'admin',
      sessionId: 's1',
      jti: 'j1',
    };
    expect(() => {
      requirePermission(operator, 'accounts:read');
    }).not.toThrow();
  });

  it('権限がない場合は 403 Insufficient permissions を throw する', () => {
    const operator = {
      id: '1',
      email: 'a@example.com',
      role: 'viewer',
      sessionId: 's1',
      jti: 'j1',
    };
    let caught: unknown;
    try {
      requirePermission(operator, 'accounts:suspend');
    } catch (error) {
      caught = error;
    }
    expect(caught).toBeDefined();
    expect(caught).toMatchObject({
      status: 403,
      body: { message: 'Insufficient permissions' },
    });
  });

  it('operator が null の場合は 403 Insufficient permissions を throw する', () => {
    let caught: unknown;
    try {
      requirePermission(null, 'accounts:read');
    } catch (error) {
      caught = error;
    }
    expect(caught).toBeDefined();
    expect(caught).toMatchObject({
      status: 403,
      body: { message: 'Insufficient permissions' },
    });
  });
});
