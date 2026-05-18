<script lang="ts">
  import { goto } from '$app/navigation';
  import { env } from '$env/dynamic/public';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const supportHref = env.PUBLIC_SUPPORT_URL ?? '/login/recovery';
</script>

<AuthLayout>
  <Card class="w-full">
    <CardContent class="flex flex-col items-center gap-4 text-center">
      <span class="inline-flex text-destructive" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z" />
          <path d="M9.5 9.5 14.5 14.5" />
          <path d="M14.5 9.5 9.5 14.5" />
        </svg>
      </span>
      <p class="m-0 text-xs font-semibold uppercase tracking-wide text-destructive">Account unavailable</p>
      <h1 class="m-0 text-2xl font-bold text-center">このアカウントは現在ご利用いただけません</h1>
      <p class="m-0 text-sm text-muted-foreground text-center">
        安全確認のため、このアカウントのセッションを終了しました。解除や詳細確認が必要な場合はサポートまでお問い合わせください。
      </p>
      <p class="m-0 text-sm text-muted-foreground text-center">
        お問い合わせ時は、登録メールアドレスとこの画面が表示された時刻をお伝えください。
      </p>
      <Button
        class="w-full"
        href={supportHref}
        target={supportHref.startsWith('http') ? '_blank' : undefined}
        rel={supportHref.startsWith('http') ? 'noreferrer' : undefined}
      >
        サポート窓口へ進む
      </Button>
      <Button class="w-full" variant="outline" onclick={() => { void goto('/login'); }}>
        別のアカウントでログイン
      </Button>
      <Separator />
      <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトへ</a>
    </CardContent>
  </Card>
</AuthLayout>
