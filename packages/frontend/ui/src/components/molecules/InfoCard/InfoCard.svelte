<script lang="ts">
  import type { IconCircle } from '@tabler/icons-svelte';
  import type { Snippet } from 'svelte';

  import Icon from '@ui/components/atoms/Icon/Icon.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';

  import styles from './InfoCard.module.scss';

  interface InfoCardProps {
    title?: unknown;
    description?: unknown;
    icon?: typeof IconCircle;
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
    title,
    description,
    icon,
    meta,
    action,
    className,
    iconContent,
    metaContent,
    titleContent,
    descriptionContent,
    actionContent,
  }: InfoCardProps = $props();

  const renderFallback = (value: unknown): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const rootClassName = $derived([styles.card ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const hasIcon = $derived(iconContent !== undefined || icon !== undefined);
  const hasMeta = $derived(metaContent !== undefined || meta !== undefined);
  const hasDescription = $derived(descriptionContent !== undefined || description !== undefined);
  const hasAction = $derived(actionContent !== undefined || action !== undefined);
</script>

<Card className={rootClassName}>
  <CardBody className={styles.body}>
    {#if hasIcon || hasMeta}
      <div class={styles.top}>
        {#if hasIcon}
          <div class={styles.icon}>
            {#if iconContent !== undefined}
              {@render iconContent()}
            {:else if icon !== undefined}
              <Icon {icon} />
            {/if}
          </div>
        {/if}
        {#if hasMeta}
          <div class={styles.meta}>
            {#if metaContent !== undefined}
              {@render metaContent()}
            {:else}
              {renderFallback(meta)}
            {/if}
          </div>
        {/if}
      </div>
    {/if}
    <div class={styles.title}>
      {#if titleContent !== undefined}
        {@render titleContent()}
      {:else}
        {renderFallback(title)}
      {/if}
    </div>
    {#if hasDescription}
      <div class={styles.description}>
        {#if descriptionContent !== undefined}
          {@render descriptionContent()}
        {:else}
          {renderFallback(description)}
        {/if}
      </div>
    {/if}
    {#if hasAction}
      <div class={styles.action}>
        {#if actionContent !== undefined}
          {@render actionContent()}
        {:else}
          {renderFallback(action)}
        {/if}
      </div>
    {/if}
  </CardBody>
</Card>
