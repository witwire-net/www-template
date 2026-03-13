<script lang="ts">
  import { IconCircle } from '@tabler/icons-svelte';

  import styles from './Icon.module.scss';

  type IconComponentProps = {
    class?: string;
    color?: string;
    size?: string | number;
    stroke?: string | number;
    style?: string;
    title?: string;
    'aria-hidden'?: boolean;
    'aria-label'?: string;
    'aria-labelledby'?: string;
    role?: 'img' | 'presentation';
  };

  type Props = IconComponentProps & {
    className?: string;
    icon?: typeof IconCircle;
  } & Record<string, unknown>;

  let {
    icon = undefined,
    size = 20,
    stroke = 1.8,
    color = undefined,
    title = undefined,
    class: classProp = undefined,
    className = undefined,
    style = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.icon ?? '', classProp ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );
  const mergedStyle = $derived(
    [style ?? '', color !== undefined ? `color: ${color};` : '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const IconComponent = $derived(icon ?? IconCircle);
  const ariaLabel = $derived(
    typeof restProps['aria-label'] === 'string' && restProps['aria-label'] !== ''
      ? restProps['aria-label']
      : undefined
  );
  const ariaLabelledby = $derived(
    typeof restProps['aria-labelledby'] === 'string' && restProps['aria-labelledby'] !== ''
      ? restProps['aria-labelledby']
      : undefined
  );
  const hasTitle = $derived(typeof title === 'string' && title !== '');
  const resolvedTitle = $derived(hasTitle ? title : ariaLabel);
  const accessibleLabel = $derived(ariaLabel ?? (hasTitle ? title : undefined));
  const isDecorative = $derived(accessibleLabel === undefined && ariaLabelledby === undefined);
  const iconProps = $derived({
    class: rootClassName,
    style: mergedStyle === '' ? undefined : mergedStyle,
    size,
    stroke,
    color,
    title: resolvedTitle,
    'aria-label': accessibleLabel,
    role: isDecorative ? 'presentation' : 'img',
    'aria-hidden': isDecorative ? true : undefined,
  });
</script>

<IconComponent {...iconProps} {...restProps} />
