import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import List from '@ui/components/data/List/List.svelte';

describe('List', () => {
  describe('デフォルト（非仮想化）モード', () => {
    it('items を通常レンダリングする', () => {
      render(List, {
        props: {
          items: [
            { title: 'Deploy', description: 'Production build', meta: '2m ago' },
            { title: 'Invoice', description: 'Team plan', meta: '1h ago' },
          ],
        },
      });

      expect(screen.queryByText('Deploy')).not.toBeNull();
      expect(screen.queryByText('Invoice')).not.toBeNull();
      expect(screen.queryByText('Production build')).not.toBeNull();
    });

    it('空の items で何もレンダリングしない', () => {
      const { container } = render(List, {
        props: { items: [] },
      });

      const items = container.querySelectorAll('[class]');

      expect(items.length).toBeLessThanOrEqual(1);
    });

    it('items を渡さなくても動作する（デフォルト空配列）', () => {
      const { container } = render(List, { props: {} });

      expect(container).toBeTruthy();
    });

    it('virtualize を渡さない場合、role="list" は使わない', () => {
      render(List, {
        props: {
          items: [{ title: 'Test' }],
        },
      });

      expect(screen.queryByRole('list')).toBeNull();
    });
  });

  describe('virtualize prop の型互換', () => {
    it('virtualize が undefined のとき既存の動作を維持する', () => {
      render(List, {
        props: {
          items: [{ title: 'Legacy item', description: 'Old style' }],
          virtualize: undefined,
        },
      });

      expect(screen.queryByText('Legacy item')).not.toBeNull();
    });
  });

  describe('仮想化モード', () => {
    it('virtualize を渡すと role="list" のコンテナをレンダリングする', () => {
      render(List, {
        props: {
          items: [{ title: 'Virtual item' }],
          virtualize: { estimateSize: 48 },
        },
      });

      expect(screen.queryByRole('list')).not.toBeNull();
    });

    it('仮想化コンテナに aria-label="List" が設定される', () => {
      render(List, {
        props: {
          items: [{ title: 'Labeled item' }],
          virtualize: { estimateSize: 48 },
        },
      });

      const list = screen.getByRole('list');

      expect(list.getAttribute('aria-label')).toBe('List');
    });

    it('空の items で仮想化しても listitem をレンダリングしない', () => {
      render(List, {
        props: {
          items: [],
          virtualize: { estimateSize: 48 },
        },
      });

      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    });
  });

  describe('className の適用', () => {
    it('カスタム className を適用する', () => {
      const { container } = render(List, {
        props: {
          items: [{ title: 'Item' }],
          className: 'custom-class',
        },
      });

      expect(container.querySelector('.custom-class')).not.toBeNull();
    });
  });
});
