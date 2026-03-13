<script lang="ts">
  import styles from './Sparkline.module.scss';

  interface SparklineProps {
    data?: readonly number[];
    width?: number;
    height?: number;
    className?: string;
  }

  interface SparklinePoint {
    x: number;
    y: number;
  }

  let { data = [], width = 120, height = 32, className }: SparklineProps = $props();

  const rootClassName = $derived([styles.sparkline ?? '', className ?? ''].filter((value) => value !== '').join(' '));
  const max = $derived(data.length > 0 ? Math.max(...data) : 1);
  const min = $derived(data.length > 0 ? Math.min(...data) : 0);
  const range = $derived(max - min === 0 ? 1 : max - min);
  const divisor = $derived(data.length > 1 ? data.length - 1 : 1);
  const points = $derived(
    data.map((value, index): SparklinePoint => {
      const x = (index / divisor) * width;
      const y = height - ((value - min) / range) * height;

      return { x, y };
    })
  );
  const path = $derived(
    points
      .map((point, index) => `${index === 0 ? 'M' : 'L'} ${String(point.x)} ${String(point.y)}`)
      .join(' ')
  );
</script>

<svg class={rootClassName} {width} {height} viewBox={`0 0 ${String(width)} ${String(height)}`}>
  <path d={path} class={styles.line ?? ''}></path>
</svg>
