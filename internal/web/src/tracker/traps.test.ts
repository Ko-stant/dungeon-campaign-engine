import { describe, expect, test } from 'bun:test';
import { triggerButton } from './traps.ts';

describe('triggerButton', () => {
  test('says what triggering does for each kind (mirrors trapTriggers in Go)', () => {
    expect(triggerButton('pit').label).toBe('Trigger (stays)');
    expect(triggerButton('long_pit').label).toBe('Trigger (stays)');
    expect(triggerButton('falling_block').label).toBe('Trigger (blocks the square)');
    expect(triggerButton('spear').label).toBe('Trigger (gone)');
    expect(triggerButton('boulder').label).toBe('Trigger (rolls)');
    expect(triggerButton('teleport').label).toBe('Trigger');
    expect(triggerButton('boulder').title).toContain('Turn into blocked squares');
  });
});
