<script lang="ts">
  /**
   * ユーザーメニューコンポーネント。
   * サイドバー下部に配置し、アカウント表示・切替・追加・設定・ログアウトを集約する。
   *
   * @param sessions - メモリ上に保持されている認証セッション一覧
   * @param activeSessionId - 現在アクティブなセッション ID
   * @param onSwitch - アクティブセッション切替コールバック
   * @param labels - i18n 経由の翻訳済みラベル群
   */
  import { goto } from '$app/navigation';

  import { Avatar, DropdownMenu } from '@www-template/ui/components';

  import type { AuthSessionSummary } from '@www-template/domain/auth';

  interface AppUserMenuProps {
    /** メモリ上に保持されている認証セッション一覧。 */
    sessions: AuthSessionSummary[];
    /** 現在アクティブなセッションの ID。 */
    activeSessionId: string | null;
    /** アクティブセッションを切り替えたときに呼ばれるコールバック。 */
    onSwitch: (sessionId: string) => void;
    /** i18n 経由の翻訳済みラベル群。 */
    labels: {
      userMenuAriaLabel: string;
      switchAccount: string;
      addAccount: string;
      accountSettings: string;
      logout: string;
      accountLabel: string;
    };
  }

  let {
    sessions,
    activeSessionId,
    onSwitch,
    labels,
  }: AppUserMenuProps = $props();

  /**
   * アカウント表示ラベルを生成する。
   * 内部 ID をそのまま表示せず、「アカウント N」形式で表示する。
   *
   * @param index - セッションのインデックス
   * @returns ユーザー向け表示ラベル
   */
  function accountLabel(index: number): string {
    return `${labels.accountLabel} ${index + 1}`;
  }

  /** 複数セッションが存在するかどうか。 */
  let hasMultipleSessions = $derived(sessions.length > 1);

  /** アクティブセッションのインデックス。 */
  let activeIndex = $derived(
    sessions.findIndex((s) => s.sessionId === activeSessionId)
  );

  /** 現在のアカウント表示ラベル。 */
  let currentLabel = $derived(
    activeIndex >= 0 ? accountLabel(activeIndex) : labels.accountLabel
  );

  /** アバター用のフォールバック文字。 */
  let avatarFallback = $derived(
    currentLabel.length > 0 ? currentLabel[0] : '?'
  );

  /**
   * 指定パスへナビゲーションする。
   * DropdownMenuItem の onclick から呼ばれ、メニューは自動的に閉じる。
   *
   * @param path - 遷移先パス
   */
  function navigateTo(path: string): void {
    void goto(path);
  }
</script>

<DropdownMenu.DropdownMenu>
  <DropdownMenu.DropdownMenuTriggerButton
    variant="ghost"
    class="h-auto w-full justify-start gap-2 rounded-sm px-3 py-2 text-left"
    aria-label={labels.userMenuAriaLabel}
  >
    <Avatar.Avatar class="h-6 w-6">
      <Avatar.AvatarFallback>{avatarFallback}</Avatar.AvatarFallback>
    </Avatar.Avatar>
    <span class="truncate text-sm">{currentLabel}</span>
  </DropdownMenu.DropdownMenuTriggerButton>
  <DropdownMenu.DropdownMenuContent align="start" class="w-56">
    {#if hasMultipleSessions}
      <DropdownMenu.DropdownMenuLabel>
        {labels.switchAccount}
      </DropdownMenu.DropdownMenuLabel>
      <DropdownMenu.DropdownMenuRadioGroup
        value={activeSessionId ?? ''}
        onValueChange={(v) => onSwitch(v)}
      >
        {#each sessions as session, index (session.sessionId)}
          <DropdownMenu.DropdownMenuRadioItem value={session.sessionId}>
            {accountLabel(index)}
          </DropdownMenu.DropdownMenuRadioItem>
        {/each}
      </DropdownMenu.DropdownMenuRadioGroup>
      <DropdownMenu.DropdownMenuSeparator />
    {/if}
    <DropdownMenu.DropdownMenuItem onclick={() => navigateTo('/login')}>
      {labels.addAccount}
    </DropdownMenu.DropdownMenuItem>
    <DropdownMenu.DropdownMenuItem onclick={() => navigateTo('/settings/general/language')}>
      {labels.accountSettings}
    </DropdownMenu.DropdownMenuItem>
    <DropdownMenu.DropdownMenuSeparator />
    <DropdownMenu.DropdownMenuItem onclick={() => navigateTo('/logout')}>
      {labels.logout}
    </DropdownMenu.DropdownMenuItem>
  </DropdownMenu.DropdownMenuContent>
</DropdownMenu.DropdownMenu>
