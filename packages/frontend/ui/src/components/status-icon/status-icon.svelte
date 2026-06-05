<!--
  StatusIcon — 認証 surface で繰り返し現れる status / check / alert アイコンの
  共通 primitive。@tabler/icons-svelte の標準アイコンを
  www-template ブランドガイド 06 に従い currentColor と統一ストロークで描画する。
  tone 値に応じて data-tone 属性で CSS 側の色を上書きできる。
-->
<script lang="ts" module>
  export type StatusIconTone = 'neutral' | 'destructive' | 'warning' | 'success' | 'accent';
  /** 利用する Tabler Icons の識別子。 */
  export type StatusIconName = 'check' | 'circle-dot' | 'shield-x' | 'alert-circle' | 'loader';

  export interface StatusIconProps {
    /** 表示する icon の種類。 */
    name: StatusIconName;
    /** 色調。auth.css の .auth-shell__status-icon[data-tone] と連動する。 */
    tone?: StatusIconTone;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { cn } from '@www-template/ui/lib/utils';

  import IconAlertCircle from '@tabler/icons-svelte/icons/alert-circle';
  import IconCircleDot from '@tabler/icons-svelte/icons/circle-dot';
  import IconCircleCheck from '@tabler/icons-svelte/icons/circle-check';
  import IconShieldX from '@tabler/icons-svelte/icons/shield-x';
  import IconLoader from '@tabler/icons-svelte/icons/loader-2';

  let { name, tone = 'neutral', class: className }: StatusIconProps = $props();

  const Component = $derived(
    name === 'check'
      ? IconCircleCheck
      : name === 'shield-x'
        ? IconShieldX
        : name === 'alert-circle'
          ? IconAlertCircle
          : name === 'loader'
            ? IconLoader
            : IconCircleDot
  );
</script>

<span
  data-slot="status-icon"
  data-tone={tone}
  class={cn('inline-flex items-center justify-center', className)}
  aria-hidden="true"
>
  <Component stroke={1.5} size={20} />
</span>
