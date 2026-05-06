<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
</script>

<AuthLayout>
  <Card class="w-full">
    <CardContent class="flex flex-col items-center gap-4 text-center">
      <span class="inline-flex text-warning" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
          <path d="M12 9v4"/>
          <path d="M12 17h.01"/>
        </svg>
      </span>
      <h1 class="m-0 text-2xl font-bold text-center">セッションが切れました</h1>
      <p class="m-0 text-sm text-muted-foreground text-center">
        セキュリティのため、セッションが終了しました。再度ログインしてください。
      </p>
      <Button class="w-full" onclick={() => { void goto('/login'); }}>
        ログインへ
      </Button>
      <Separator />
      <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトへ</a>
    </CardContent>
  </Card>
</AuthLayout>
