import { describe, expect, it } from 'vitest';

import { DEFAULT_HEIGHT, DEFAULT_OVERSCAN } from '@ui/components/shared';
import type { VirtualizeOptions } from '@ui/components/shared';

describe('virtual shared helpers', () => {
  describe('DEFAULT_OVERSCAN', () => {
    it('デフォルトの overscan は 5', () => {
      expect(DEFAULT_OVERSCAN).toBe(5);
    });
  });

  describe('DEFAULT_HEIGHT', () => {
    it('デフォルトの height は 400', () => {
      expect(DEFAULT_HEIGHT).toBe(400);
    });
  });

  describe('VirtualizeOptions type', () => {
    it('必須プロパティは estimateSize のみ', () => {
      const minimal: VirtualizeOptions = { estimateSize: 50 };

      expect(minimal.estimateSize).toBe(50);
      expect(minimal.overscan).toBeUndefined();
      expect(minimal.height).toBeUndefined();
      expect(minimal.gap).toBeUndefined();
    });

    it('すべてのプロパティを設定できる', () => {
      const keyFn = (index: number): string => `key-${String(index)}`;

      const full: VirtualizeOptions = {
        estimateSize: 60,
        overscan: 10,
        height: 600,
        gap: 12,
        getItemKey: keyFn,
      };

      expect(full.estimateSize).toBe(60);
      expect(full.overscan).toBe(10);
      expect(full.height).toBe(600);
      expect(full.gap).toBe(12);
      expect(full.getItemKey).toBe(keyFn);
    });
  });
});
