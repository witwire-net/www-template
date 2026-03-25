<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import {
    IconAlertTriangle,
    IconCheck,
    IconCircle,
    IconInfoCircle,
    IconX,
  } from '@tabler/icons-svelte';

  import styles from './StatusIcon.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    className?: string;
    icon?: typeof IconCircle;
    iconSize?: number;
    iconStroke?: number;
    size?: 'sm' | 'md' | 'lg';
    style?: 'outlined' | 'filled';
    variant?: 'success' | 'warning' | 'error' | 'info' | 'primary';
  };

  let {
    icon = undefined,
    variant = 'success',
    size = 'md',
    style: iconStyle = 'outlined',
    iconSize = undefined,
    iconStroke = 1.5,
    className = undefined,
    ...restProps
  }: Props = $props();

  const defaultIconSizeMap: Record<string, number> = {
    sm: 16,
    md: 24,
    lg: 32,
  };

  const defaultIconMap: Record<string, typeof IconCircle> = {
    success: IconCheck,
    warning: IconAlertTriangle,
    error: IconX,
    info: IconInfoCircle,
    primary: IconCircle,
  };

  const resolvedIconSize = $derived(iconSize ?? (defaultIconSizeMap[size] ?? 24));
  const rootClassName = $derived(
    [
      styles.statusIcon ?? '',
      styles[variant] ?? '',
      styles[size] ?? '',
      styles[iconStyle] ?? '',
      className ?? '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
  const IconComponent = $derived(icon ?? (defaultIconMap[variant] ?? IconCircle));
</script>

<div class={rootClassName} aria-hidden="true" {...restProps}>
  <IconComponent size={resolvedIconSize} stroke={iconStroke} />
</div>
