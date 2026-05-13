import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { DeviceManager } from '@www-template/ui/components';
import type { DeviceSession } from '@www-template/ui/components/device-manager';

function createDevice(overrides: Partial<DeviceSession> = {}): DeviceSession {
  return {
    sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    deviceName: 'Chrome on macOS',
    loginAt: '2026-01-01T00:00:00.000Z',
    lastActiveAt: '2026-01-01T12:00:00.000Z',
    ipHash: 'abc123',
    isCurrentSession: false,
    ...overrides,
  };
}

describe('DeviceManager', () => {
  it('[AUTH-FE-S034] renders session list with device names and timestamps', () => {
    // Arrange: 2 つのデバイスセッションを props として渡す
    const devices = [
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', deviceName: 'Chrome on macOS' }),
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAW', deviceName: 'Safari on iOS' }),
    ];

    render(DeviceManager, {
      props: {
        devices,
        currentSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        loading: false,
        error: null,
        onRevoke: vi.fn(),
        onRevokeOthers: vi.fn(),
      },
    });

    // Assert: 各デバイス名が表示される
    expect(screen.getByText('Chrome on macOS')).toBeInTheDocument();
    expect(screen.getByText('Safari on iOS')).toBeInTheDocument();

    // Assert: ログイン時刻と最終アクティブ時刻が表示される（複数あるため getAllByText を使用）
    expect(screen.getAllByText(/ログイン:/)).toHaveLength(2);
    expect(screen.getAllByText(/最終アクティブ:/)).toHaveLength(2);

    // Assert: 現在のデバイスインジケーターが表示される
    expect(screen.getByLabelText('現在のデバイス')).toBeInTheDocument();
  });

  it('[AUTH-FE-S035] triggers onRevoke with sessionId when logout button is clicked', async () => {
    // Arrange: onRevoke スパイを準備
    const user = userEvent.setup();
    const onRevoke = vi.fn();
    const devices = [
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', deviceName: 'Chrome on macOS' }),
    ];

    render(DeviceManager, {
      props: {
        devices,
        currentSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        loading: false,
        error: null,
        onRevoke,
        onRevokeOthers: vi.fn(),
      },
    });

    // Act: ログアウトボタンをクリック
    const logoutButton = screen.getByRole('button', { name: 'Chrome on macOS をログアウト' });
    await user.click(logoutButton);

    // Assert: onRevoke が正しい sessionId で呼ばれる
    expect(onRevoke).toHaveBeenCalledTimes(1);
    expect(onRevoke).toHaveBeenCalledWith('01ARZ3NDEKTSV4RRFFQ69G5FAV');
  });

  it('[AUTH-FE-S036] triggers onRevokeOthers when revoke-others button is clicked', async () => {
    // Arrange: onRevokeOthers スパイを準備
    const user = userEvent.setup();
    const onRevokeOthers = vi.fn();
    const devices = [
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', deviceName: 'Chrome on macOS' }),
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAW', deviceName: 'Safari on iOS' }),
    ];

    render(DeviceManager, {
      props: {
        devices,
        currentSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        loading: false,
        error: null,
        onRevoke: vi.fn(),
        onRevokeOthers,
      },
    });

    // Act: 「他のすべてのデバイスをログアウト」ボタンをクリック
    const revokeOthersButton = screen.getByRole('button', {
      name: '他のすべてのデバイスをログアウト',
    });
    await user.click(revokeOthersButton);

    // Assert: onRevokeOthers が呼ばれる
    expect(onRevokeOthers).toHaveBeenCalledTimes(1);
  });

  it('shows generic error message when error prop is provided', () => {
    // Arrange: エラーメッセージを props で渡す
    render(DeviceManager, {
      props: {
        devices: [],
        currentSessionId: '',
        loading: false,
        error: 'デバイス一覧の取得に失敗しました。',
        onRevoke: vi.fn(),
        onRevokeOthers: vi.fn(),
      },
    });

    // Assert: Alert ロールでエラーメッセージが表示される
    const alert = screen.getByRole('alert');
    expect(alert).toBeInTheDocument();
    expect(alert.textContent).toContain('デバイス一覧の取得に失敗しました。');
  });

  it('disables revoke-others button when no other devices exist', () => {
    // Arrange: 現在のデバイスのみを props で渡す
    const devices = [
      createDevice({ sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', deviceName: 'Chrome on macOS' }),
    ];

    render(DeviceManager, {
      props: {
        devices,
        currentSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        loading: false,
        error: null,
        onRevoke: vi.fn(),
        onRevokeOthers: vi.fn(),
      },
    });

    // Assert: 「他のすべてのデバイスをログアウト」ボタンが無効化される
    const revokeOthersButton = screen.getByRole('button', {
      name: '他のすべてのデバイスをログアウト',
    });
    expect(revokeOthersButton).toBeDisabled();
  });

  it('shows loading spinner when loading is true', () => {
    // Arrange: loading=true でレンダリング
    render(DeviceManager, {
      props: {
        devices: [],
        currentSessionId: '',
        loading: true,
        error: null,
        onRevoke: vi.fn(),
        onRevokeOthers: vi.fn(),
      },
    });

    // Assert: ローディングテキストが表示される
    expect(screen.getByText('読み込み中…')).toBeInTheDocument();
  });
});
