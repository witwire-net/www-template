import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import AuditLog from '@ui/components/app/AuditLog/AuditLog.svelte';

describe('AuditLog', () => {
  describe('デフォルト（非仮想化）モード', () => {
    it('entries を通常レンダリングする', () => {
      render(AuditLog, {
        props: {
          entries: [
            { actor: 'Jordan', action: 'updated', target: 'Settings', time: '09:15' },
            { actor: 'Sam', action: 'invited', time: '18:22' },
          ],
        },
      });

      expect(screen.queryByText('Jordan')).not.toBeNull();
      expect(screen.queryByText('updated')).not.toBeNull();
      expect(screen.queryByText('Settings')).not.toBeNull();
      expect(screen.queryByText('Sam')).not.toBeNull();
    });

    it('空の entries で何もレンダリングしない', () => {
      const { container } = render(AuditLog, {
        props: { entries: [] },
      });

      const items = container.querySelectorAll('[class]');

      expect(items.length).toBeLessThanOrEqual(1);
    });

    it('virtualize を渡さない場合、role="list" は使わない', () => {
      render(AuditLog, {
        props: {
          entries: [{ actor: 'Test', action: 'did', time: 'now' }],
        },
      });

      expect(screen.queryByRole('list')).toBeNull();
    });
  });

  describe('virtualize prop の型互換', () => {
    it('virtualize が undefined のとき既存の動作を維持する', () => {
      render(AuditLog, {
        props: {
          entries: [{ actor: 'Legacy', action: 'read', time: 'yesterday' }],
          virtualize: undefined,
        },
      });

      expect(screen.queryByText('Legacy')).not.toBeNull();
    });
  });

  describe('仮想化モード', () => {
    it('virtualize を渡すと role="list" のコンテナをレンダリングする', () => {
      render(AuditLog, {
        props: {
          entries: [{ actor: 'Admin', action: 'deleted', time: '12:00' }],
          virtualize: { estimateSize: 40 },
        },
      });

      expect(screen.queryByRole('list')).not.toBeNull();
    });

    it('仮想化コンテナに aria-label="Audit log" が設定される', () => {
      render(AuditLog, {
        props: {
          entries: [{ actor: 'Test', action: 'viewed', time: '08:00' }],
          virtualize: { estimateSize: 40 },
        },
      });

      const list = screen.getByRole('list');

      expect(list.getAttribute('aria-label')).toBe('Audit log');
    });

    it('空の entries で仮想化しても listitem をレンダリングしない', () => {
      render(AuditLog, {
        props: {
          entries: [],
          virtualize: { estimateSize: 40 },
        },
      });

      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    });
  });
});
