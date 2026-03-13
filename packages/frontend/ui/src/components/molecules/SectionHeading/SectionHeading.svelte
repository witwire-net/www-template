<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './SectionHeading.module.scss';

  type HeadingTag = 'h1' | 'h2' | 'h3' | 'h4' | 'p' | 'div';

  interface SectionHeadingProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
    eyebrow?: unknown;
    title?: unknown;
    description?: unknown;
    align?: 'left' | 'center';
    titleAs?: HeadingTag;
    className?: string;
    eyebrowClassName?: string;
    titleClassName?: string;
    descriptionClassName?: string;
    eyebrowContent?: Snippet;
    titleContent?: Snippet;
    descriptionContent?: Snippet;
  }

  let {
    eyebrow,
    title,
    description,
    align = 'left',
    titleAs = 'h2',
    className,
    eyebrowClassName,
    titleClassName,
    descriptionClassName,
    eyebrowContent,
    titleContent,
    descriptionContent,
    ...restProps
  }: SectionHeadingProps = $props();

  const renderFallback = (value: unknown): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const alignClassName = $derived(align === 'center' ? styles.center : styles.left);
  const rootClassName = $derived([styles.heading ?? '', alignClassName ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const eyebrowRootClassName = $derived([styles.eyebrow ?? '', eyebrowClassName ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const titleRootClassName = $derived([styles.title ?? '', titleClassName ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const descriptionRootClassName = $derived([
    styles.description ?? '',
    descriptionClassName ?? '',
  ]
    .filter((value) => value !== '')
    .join(' '));
  const hasEyebrow = $derived(eyebrowContent !== undefined || eyebrow !== undefined);
  const hasTitle = $derived(titleContent !== undefined || title !== undefined);
  const hasDescription = $derived(descriptionContent !== undefined || description !== undefined);
</script>

<div class={rootClassName} {...restProps}>
  {#if hasEyebrow}
    <div class={eyebrowRootClassName}>
      {#if eyebrowContent !== undefined}
        {@render eyebrowContent()}
      {:else}
        {renderFallback(eyebrow)}
      {/if}
    </div>
  {/if}

  {#if hasTitle}
    <svelte:element this={titleAs} class={titleRootClassName}>
      {#if titleContent !== undefined}
        {@render titleContent()}
      {:else}
        {renderFallback(title)}
      {/if}
    </svelte:element>
  {/if}

  {#if hasDescription}
    <p class={descriptionRootClassName}>
      {#if descriptionContent !== undefined}
        {@render descriptionContent()}
      {:else}
        {renderFallback(description)}
      {/if}
    </p>
  {/if}
</div>
