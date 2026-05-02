import { render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import PasskeyList from '../../lib/profiles/PasskeyList.svelte';

/**
 * PasskeyList コンポーネントのレンダリングと表示検証テスト。
 *
 * テスト戦略:
 * - PasskeyList は純粋な presentational component（$props / $derived のみ）なので、
 *   @testing-library/svelte + jsdom で直接レンダリングできる。
 * - domain hook の $state rune を使う親 page とは分離して、props 経由の振る舞いを検証する。
 */
describe('PasskeyList', () => {
  it('[AUTH-FE-S016] otpIssued=true の場合、メール送信済み guidance が表示される', () => {
    // Arrange: OTP 発行済み状態でレンダリング
    render(PasskeyList, {
      props: {
        passkeys: [],
        loading: false,
        error: null,
        otpIssued: true,
        onAddPasskey: vi.fn(),
        onDeletePasskey: vi.fn(),
        onIssueOtp: vi.fn(),
      },
    });

    // Assert: 平文 OTP は表示されず、案内メッセージが表示される
    expect(screen.getByText(/ログイン有効化コードを送信しました/)).toBeInTheDocument();
    expect(
      screen.getByText(/登録済みのメールアドレス宛にコードを送信しました/)
    ).toBeInTheDocument();
    expect(screen.getByText(/有効期限: 5分/)).toBeInTheDocument();

    // OTP 値自体は画面に表示されない
    expect(screen.queryByText(/123456/)).not.toBeInTheDocument();
  });

  it('[AUTH-FE-S022] 再認証エラー時に Alert が表示される', () => {
    // Arrange: 再認証が必要なエラー状態でレンダリング
    render(PasskeyList, {
      props: {
        passkeys: [
          {
            id: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
            identifier: 'MacBook Pro',
            createdAt: '2026-01-01T00:00:00.000Z',
          },
        ],
        loading: false,
        error: '再認証が必要です。',
        otpIssued: false,
        onAddPasskey: vi.fn(),
        onDeletePasskey: vi.fn(),
        onIssueOtp: vi.fn(),
      },
    });

    // Assert: エラーアラートが表示される
    const alert = screen.getByRole('alert');
    expect(alert).toBeInTheDocument();
    expect(alert.textContent).toContain('再認証が必要です。');

    // 削除ボタンは表示されるが、実際の削除は親 page の reauth ガードでブロックされる
    const deleteButton = screen.getByLabelText('MacBook Pro を削除');
    expect(deleteButton).toBeInTheDocument();
  });
});
