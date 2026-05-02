<script lang="ts">
  import type { PasskeyItem } from '@www-template/domain/auth';
  import { Alert, Button, Item, Separator, Spinner } from '@www-template/ui/components';

  interface PasskeyListProps {
    /** List of registered passkeys. */
    passkeys: PasskeyItem[];
    /** Whether any async operation is in progress. */
    loading: boolean;
    /** Error message to display, or null. */
    error: string | null;
    /** Whether an OTP has been issued for new-device login enablement. */
    otpIssued: boolean;
    /** Called when the user clicks "この端末でログインを有効にする" button. */
    onAddPasskey: () => void;
    /** Called when the user clicks "削除" on a passkey row. */
    onDeletePasskey: (id: string) => void;
    /** Called when the user clicks "新しい端末でログインを有効にする" to issue OTP. */
    onIssueOtp: () => void;
  }

  let {
    passkeys,
    loading,
    error,
    otpIssued,
    onAddPasskey,
    onDeletePasskey,
    onIssueOtp,
  }: PasskeyListProps = $props();

  let isLastPasskey = $derived(passkeys.length === 1);

  function formatDate(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }
</script>

<section aria-label="パスキー管理" class="flex flex-col gap-4">
  {#if error !== null}
    <Alert.Alert variant="destructive">
      <Alert.AlertDescription>{error}</Alert.AlertDescription>
    </Alert.Alert>
  {/if}

  {#if loading}
    <div class="flex items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
      <Spinner />
      <span>処理中…</span>
    </div>
  {/if}

  {#if passkeys.length === 0}
    <p class="py-6 text-center text-sm text-muted-foreground">パスキーが登録されていません</p>
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
                aria-label="{passkey.identifier} を削除"
                onclick={() => onDeletePasskey(passkey.id)}
              >
                削除
              </Button>
            </Item.ItemActions>
          </Item.ItemHeader>
        </Item.Item>
      {/each}
    </Item.ItemGroup>

    {#if isLastPasskey}
      <p class="text-xs text-muted-foreground">最後のパスキーは削除できません</p>
    {/if}
  {/if}

  {#if otpIssued}
    <!--
      平文 OTP は画面に表示せず、メール送信済み案内と共有禁止の注意喚起を表示する。
      これにより画面共有や覗き見による secret leakage を防ぐ。
    -->
    <Alert.Alert aria-live="polite">
      <Alert.AlertTitle>ログイン有効化コードを送信しました</Alert.AlertTitle>
      <Alert.AlertDescription>
        登録済みのメールアドレス宛にコードを送信しました。新しい端末でメールアドレスとコードを入力してください。コードは第三者と共有しないでください。
        <br />
        有効期限: 5分
      </Alert.AlertDescription>
    </Alert.Alert>
  {/if}

  <Separator />

  <div class="flex flex-wrap justify-end gap-2">
    <!--
      新しい端末でのログインを有効にするため、再認証後に OTP を発行するアクション。
      技術用語「OTPを発行」ではなく、利用者の目的に即したラベルを使用する。
    -->
    <Button variant="outline" disabled={loading} onclick={onIssueOtp}>
      新しい端末でログインを有効にする
    </Button>
    <!--
      現在の端末に直接パスキーを追加するアクション。
      こちらも「add key」という技術用語を避け、利用者の目的に即した表現に統一する。
    -->
    <Button disabled={loading} onclick={onAddPasskey}>
      この端末でログインを有効にする
    </Button>
  </div>
</section>
