<script lang="ts">
  import styles from './Timeline.module.scss';

  interface TimelineEvent {
    title: string;
    description?: string;
    time?: string;
    status?: 'default' | 'success' | 'warning' | 'error' | 'info';
  }

  interface TimelineProps {
    events?: readonly TimelineEvent[];
    className?: string;
  }

  let { events = [], className }: TimelineProps = $props();

  const rootClassName = $derived([styles.timeline ?? '', className ?? ''].filter((value) => value !== '').join(' '));

  const getStatusClassName = (status: NonNullable<TimelineEvent['status']>): string => {
    switch (status) {
      case 'success': {
        return styles.success ?? '';
      }
      case 'warning': {
        return styles.warning ?? '';
      }
      case 'error': {
        return styles.error ?? '';
      }
      case 'info': {
        return styles.info ?? '';
      }
      case 'default':
      default: {
        return styles.default ?? '';
      }
    }
  };

  const getEventKey = (event: TimelineEvent): string => {
    const safeDescription = event.description ?? '';
    const safeTime = event.time ?? '';

    return `${event.title}-${safeDescription}-${safeTime}`;
  };
</script>

<div class={rootClassName}>
  {#each events as event (getEventKey(event))}
    <div class={styles.event ?? ''}>
      <div class={[styles.dot ?? '', getStatusClassName(event.status ?? 'default')].filter((value) => value !== '').join(' ')}></div>
      <div class={styles.content ?? ''}>
        <div class={styles.title ?? ''}>{event.title}</div>
        {#if event.description !== undefined && event.description !== ''}
          <div class={styles.description ?? ''}>{event.description}</div>
        {/if}
      </div>
      {#if event.time !== undefined && event.time !== ''}
        <div class={styles.time ?? ''}>{event.time}</div>
      {/if}
    </div>
  {/each}
</div>
