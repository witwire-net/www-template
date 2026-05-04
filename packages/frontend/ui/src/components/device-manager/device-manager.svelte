<script lang="ts">
  import { Alert, AlertDescription } from "@ui/components/alert/index.js";
  import { Badge } from "@ui/components/badge/index.js";
  import { Button } from "@ui/components/button/index.js";
  import {
    Item,
    ItemActions,
    ItemContent,
    ItemDescription,
    ItemGroup,
    ItemHeader,
    ItemTitle,
  } from "@ui/components/item/index.js";
  import { Separator } from "@ui/components/separator/index.js";
  import { Spinner } from "@ui/components/spinner/index.js";

  /** ドメイン層で使用するデバイスセッション表示モデル。 */
  export interface DeviceSession {
    sessionId: string;
    deviceName: string;
    loginAt: string;
    lastActiveAt: string;
    ipHash: string;
    isCurrentSession: boolean;
  }

  interface DeviceManagerProps {
    /** ログイン中のデバイス（セッション）一覧。 */
    devices: DeviceSession[];
    /** 現在のセッション ID。 */
    currentSessionId: string;
    /** 一覧取得や操作中のローディング状態。 */
    loading: boolean;
    /** エラーメッセージ。null の場合はエラーなし。 */
    error: string | null;
    /** 特定デバイスのログアウトアクション。 */
    onRevoke: (sessionId: string) => void;
    /** 他のすべてのデバイスのログアウトアクション。 */
    onRevokeOthers: () => void;
  }

  let {
    devices,
    currentSessionId,
    loading,
    error,
    onRevoke,
    onRevokeOthers,
  }: DeviceManagerProps = $props();

  /**
   * ISO 8601 日時文字列を日本語の読みやすい形式に変換する。
   *
   * @param iso - ISO 8601 形式の日時文字列
   * @returns 日本語の日時表示文字列
   */
  function formatDateTime(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  /** 現在のデバイスを除く他のデバイスが存在するかどうか。 */
  let hasOtherDevices = $derived(
    devices.some((d) => d.sessionId !== currentSessionId)
  );
</script>

<section aria-label="デバイス管理" class="flex flex-col gap-4">
  {#if error !== null}
    <!--
      デバイス一覧取得や操作でエラーが発生した場合、
      汎用的なエラーメッセージを Alert で表示する。
      機密データ（トークン等）は表示しない。
    -->
    <Alert variant="destructive">
      <AlertDescription>{error}</AlertDescription>
    </Alert>
  {/if}

  {#if loading}
    <div class="flex items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
      <Spinner />
      <span>読み込み中…</span>
    </div>
  {/if}

  {#if devices.length === 0 && !loading}
    <p class="py-6 text-center text-sm text-muted-foreground">
      ログイン中のデバイスはありません
    </p>
  {:else}
    <ItemGroup>
      {#each devices as device (device.sessionId)}
        <Item variant="outline" size="sm">
          <ItemHeader>
            <ItemContent>
              <ItemTitle class="flex items-center gap-2">
                {device.deviceName}
                {#if device.sessionId === currentSessionId}
                  <!--
                    現在操作中のデバイスには「現在のデバイス」バッジを表示する。
                    これにより、ユーザーがどのセッションが自分の端末かを判別できる。
                  -->
                  <Badge variant="default" aria-label="現在のデバイス">
                    現在のデバイス
                  </Badge>
                {/if}
              </ItemTitle>
              <ItemDescription>
                <span class="block text-xs text-muted-foreground">
                  ログイン: {formatDateTime(device.loginAt)}
                </span>
                <span class="block text-xs text-muted-foreground">
                  最終アクティブ: {formatDateTime(device.lastActiveAt)}
                </span>
              </ItemDescription>
            </ItemContent>
            <ItemActions>
              <!--
                各デバイスに個別のログアウトボタンを配置する。
                クリック時に onRevoke コールバックを発火し、
                親コンポーネント（ページ）がドメイン hook 経由で API を呼び出す。
              -->
              <Button
                variant="destructive"
                size="sm"
                disabled={loading}
                aria-label="{device.deviceName} をログアウト"
                onclick={() => onRevoke(device.sessionId)}
              >
                ログアウト
              </Button>
            </ItemActions>
          </ItemHeader>
        </Item>
      {/each}
    </ItemGroup>
  {/if}

  <Separator />

  <div class="flex flex-wrap justify-end gap-2">
    <!--
      他のすべてのデバイスを一括ログアウトするボタン。
      不審なアクティビティを検知した際に、現在の端末のみを残して他を無効化するための緊急アクション。
      他デバイスが存在しない場合は無効化する。
    -->
    <Button
      variant="destructive"
      size="sm"
      disabled={loading || !hasOtherDevices}
      onclick={onRevokeOthers}
    >
      他のすべてのデバイスをログアウト
    </Button>
  </div>
</section>
