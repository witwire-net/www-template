<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import ArticleCardList from './ArticleCardList.svelte';

  interface Props {
    className?: string;
  }

  let { className }: Props = $props();

  const items = [
    {
      title: 'Launch notes for the partner dashboard',
      excerpt: 'A walk through of the rollout plan, guardrails, and review checkpoints.',
      date: 'Mar 12, 2026',
      tag: 'Operations',
      actionContent: openLaunchNotes,
    },
    {
      title: 'How we run weekly release reviews',
      excerpt: 'A reusable cadence for preview environments, sign-off, and changelog ownership.',
      date: 'Mar 07, 2026',
      tag: 'Engineering',
      actionContent: watchReviewReplay,
    },
    {
      title: 'Designing docs that survive migrations',
      excerpt: 'A note on API discoverability while components move toward snippet-first composition.',
      date: 'Mar 03, 2026',
      tag: 'Design Systems',
      action: 'Read article',
    },
  ] as const;
</script>

<ArticleCardList {items} {className} />

{#snippet openLaunchNotes()}
  <Button size="sm" type="button">Open launch notes</Button>
{/snippet}

{#snippet watchReviewReplay()}
  <Button size="sm" variant="secondary" type="button">Watch replay</Button>
{/snippet}
