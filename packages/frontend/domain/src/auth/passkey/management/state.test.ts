import { describe, expect, it } from 'vitest';

import {
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  createPasskeyManagementInitialState,
  toPasskeyManagementErrorMessage,
} from './state';

import type { PasskeyItem } from '../../types';

describe('passkeyManagementState', () => {
  it('[AUTH-FE-S012] deletePasskey 成功時に data.passkeys から対象が除去される', () => {
    // Arrange: 2 件のパスキーが存在する状態
    const state = createPasskeyManagementInitialState();
    applyPasskeyList(state, [
      {
        id: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
        identifier: 'MacBook Pro',
        createdAt: '2026-01-01T00:00:00.000Z',
      },
      {
        id: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
        identifier: 'iPhone 15',
        createdAt: '2026-02-01T00:00:00.000Z',
      },
    ]);

    expect(state.passkeys).toHaveLength(2);

    // Act: deletePasskey 成功後の state 更新（applyPasskeyDeleted）
    applyPasskeyDeleted(state, '01ARZ3NDEKTSV4RRFFQ69G5FAX');

    // Assert: 対象が除去され、残りは変化しない
    expect(state.passkeys).toHaveLength(1);
    expect(state.passkeys.map((p: PasskeyItem) => p.id)).not.toContain(
      '01ARZ3NDEKTSV4RRFFQ69G5FAX'
    );
    expect(state.passkeys.map((p: PasskeyItem) => p.id)).toContain('01ARZ3NDEKTSV4RRFFQ69G5FB1');
    expect(state.error).toBeNull();
  });

  it('[AUTH-FE-S013] 初期 state の deviceLinkSent は false', () => {
    const state = createPasskeyManagementInitialState();
    expect(state.deviceLinkSent).toBe(false);
  });

  it('[AUTH-FE-S015] deletePasskey で API エラー時に data.passkeys が変化しない', () => {
    // Arrange: 1 件のパスキーが存在する状態
    const state = createPasskeyManagementInitialState();
    applyPasskeyList(state, [
      {
        id: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
        identifier: 'MacBook Pro',
        createdAt: '2026-01-01T00:00:00.000Z',
      },
    ]);

    expect(state.passkeys).toHaveLength(1);

    // Act: API エラー発生時（applyPasskeyDeleted は呼ばれず、applyPasskeyError のみ呼ばれる）
    const apiError = new Error('last_passkey_cannot_be_deleted');
    applyPasskeyError(state, toPasskeyManagementErrorMessage(apiError));

    // Assert: passkeys は変化せず、error が設定される
    expect(state.passkeys).toHaveLength(1);
    expect(state.passkeys.map((p: PasskeyItem) => p.id)).toContain('01ARZ3NDEKTSV4RRFFQ69G5FAX');
    expect(state.error).toBe('last_passkey_cannot_be_deleted');
  });
});
