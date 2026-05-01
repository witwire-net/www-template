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
    /** Current OTP code to display, or null when not issued. */
    otp: string | null;
    /** Called when the user clicks "パスキーを追加" button. */
    onAddPasskey: () => void;
    /** Called when the user clicks "削除" on a passkey row. */
    onDeletePasskey: (id: string) => void;
    /** Called when the user clicks "OTPを発行". */
    onIssueOtp: () => void;
  }

  let {
    passkeys,
    loading,
    error,
    otp,
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

  {#if otp !== null}
    <div class="flex flex-col gap-1 rounded-md border border-border bg-muted p-4" aria-live="polite">
      <span class="text-xs text-muted-foreground">発行済み OTP</span>
      <span class="font-mono text-2xl font-bold tracking-widest">{otp}</span>
      <span class="text-sm text-muted-foreground">このコードを新しい端末で入力してください</span>
    </div>
  {/if}

  <Separator />

  <div class="flex flex-wrap justify-end gap-2">
    <Button variant="outline" disabled={loading} onclick={onIssueOtp}>
      OTPを発行
    </Button>
    <Button disabled={loading} onclick={onAddPasskey}>
      + パスキーを追加
    </Button>
  </div>
</section>
