<script lang="ts">
  import type { Snippet } from 'svelte';

  import Avatar from '@ui/components/atoms/Avatar/Avatar.svelte';
  import Badge from '@ui/components/atoms/Badge/Badge.svelte';

  import styles from './UserList.module.scss';

  type BadgeVariant = 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';
  type Renderable = Snippet | string | number | null | undefined;

  interface UserListBadge {
    label: string;
    variant?: BadgeVariant;
  }

  interface UserListCell {
    id?: string | number;
    align?: 'start' | 'center' | 'end';
    badge?: UserListBadge;
    className?: string;
    content?: Renderable;
  }

  interface UserListItem {
    id: string | number;
    title: string;
    subtitle?: string;
    avatarLabel?: string;
    leading?: Snippet;
    cells?: readonly UserListCell[];
  }

  interface LegacyUserListItem {
    id: string | number;
    name: string;
    email?: string;
    role?: string;
    status?: string;
    avatar?: Snippet;
    action?: Renderable;
  }

  interface Props {
    items?: readonly UserListItem[];
    users?: readonly LegacyUserListItem[];
  }

  let { items = undefined, users = [] }: Props = $props();

  const hasValue = (value?: string): boolean => {
    return typeof value === 'string' && value !== '';
  };

  const hasRenderable = (value: Renderable): boolean => {
    return value !== undefined && value !== null;
  };

  const isSnippet = (value: Renderable): value is Snippet => {
    return typeof value === 'function';
  };

  const getTextContent = (value: Renderable): string => {
    if (typeof value === 'string') {
      return value;
    }

    if (typeof value === 'number') {
      return String(value);
    }

    return '';
  };

  const getStatusVariant = (status?: string): BadgeVariant => {
    return status === 'Active' ? 'success' : 'neutral';
  };

  const getCellAlignmentClassName = (align?: UserListCell['align']): string => {
    if (align === 'center') {
      return styles.alignCenter ?? '';
    }

    if (align === 'end') {
      return styles.alignEnd ?? '';
    }

    return styles.alignStart ?? '';
  };

  const joinClassName = (...values: (string | undefined)[]): string => {
    return values.filter((value) => value !== undefined && value !== '').join(' ');
  };

  const normalizeLegacyUser = (user: LegacyUserListItem): UserListItem => {
    const cells: UserListCell[] = [];

    if (hasValue(user.role)) {
      cells.push({
        id: 'role',
        badge: {
          label: user.role ?? '',
          variant: 'neutral',
        },
      });
    }

    if (hasValue(user.status)) {
      cells.push({
        id: 'status',
        badge: {
          label: user.status ?? '',
          variant: getStatusVariant(user.status),
        },
      });
    }

    if (hasRenderable(user.action)) {
      cells.push({
        id: 'action',
        align: 'end',
        content: user.action,
      });
    }

    return {
      id: user.id,
      title: user.name,
      subtitle: user.email,
      avatarLabel: user.name,
      leading: user.avatar,
      cells,
    };
  };

  const normalizedItems = $derived(
    items ?? users.map((user) => {
      return normalizeLegacyUser(user);
    })
  );

  const columnCount = $derived(
    normalizedItems.reduce((max, item) => {
      return Math.max(max, item.cells?.length ?? 0);
    }, 0)
  );

  const rowTemplate = $derived(
    columnCount > 0
      ? `grid-template-columns: minmax(0, 2fr) repeat(${columnCount}, minmax(0, 1fr));`
      : 'grid-template-columns: minmax(0, 1fr);'
  );
</script>

<div class={styles.list ?? ''}>
  {#each normalizedItems as item (item.id)}
    <div class={styles.row ?? ''} style={rowTemplate}>
      <div class={styles.user ?? ''}>
        <div class={styles.avatar ?? ''}>
          {#if item.leading !== undefined}
            {@render item.leading()}
          {:else}
            <Avatar name={item.avatarLabel ?? item.title} size="sm" />
          {/if}
        </div>
        <div class={styles.identity ?? ''}>
          <div class={styles.title ?? ''}>{item.title}</div>
          {#if hasValue(item.subtitle)}
            <div class={styles.subtitle ?? ''}>{item.subtitle}</div>
          {/if}
        </div>
      </div>
      {#each item.cells ?? [] as cell, cellIndex (String(cell.id ?? `${item.id}-${cellIndex}`))}
        <div class={joinClassName(styles.cell, getCellAlignmentClassName(cell.align), cell.className)}>
          {#if cell.badge !== undefined && hasValue(cell.badge.label)}
            <Badge variant={cell.badge.variant ?? 'neutral'} size="sm">{cell.badge.label}</Badge>
          {:else if hasRenderable(cell.content)}
            {#if isSnippet(cell.content)}
              {@render cell.content()}
            {:else}
              {getTextContent(cell.content)}
            {/if}
          {/if}
        </div>
      {/each}
    </div>
  {/each}
</div>
