<script lang="ts">
  import { DropdownMenu } from '@www-template/ui/components';

  import type { AuthSessionSummary } from '@www-template/domain/auth';

  interface AccountSwitcherProps {
    /** メモリ上に保持されている認証セッションの一覧。 */
    sessions: AuthSessionSummary[];
    /** 現在アクティブなセッションの ID。 */
    activeSessionId: string | null;
    /** アクティブセッションを切り替えたときに呼ばれるコールバック。 */
    onSwitch: (sessionId: string) => void;
  }

  let { sessions, activeSessionId, onSwitch }: AccountSwitcherProps = $props();

  /**
   * アカウント識別子を短縮表示用に整形する。
   * ULID の先頭 8 文字 + 末尾 4 文字を残し、中間を「…」で省略する。
   *
   * @param id - 元の ULID 文字列
   * @returns 短縮表示用文字列
   */
  function abbreviateId(id: string): string {
    if (id.length <= 14) {
      return id;
    }
    return `${id.slice(0, 8)}…${id.slice(-4)}`;
  }

  /** アクティブセッションのオブジェクトを取得する。 */
  let activeSession = $derived(sessions.find((s) => s.sessionId === activeSessionId) ?? null);

  /** 複数セッションが存在するかどうか。 */
  let hasMultipleSessions = $derived(sessions.length > 1);
</script>

{#if hasMultipleSessions && activeSession != null}
  <!--
    複数アカウントがログインしている場合にのみ表示する切り替えコントロール。
    DropdownMenuRadioGroup を使用し、選択状態の視覚表現を UI primitive に委譲する。
    切り替えはメモリ上の activeSessionId のみを変更し、永続化や再認証を行わない。
  -->
  <DropdownMenu.DropdownMenu>
    <DropdownMenu.DropdownMenuTriggerButton aria-label="アカウントを切り替える">
      {abbreviateId(activeSession.accountId)}
    </DropdownMenu.DropdownMenuTriggerButton>
    <DropdownMenu.DropdownMenuContent align="end">
      <DropdownMenu.DropdownMenuRadioGroup value={activeSessionId ?? ''} onValueChange={(v) => onSwitch(v)}>
        {#each sessions as session (session.sessionId)}
          <DropdownMenu.DropdownMenuRadioItem value={session.sessionId}>
            {abbreviateId(session.accountId)}
          </DropdownMenu.DropdownMenuRadioItem>
        {/each}
      </DropdownMenu.DropdownMenuRadioGroup>
    </DropdownMenu.DropdownMenuContent>
  </DropdownMenu.DropdownMenu>
{/if}
