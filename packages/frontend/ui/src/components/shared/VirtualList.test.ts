import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import { VirtualList } from '@ui/components/shared';

import type { Snippet } from 'svelte';

/** Noop snippet for tests that only inspect the container. */
const noop = ((_index: number) => undefined) as unknown as Snippet<[number]>;

describe('VirtualList', () => {
  it('スクロールコンテナを role="list" でレンダリングする', () => {
    render(VirtualList, {
      props: {
        count: 5,
        estimateSize: 40,
        row: noop,
      },
    });

    expect(screen.queryByRole('list')).not.toBeNull();
  });

  it('aria-label を設定できる', () => {
    render(VirtualList, {
      props: {
        count: 3,
        estimateSize: 40,
        ariaLabel: 'Test list',
        row: noop,
      },
    });

    const list = screen.getByRole('list');

    expect(list.getAttribute('aria-label')).toBe('Test list');
  });

  it('className をスクロールコンテナに適用する', () => {
    const { container } = render(VirtualList, {
      props: {
        count: 2,
        estimateSize: 40,
        className: 'custom-scroll',
        row: noop,
      },
    });

    expect(container.querySelector('.custom-scroll')).not.toBeNull();
  });

  it('height を指定するとスクロールコンテナのスタイルに反映される', () => {
    render(VirtualList, {
      props: {
        count: 10,
        estimateSize: 40,
        height: 300,
        row: noop,
      },
    });

    const list = screen.getByRole('list');

    expect(list.style.height).toBe('300px');
  });

  it('count が 0 のとき listitem をレンダリングしない', () => {
    render(VirtualList, {
      props: {
        count: 0,
        estimateSize: 40,
        row: noop,
      },
    });

    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
  });
});
