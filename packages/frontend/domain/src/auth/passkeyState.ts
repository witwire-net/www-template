import type { PasskeyLoginState } from 'types';

/** passkey login state の初期値を作る。 */
function createPasskeyLoginInitialState(): PasskeyLoginState {
  return {
    identifier: '',
    isSubmitting: false,
    lastChallengeRequestId: null,
    lastSession: null,
    lastCacheControl: null,
    error: null,
  };
}

/** auth operation error を login 向け文言へ正規化する。 */
function toPasskeyErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return 'パスキー認証を完了できませんでした。時間を置いて再度お試しください。';
}

/** recovery consume/register の失敗を retry guidance 文言へ正規化する。 */
function toRecoveryErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return '復旧リンクを確認できませんでした。もう一度やり直してください。';
}

export { createPasskeyLoginInitialState, toPasskeyErrorMessage, toRecoveryErrorMessage };
