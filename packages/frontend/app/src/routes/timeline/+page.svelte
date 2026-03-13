<script lang="ts">
  import StatusCard from '../../lib/status/StatusCard.svelte';

  type TimelineCard = {
    body: string;
    title: string;
  };

  const cards: TimelineCard[] = [
    {
      title: 'Public routes stay outside `/app/*`',
      body: 'SvelteKit の公開ページは SSR 前提で残し、認証 shell と path で分離します。',
    },
    {
      title: 'Auth routes switch to CSR shell',
      body: '`src/routes/app/+layout.ts` で `/app/*` 全体を `ssr = false` に固定します。',
    },
    {
      title: 'Domain owns state and I/O',
      body: '一覧取得と作成処理は app から切り離し、domain facade/store に寄せます。',
    },
  ];
</script>

<section class="timeline-page">
  <div class="headline">
    <div class="eyebrow">PUBLIC TIMELINE</div>
    <h1>公開 route は `/timeline` に固定</h1>
    <p>SEO と HTML 応答を優先する public 入口です。認証側の操作系 UI はここに混ぜません。</p>
  </div>

  <div class="card-grid">
    {#each cards as card (card.title)}
      <article>
        <h2>{card.title}</h2>
        <p>{card.body}</p>
      </article>
    {/each}
  </div>

  <StatusCard />
</section>

<style>
  .timeline-page {
    display: grid;
    gap: 1.5rem;
  }

  .headline,
  article {
    padding: 1.4rem;
    border: 1px solid rgba(15, 23, 42, 0.08);
    border-radius: 1.4rem;
    background: rgba(255, 255, 255, 0.82);
  }

  .eyebrow {
    margin-bottom: 0.35rem;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.2em;
    color: #0284c7;
  }

  h1,
  h2,
  p {
    margin: 0;
  }

  h1 {
    font-size: clamp(2rem, 5vw, 3.4rem);
    line-height: 1;
    margin-bottom: 0.75rem;
  }

  h2 {
    margin-bottom: 0.6rem;
    font-size: 1.1rem;
  }

  p {
    color: #475569;
    line-height: 1.6;
  }

  .card-grid {
    display: grid;
    gap: 1rem;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }
</style>
