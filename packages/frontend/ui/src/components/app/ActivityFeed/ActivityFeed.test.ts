import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import ActivityFeed from '@ui/components/app/ActivityFeed/ActivityFeed.svelte';

describe('ActivityFeed', () => {
  describe('デフォルト（非仮想化）モード', () => {
    it('items を通常レンダリングする', () => {
      render(ActivityFeed, {
        props: {
          items: [
            { title: 'First activity', time: '1m ago' },
            { title: 'Second activity', description: 'Details here' },
          ],
        },
      });

      expect(screen.queryByText('First activity')).not.toBeNull();
      expect(screen.queryByText('Second activity')).not.toBeNull();
      expect(screen.queryByText('1m ago')).not.toBeNull();
      expect(screen.queryByText('Details here')).not.toBeNull();
    });

    it('空の items で何もレンダリングしない', () => {
      const { container } = render(ActivityFeed, {
        props: { items: [] },
      });

      const items = container.querySelectorAll('[class]');

      expect(items.length).toBeLessThanOrEqual(1);
    });

    it('virtualize を渡さない場合、role="list" は使わない', () => {
      render(ActivityFeed, {
        props: {
          items: [{ title: 'Test' }],
        },
      });

      expect(screen.queryByRole('list')).toBeNull();
    });
  });

  describe('virtualize prop の型互換', () => {
    it('virtualize が undefined のとき既存の動作を維持する', () => {
      render(ActivityFeed, {
        props: {
          items: [{ title: 'Legacy item', time: '5m ago' }],
          virtualize: undefined,
        },
      });

      expect(screen.queryByText('Legacy item')).not.toBeNull();
    });
  });

  describe('仮想化モード', () => {
    it('virtualize を渡すと role="list" のコンテナをレンダリングする', () => {
      render(ActivityFeed, {
        props: {
          items: [{ title: 'Virtual item' }],
          virtualize: { estimateSize: 60 },
        },
      });

      expect(screen.queryByRole('list')).not.toBeNull();
    });

    it('仮想化コンテナに aria-label="Activity feed" が設定される', () => {
      render(ActivityFeed, {
        props: {
          items: [{ title: 'Item A' }],
          virtualize: { estimateSize: 60 },
        },
      });

      const list = screen.getByRole('list');

      expect(list.getAttribute('aria-label')).toBe('Activity feed');
    });

    it('空の items で仮想化しても listitem をレンダリングしない', () => {
      render(ActivityFeed, {
        props: {
          items: [],
          virtualize: { estimateSize: 60 },
        },
      });

      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    });
  });
});
