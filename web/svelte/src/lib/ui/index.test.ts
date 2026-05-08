import { describe, it, expect } from 'vitest';
import * as ui from './index';

const expected = [
  'Button',
  'Input',
  'Textarea',
  'Select',
  'Checkbox',
  'FormField',
  'Card',
  'Tabs',
  'Sheet',
  'Modal',
  'Skeleton',
  'EmptyState',
  'Tooltip',
  'StatusBadge',
  'StatusDot',
  'Pill',
  'StatBlock',
  'Sparkline',
  'ChartFrame',
  'CommandPalette',
  'Toast',
  'chartTheme',
];

describe('lib/ui barrel', () => {
  it.each(expected)('exports %s', (name) => {
    expect(ui).toHaveProperty(name);
  });

  it('exports exactly the expected surface (no extras, no gaps)', () => {
    const actual = Object.keys(ui).sort();
    expect(actual).toEqual([...expected].sort());
  });
});
