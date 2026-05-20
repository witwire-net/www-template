import { describe, expect, it } from 'vitest';

import { createFormatters } from './formatters';

describe('formatters', () => {
  it('locale 固定で formatter を返す', () => {
    const formatters = createFormatters('en', {
      date: { dateStyle: 'short', timeZone: 'UTC' },
      dateTime: { dateStyle: 'short', timeStyle: 'short', timeZone: 'UTC' },
      number: { minimumFractionDigits: 2 },
      list: { style: 'long', type: 'conjunction' },
    });

    expect(formatters.number(1234.5)).toBe('1,234.50');
    expect(formatters.list(['one', 'two'])).toBe('one and two');
    expect(formatters.date('2024-01-02T00:00:00Z')).toBe('1/2/24');
    expect(formatters.dateTime('2024-01-02T03:04:00Z')).toBe('1/2/24, 3:04 AM');
  });
});
