/**
 * 認証 surface で表示される error コードを、ユーザー向け安全な文言へ翻訳する。
 *
 * ドメイン層から渡される文字列は英数コード（例: `passkeyOperationCancelledOrTimedOut`）
 * であり、内部実装情報としてユーザーに直接表示してはならない。
 * allowlist にあるコードのみ app の catalog 上の文言へ写像し、
 * 未知のコード・例外メッセージは汎用 safe 文言へ fail-close する。
 */

import type { Locale } from '@www-template/i18n';

import type { useI18n } from './index';

type AppI18n = ReturnType<typeof useI18n>;

const KNOWN_CODES = new Set<string>([
  'passkeyOperationCancelledOrTimedOut',
  'passkeyOperationInvalidState',
  'passkeyOperationNotSupported',
  'passkeyOperationSecurityError',
  'passkeyOperationAborted',
  'passkeyOperationBrowserUnsupported',
  'passkeyOperationFailed',
  'passkeyAddFailed',
  'passkeyDeleteFailed',
  'passkeysListLoadFailed',
  'reauthRequired',
  'reauth_session_required',
  'deviceLinkSendFailed',
  'passkeyLastDeleteBlocked',
  'last_passkey_cannot_be_deleted',
  'account-suspended',
  'session-expired',
]);

/**
 * 認証 surface の error を表示用文言へ翻訳する。
 * 未知のコードは fail-close して汎用文言を返す。
 *
 * @param i18n - useI18n() の戻り値。null 安全。
 * @param code - ドメインから渡された error 文字列。
 * @param locale - フォールバック文言の選択基準。null の場合は ja 既定。
 * @returns ユーザー向けに整形されたエラー文言。
 */
export function formatAuthError(
  i18n: AppI18n | null,
  code: string | null | undefined,
  locale: Locale = 'ja'
): string {
  if (code === null || code === undefined || code === '') {
    return '';
  }

  if (KNOWN_CODES.has(code)) {
    switch (code) {
      case 'passkeyOperationCancelledOrTimedOut':
        return (
          i18n?.t('common.passkeyOperationCancelledOrTimedOut') ?? FALLBACK.ja.passkeyCancelled
        );
      case 'passkeyOperationInvalidState':
        return i18n?.t('common.passkeyOperationInvalidState') ?? FALLBACK.ja.passkeyInvalid;
      case 'passkeyOperationNotSupported':
        return i18n?.t('common.passkeyOperationNotSupported') ?? FALLBACK.ja.passkeyNotSupported;
      case 'passkeyOperationSecurityError':
        return i18n?.t('common.passkeyOperationSecurityError') ?? FALLBACK.ja.passkeySecurity;
      case 'passkeyOperationAborted':
        return i18n?.t('common.passkeyOperationAborted') ?? FALLBACK.ja.passkeyAborted;
      case 'passkeyOperationBrowserUnsupported':
        return (
          i18n?.t('common.passkeyOperationBrowserUnsupported') ??
          FALLBACK.ja.passkeyBrowserUnsupported
        );
      case 'passkeyOperationFailed':
        return i18n?.t('common.passkeyOperationFailed') ?? FALLBACK.ja.passkeyFailed;
      case 'passkeyAddFailed':
        return i18n?.t('common.passkeyAddFailed') ?? FALLBACK.ja.addFailed;
      case 'passkeyDeleteFailed':
        return i18n?.t('common.passkeyDeleteFailed') ?? FALLBACK.ja.deleteFailed;
      case 'passkeysListLoadFailed':
        return i18n?.t('common.passkeysListLoadFailed') ?? FALLBACK.ja.listLoadFailed;
      case 'reauthRequired':
      case 'reauth_session_required':
        return i18n?.t('common.reauthRequired') ?? FALLBACK.ja.reauthRequired;
      case 'deviceLinkSendFailed':
        return i18n?.t('common.deviceLinkSendFailed') ?? FALLBACK.ja.deviceLinkFailed;
      case 'passkeyLastDeleteBlocked':
      case 'last_passkey_cannot_be_deleted':
        return i18n?.t('common.passkeyLastDeleteBlocked') ?? FALLBACK.ja.lastBlocked;
      case 'account-suspended':
        return i18n?.t('common.accountSuspendedTitle') ?? FALLBACK.ja.accountSuspended;
      case 'session-expired':
        return i18n?.t('common.sessionExpiredTitle') ?? FALLBACK.ja.sessionExpired;
    }
  }

  // 未知のコードは絶対に出さず、汎用文言を返す（fail-close）
  return locale === 'en' ? FALLBACK.en.generic : FALLBACK.ja.generic;
}

const FALLBACK = {
  ja: {
    generic: 'パスキー認証を完了できませんでした。時間を置いて再度お試しください。',
    passkeyCancelled:
      'パスキー操作がキャンセルされたか、時間切れになりました。もう一度お試しください。',
    passkeyInvalid:
      'この端末には既にパスキーが登録されているか、利用できるパスキーが見つかりません。',
    passkeyNotSupported: 'このブラウザーまたは端末はパスキーに対応していません。',
    passkeySecurity:
      'セキュリティエラーが発生しました。ページを再読み込みしてもう一度お試しください。',
    passkeyAborted: 'パスキー操作が中断されました。もう一度お試しください。',
    passkeyBrowserUnsupported:
      'パスキー操作を完了できませんでした。ブラウザーがパスキーに対応しているか確認してください。',
    passkeyFailed: 'パスキー操作に失敗しました。時間を置いて再度お試しください。',
    addFailed: 'パスキーの登録に失敗しました。',
    deleteFailed: 'パスキーの削除に失敗しました。',
    listLoadFailed: 'パスキー一覧の取得に失敗しました。',
    reauthRequired: '再認証が必要です。',
    deviceLinkFailed: 'ログイン有効化リンクの送信に失敗しました。',
    lastBlocked: '最後のパスキーは削除できません。',
    accountSuspended: 'このアカウントは現在ご利用いただけません。',
    sessionExpired: 'セッションが切れました。',
  },
  en: {
    generic: 'Could not complete the passkey operation. Please try again later.',
    passkeyCancelled: 'The passkey operation was cancelled or timed out. Please try again.',
    passkeyInvalid: 'This device already has a passkey or no passkey is available.',
    passkeyNotSupported: 'This browser or device does not support passkeys.',
    passkeySecurity: 'A security error occurred. Please reload the page and try again.',
    passkeyAborted: 'The passkey operation was aborted. Please try again.',
    passkeyBrowserUnsupported: 'Your browser may not support passkeys.',
    passkeyFailed: 'The passkey operation failed. Please try again later.',
    addFailed: 'Failed to register the passkey.',
    deleteFailed: 'Failed to delete the passkey.',
    listLoadFailed: 'Failed to load the passkey list.',
    reauthRequired: 'Re-authentication is required.',
    deviceLinkFailed: 'Failed to send the login enablement link.',
    lastBlocked: 'The last passkey cannot be deleted.',
    accountSuspended: 'This account is currently unavailable.',
    sessionExpired: 'Your session has expired.',
  },
} as const;
