import type { PasskeyItem, PasskeyManagementState } from '../../types';

/** passkey management state の初期値を作る。 */
function createPasskeyManagementInitialState(): PasskeyManagementState {
  return {
    passkeys: [],
    loading: false,
    error: null,
  };
}

/** 削除成功後に対象パスキーを state から除去する。 */
function applyPasskeyDeleted(state: PasskeyManagementState, id: string): void {
  state.passkeys = state.passkeys.filter((p: PasskeyItem) => p.id !== id);
}

/** 一覧取得成功後に state を更新する。 */
function applyPasskeyList(state: PasskeyManagementState, passkeys: PasskeyItem[]): void {
  state.passkeys = passkeys;
}

/** passkey operation エラーを state に反映する。 */
function applyPasskeyError(state: PasskeyManagementState, message: string): void {
  state.error = message;
}

/** passkey operation エラーを状態メッセージへ正規化する。 */
function toPasskeyManagementErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return 'パスキー操作に失敗しました。時間を置いて再度お試しください。';
}

export {
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  createPasskeyManagementInitialState,
  toPasskeyManagementErrorMessage,
};
