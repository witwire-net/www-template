<script lang="ts">
  import SectionHeading from '@ui/components/molecules/SectionHeading/SectionHeading.svelte';

  type HeadingTag = 'h1' | 'h2' | 'h3' | 'h4' | 'p' | 'div';

  interface Props {
    align?: 'left' | 'center';
    titleAs?: HeadingTag;
  }

  let { align = 'left', titleAs = 'h2' }: Props = $props();
</script>

<SectionHeading
  {align}
  {titleAs}
  eyebrowContent={eyebrowContent}
  titleContent={titleContent}
  descriptionContent={descriptionContent}
/>

{#snippet eyebrowContent()}
  <span style="display: inline-flex; align-items: center; gap: 8px;">
    <span
      style="display: inline-flex; min-width: 28px; align-items: center; justify-content: center; border-radius: 999px; background: color-mix(in srgb, var(--color-info) 15%, var(--color-surface)); color: var(--color-info); font-size: 0.75rem; font-weight: 700; padding: 4px 8px;"
    >
      New
    </span>
    Editorial workflow
  </span>
{/snippet}

{#snippet titleContent()}
  <span>
    Build <em>rich headings</em> with migrated snippet content
  </span>
{/snippet}

{#snippet descriptionContent()}
  <span>
    Mix inline formatting, emphasis, and links without dropping back to plain text props.
    <a href="https://storybook.js.org/" target="_blank" rel="noopener noreferrer">View docs</a>
  </span>
{/snippet}
