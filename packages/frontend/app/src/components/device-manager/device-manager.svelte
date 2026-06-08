<script lang="ts">
  import { Alert, AlertDescription } from "@www-template/ui/components/alert";
  import { Badge } from "@www-template/ui/components/badge";
  import { Button } from "@www-template/ui/components/button";
  import {
    Item,
    ItemActions,
    ItemContent,
    ItemDescription,
    ItemGroup,
    ItemHeader,
    ItemTitle,
  } from "@www-template/ui/components/item";
  import { Separator } from "@www-template/ui/components/separator";
  import { Spinner } from "@www-template/ui/components/spinner";

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
    /** 他端末一括ログアウトボタンを表示するかどうか。danger zone 分離時に false にする。 */
    showRevokeOthers?: boolean;
    /** 日時 formatter。呼び出し側から注入する。 */
    formatDateTime: (iso: string) => string;
    /** 翻訳済みラベル群。呼び出し側から注入する。 */
    labels: {
      sectionAriaLabel: string;
      loadingText: string;
      emptyText: string;
      loginAtLabel: string;
      lastActiveAtLabel: string;
      currentDeviceBadge: string;
      logoutButtonAriaLabel: (deviceName: string) => string;
      logoutButtonText: string;
      revokeOthersButtonText: string;
    };
  }

  let {
    devices,
    currentSessionId,
    loading,
    error,
    onRevoke,
    onRevokeOthers,
    showRevokeOthers = true,
    formatDateTime,
    labels,
  }: DeviceManagerProps = $props();

  /** 現在のデバイスを除く他のデバイスが存在するかどうか。 */
  let hasOtherDevices = $derived(
    devices.some((d) => d.sessionId !== currentSessionId)
  );
</script>

<section aria-label={labels.sectionAriaLabel} class="flex flex-col gap-4">
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
      <Spinner aria-hidden="true" />
      <span>{labels.loadingText}</span>
    </div>
  {/if}

  {#if devices.length === 0 && !loading}
    <p class="py-6 text-center text-sm text-muted-foreground">
      {labels.emptyText}
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
                  <Badge variant="default" aria-label={labels.currentDeviceBadge}>
                    {labels.currentDeviceBadge}
                  </Badge>
                {/if}
              </ItemTitle>
              <ItemDescription>
                <span class="block text-xs text-muted-foreground">
                  {labels.loginAtLabel}: {formatDateTime(device.loginAt)}
                </span>
                <span class="block text-xs text-muted-foreground">
                  {labels.lastActiveAtLabel}: {formatDateTime(device.lastActiveAt)}
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
                aria-label={labels.logoutButtonAriaLabel(device.deviceName)}
                onclick={() => onRevoke(device.sessionId)}
              >
                {labels.logoutButtonText}
              </Button>
            </ItemActions>
          </ItemHeader>
        </Item>
      {/each}
    </ItemGroup>
  {/if}

  {#if showRevokeOthers}
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
        {labels.revokeOthersButtonText}
      </Button>
    </div>
  {/if}
</section>
