<script lang="ts">
  import styles from './Calendar.module.scss';

  interface CalendarCell {
    day: number | null;
    key: string;
  }

  interface CalendarProps {
    month?: Date;
    onSelect?: (date: Date) => void;
    className?: string;
  }

  const weekdayLabels = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const;

  const getDaysInMonth = (date: Date): number => {
    return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
  };

  let { month = new Date(), onSelect, className }: CalendarProps = $props();

  const year = $derived(month.getFullYear());
  const monthIndex = $derived(month.getMonth());
  const daysInMonth = $derived(getDaysInMonth(month));
  const startDay = $derived(new Date(year, monthIndex, 1).getDay());
  const headerLabel = $derived(`${month.toLocaleString('default', { month: 'long' })} ${String(year)}`);
  const rootClassName = $derived([styles.calendar ?? '', className ?? ''].filter((value) => value !== '').join(' '));
  const cells = $derived(
    Array.from({ length: 42 }, (_, index): CalendarCell => {
      const dayNumber = index - startDay + 1;
      const isActive = dayNumber > 0 && dayNumber <= daysInMonth;

      return {
        day: isActive ? dayNumber : null,
        key: `${String(year)}-${String(monthIndex)}-${String(index)}`,
      };
    })
  );

  const getCellClassName = (isActive: boolean): string => {
    return [styles.cell ?? '', isActive ? (styles.active ?? '') : (styles.empty ?? '')]
      .filter((value) => value !== '')
      .join(' ');
  };

  const handleSelect = (day: number | null): void => {
    if (day === null) {
      return;
    }

    onSelect?.(new Date(year, monthIndex, day));
  };
</script>

<div class={rootClassName}>
  <div class={styles.header ?? ''}>{headerLabel}</div>
  <div class={styles.weekdays ?? ''}>
    {#each weekdayLabels as day (day)}
      <div class={styles.weekday ?? ''}>{day}</div>
    {/each}
  </div>
  <div class={styles.grid ?? ''}>
    {#each cells as cell (cell.key)}
      <button
        type="button"
        class={getCellClassName(cell.day !== null)}
        disabled={cell.day === null}
        onclick={() => {
          handleSelect(cell.day);
        }}
      >
        {cell.day === null ? '' : String(cell.day)}
      </button>
    {/each}
  </div>
</div>
