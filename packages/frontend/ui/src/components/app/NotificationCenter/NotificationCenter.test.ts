import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import NotificationCenter from '@ui/components/app/NotificationCenter/NotificationCenter.svelte';

describe('NotificationCenter', () => {
  describe('デフォルト（非仮想化）モード', () => {
    it('items を通常レンダリングする', () => {
      render(NotificationCenter, {
        props: {
          items: [
            { title: 'Report ready', description: 'Available now', time: '1h ago' },
            { title: 'New comment', read: true },
          ],
        },
      });

      expect(screen.queryByText('Report ready')).not.toBeNull();
      expect(screen.queryByText('New comment')).not.toBeNull();
      expect(screen.queryByText('Available now')).not.toBeNull();
    });

    it('空の items で何もレンダリングしない', () => {
      const { container } = render(NotificationCenter, {
        props: { items: [] },
      });

      const items = container.querySelectorAll('[class]');

      expect(items.length).toBeLessThanOrEqual(1);
    });

    it('virtualize を渡さない場合、role="list" は使わない', () => {
      render(NotificationCenter, {
        props: {
          items: [{ title: 'Test' }],
        },
      });

      expect(screen.queryByRole('list')).toBeNull();
    });
  });

  describe('virtualize prop の型互換', () => {
    it('virtualize が undefined のとき既存の動作を維持する', () => {
      render(NotificationCenter, {
        props: {
          items: [{ title: 'Legacy item' }],
          virtualize: undefined,
        },
      });

      expect(screen.queryByText('Legacy item')).not.toBeNull();
    });
  });

  describe('仮想化モード', () => {
    it('virtualize を渡すと role="list" のコンテナをレンダリングする', () => {
      render(NotificationCenter, {
        props: {
          items: [{ title: 'Notif virtual' }],
          virtualize: { estimateSize: 50 },
        },
      });

      expect(screen.queryByRole('list')).not.toBeNull();
    });

    it('仮想化コンテナに aria-label="Notification center" が設定される', () => {
      render(NotificationCenter, {
        props: {
          items: [{ title: 'Notif A' }],
          virtualize: { estimateSize: 50 },
        },
      });

      const list = screen.getByRole('list');

      expect(list.getAttribute('aria-label')).toBe('Notification center');
    });

    it('空の items で仮想化しても listitem をレンダリングしない', () => {
      render(NotificationCenter, {
        props: {
          items: [],
          virtualize: { estimateSize: 50 },
        },
      });

      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    });
  });
});
