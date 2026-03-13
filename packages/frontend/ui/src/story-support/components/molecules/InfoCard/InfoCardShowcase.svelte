<script lang="ts">
  import { IconSparkles } from '@tabler/icons-svelte';
  import type { Snippet } from 'svelte';

  import Badge from '@ui/components/atoms/Badge/Badge.svelte';
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import InfoCard from '@ui/components/molecules/InfoCard/InfoCard.svelte';

  interface Props {
    title?: unknown;
    description?: unknown;
    icon?: typeof IconSparkles;
    meta?: unknown;
    action?: unknown;
    className?: string;
    iconContent?: Snippet;
    metaContent?: Snippet;
    titleContent?: Snippet;
    descriptionContent?: Snippet;
    actionContent?: Snippet;
  }

  let {
    title = 'Automated review',
    description = 'Compose atoms inside molecule slots while keeping the card API compact.',
    icon = IconSparkles,
  }: Props = $props();
</script>

<InfoCard
  {title}
  {description}
  {icon}
  metaContent={metaContent}
  actionContent={actionContent}
>
</InfoCard>

{#snippet metaContent()}
  <Badge variant="info" size="sm">New</Badge>
{/snippet}

{#snippet actionContent()}
  <Button variant="secondary" size="sm" type="button">Open details</Button>
{/snippet}
