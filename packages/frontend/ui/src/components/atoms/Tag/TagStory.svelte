<script lang="ts">
  import { IconBolt } from '@tabler/icons-svelte';
  import Tag from './Tag.svelte';

  let {
    variant = 'default',
    size = 'md',
    showIcon = false,
    text = 'Tag',
  }: {
    showIcon?: boolean;
    size?: 'sm' | 'md';
    text?: string;
    variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  } = $props();
</script>

<Tag {variant} {size} icon={showIcon ? IconBolt : undefined}>{text}</Tag>
