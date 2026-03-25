<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import IntegrationCard from './IntegrationCard.svelte';

  interface Props {
    name?: string;
    description?: string;
    status?: 'connected' | 'available' | 'beta';
  }

  let {
    name = 'Linear',
    description = 'Keep issue status and incident follow-up in sync.',
    status = 'connected',
  }: Props = $props();
</script>

<IntegrationCard {name} {description} {status} icon={iconContent} action={actionContent} />

{#snippet iconContent()}
  <span
    style="display: inline-flex; height: 40px; width: 40px; align-items: center; justify-content: center; border-radius: 12px; background: var(--color-text); color: white; font-size: 0.875rem; font-weight: 700;"
  >
    LN
  </span>
{/snippet}

{#snippet actionContent()}
  <Button size="sm" variant={status === 'beta' ? 'secondary' : 'ghost'} type="button">
    {status === 'beta' ? 'Join waitlist' : 'Manage sync'}
  </Button>
{/snippet}
