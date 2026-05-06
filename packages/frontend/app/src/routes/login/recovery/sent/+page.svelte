<script lang="ts">
  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { Card, CardContent, Separator } from '@www-template/ui/components';

  const { data } = useRecoveryFlow();
</script>

<div class="flex flex-col items-center min-h-screen px-4 py-8 font-sans bg-background text-foreground">
  <header class="flex justify-center py-4">
    <a href="/" class="no-underline text-inherit" aria-label="www-template トップページ">
      <span class="font-bold tracking-[0.08em]">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="flex flex-1 w-full max-w-[400px] items-center justify-center py-8">
    <Card class="w-full">
      <CardContent>
        <div class="flex flex-col items-center gap-4 text-center">
          <span class="inline-flex text-success" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <path d="m9 11 3 3L22 4"/>
            </svg>
          </span>
          <h1 class="m-0 text-2xl font-bold text-center">{data.state.sentView.title}</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">{data.state.sentView.description}</p>
          <p class="m-0 text-sm text-muted-foreground text-center">{data.state.sentView.helper}</p>

          <Separator />

          <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">ログインに戻る</a>
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="flex justify-center py-4">
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  </footer>
</div>
