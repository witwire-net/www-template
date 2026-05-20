<script lang="ts">
  import type { PasskeyItem } from '@www-template/domain/auth';
  import { useAccount } from '@www-template/domain';
  import { Alert, Button, Item, Separator, Spinner } from '@www-template/ui/components';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

/**
 * パスキー一覧と管理アクションを表示するコンポーネント。
 * 登録済みパスキーの表示、追加、削除、およびデバイスリンク送信の UI を提供する。
 */
  interface PasskeyListProps {
    /** List of registered passkeys. */
    passkeys: PasskeyItem[];
    /** Whether any async operation is in progress. */
    loading: boolean;
    /** Error message to display, or null. */
    error: string | null;
    /** Whether a device-link has been sent for new-device login enablement. */
    deviceLinkSent: boolean;
    /** Called when the user clicks "この端末でログインを有効にする" button. */
    onAddPasskey: () => void;
    /** Called when the user clicks "削除" on a passkey row. */
    onDeletePasskey: (id: string) => void;
    /** Called when the user clicks "新しい端末でログインを有効にする" to send a device-link. */
    onSendDeviceLink: () => void;
  }

  let {
    passkeys,
    loading,
    error,
    deviceLinkSent,
    onAddPasskey,
    onDeletePasskey,
    onSendDeviceLink,
  }: PasskeyListProps = $props();

  let isLastPasskey = $derived(passkeys.length === 1);

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  function formatDate(iso: string): string {
    return i18n.formatters.date(new Date(iso), {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  function formatPasskeyError(errorCode: string): string {
    // domain 層から渡された機械可読コードを app 層の catalog key に対応付ける。
    switch (errorCode) {
      case 'passkeysListLoadFailed':
        return i18n.t('common.passkeysListLoadFailed');
      case 'passkeyAddFailed':
        return i18n.t('common.passkeyAddFailed');
      case 'passkeyDeleteFailed':
        return i18n.t('common.passkeyDeleteFailed');
      case 'reauthRequired':
      case 'reauth_session_required':
        return i18n.t('common.reauthRequired');
      case 'deviceLinkSendFailed':
        return i18n.t('common.deviceLinkSendFailed');
      case 'last_passkey_cannot_be_deleted':
        return i18n.t('common.passkeyLastDeleteBlocked');
      case 'passkeyOperationCancelledOrTimedOut':
        return i18n.t('common.passkeyOperationCancelledOrTimedOut');
      case 'passkeyOperationInvalidState':
        return i18n.t('common.passkeyOperationInvalidState');
      case 'passkeyOperationNotSupported':
        return i18n.t('common.passkeyOperationNotSupported');
      case 'passkeyOperationSecurityError':
        return i18n.t('common.passkeyOperationSecurityError');
      case 'passkeyOperationAborted':
        return i18n.t('common.passkeyOperationAborted');
      case 'passkeyOperationBrowserUnsupported':
        return i18n.t('common.passkeyOperationBrowserUnsupported');
      default:
        // 未知の API コードや例外文をそのまま表示せず、汎用文言へ fail-close する。
        return i18n.t('common.passkeyOperationFailed');
    }
  }
</script>

<section aria-label={i18n.t('common.passkeysAriaLabel')} class="flex flex-col gap-4">
  {#if error !== null}
    <Alert.Alert variant="destructive">
      <Alert.AlertDescription>{formatPasskeyError(error)}</Alert.AlertDescription>
    </Alert.Alert>
  {/if}

  {#if loading}
    <div class="flex items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
      <Spinner aria-hidden="true" />
      <span>{i18n.t('common.passkeysLoadingText')}</span>
    </div>
  {/if}

  {#if passkeys.length === 0}
    <p class="py-6 text-center text-sm text-muted-foreground">{i18n.t('common.passkeysEmptyText')}</p>
  {:else}
    <Item.ItemGroup>
      {#each passkeys as passkey (passkey.id)}
        <Item.Item variant="outline" size="sm">
          <Item.ItemHeader>
            <Item.ItemContent>
              <Item.ItemTitle>{passkey.identifier}</Item.ItemTitle>
              <Item.ItemDescription>{formatDate(passkey.createdAt)}</Item.ItemDescription>
            </Item.ItemContent>
            <Item.ItemActions>
              <Button
                variant="destructive"
                size="sm"
                disabled={isLastPasskey || loading}
                aria-label={i18n.t('common.passkeysDeleteButtonAriaLabel', { passkeyIdentifier: passkey.identifier })}
                onclick={() => onDeletePasskey(passkey.id)}
              >
                {i18n.t('common.passkeysDeleteButton')}
              </Button>
            </Item.ItemActions>
          </Item.ItemHeader>
        </Item.Item>
      {/each}
    </Item.ItemGroup>

    {#if isLastPasskey}
      <p class="text-xs text-muted-foreground">{i18n.t('common.passkeysLastWarning')}</p>
    {/if}
  {/if}

  {#if deviceLinkSent}
    <!--
      デバイスリンク URL は画面に表示せず、メール送信済み案内と共有禁止の注意喚起を表示する。
      これにより画面共有や覗き見による secret leakage を防ぐ。
    -->
    <Alert.Alert aria-live="polite">
      <Alert.AlertTitle>{i18n.t('common.passkeysLinkSentTitle')}</Alert.AlertTitle>
      <Alert.AlertDescription>
        {i18n.t('common.passkeysLinkSentDescription')}
        <br />
        {i18n.t('common.passkeysLinkSentExpiry')}
      </Alert.AlertDescription>
    </Alert.Alert>
  {/if}

  <Separator />

  <div class="flex flex-wrap justify-end gap-2">
    <!--
      新しい端末でのログインを有効にするため、再認証後にデバイスリンクを送信するアクション。
      技術用語「デバイスリンク」ではなく、利用者の目的に即したラベルを使用する。
    -->
    <Button variant="outline" disabled={loading} onclick={onSendDeviceLink}>
      {i18n.t('common.passkeysEnableNewDevice')}
    </Button>
    <!--
      現在の端末に直接パスキーを追加するアクション。
      こちらも「add key」という技術用語を避け、利用者の目的に即した表現に統一する。
    -->
    <Button disabled={loading} onclick={onAddPasskey}>
      {i18n.t('common.passkeysEnableThisDevice')}
    </Button>
  </div>
</section>
