import { describe, it, expect } from 'vitest';

import { PERMISSIONS, ROLE_PERMISSIONS } from './permissions.js';

import type { Permission } from './permissions.js';

describe('PERMISSIONS', () => {
  it('OpenSpec で定義された 8 つの権限を正しい順序で含む', () => {
    expect(PERMISSIONS).toEqual([
      'accounts:read',
      'accounts:suspend',
      'accounts:restore',
      'audit:read',
      'operators:read',
      'operators:write',
      'operators:setup_token',
      'operators:deactivate',
    ]);
  });
});

describe('ROLE_PERMISSIONS', () => {
  const allPerms = PERMISSIONS as readonly Permission[];

  it('admin は全 8 権限を持つ', () => {
    const adminPerms = ROLE_PERMISSIONS.get('admin');
    expect(adminPerms).toBeDefined();
    for (const perm of allPerms) {
      expect(adminPerms?.get(perm)).toBe(true);
    }
  });

  it('operator は accounts:read/suspend/restore と audit:read のみ持つ', () => {
    const operatorPerms = ROLE_PERMISSIONS.get('operator');
    expect(operatorPerms).toBeDefined();
    expect(operatorPerms?.get('accounts:read')).toBe(true);
    expect(operatorPerms?.get('accounts:suspend')).toBe(true);
    expect(operatorPerms?.get('accounts:restore')).toBe(true);
    expect(operatorPerms?.get('audit:read')).toBe(true);
    expect(operatorPerms?.get('operators:read')).toBe(false);
    expect(operatorPerms?.get('operators:write')).toBe(false);
    expect(operatorPerms?.get('operators:setup_token')).toBe(false);
    expect(operatorPerms?.get('operators:deactivate')).toBe(false);
  });

  it('viewer は accounts:read と audit:read のみ持つ', () => {
    const viewerPerms = ROLE_PERMISSIONS.get('viewer');
    expect(viewerPerms).toBeDefined();
    expect(viewerPerms?.get('accounts:read')).toBe(true);
    expect(viewerPerms?.get('audit:read')).toBe(true);
    expect(viewerPerms?.get('accounts:suspend')).toBe(false);
    expect(viewerPerms?.get('accounts:restore')).toBe(false);
    expect(viewerPerms?.get('operators:read')).toBe(false);
    expect(viewerPerms?.get('operators:write')).toBe(false);
    expect(viewerPerms?.get('operators:setup_token')).toBe(false);
    expect(viewerPerms?.get('operators:deactivate')).toBe(false);
  });
});
