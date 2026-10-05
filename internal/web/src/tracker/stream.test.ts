import { describe, expect, test } from 'bun:test';
import { parseStreamMessage } from './stream.ts';

describe('parseStreamMessage', () => {
  test('a change carries the state, event and sequence', () => {
    const msg = parseStreamMessage(JSON.stringify({ state: { round: 2 }, event: { seq: 4 }, eventSeq: 4 }));
    expect(msg?.kind).toBe('change');
  });

  test('presence lists who is connected', () => {
    const msg = parseStreamMessage(JSON.stringify({ presence: [{ name: 'Sam', heroes: ['Vex'] }] }));
    expect(msg).toEqual({ kind: 'presence', presence: [{ name: 'Sam', heroes: ['Vex'] }] });
    expect(parseStreamMessage(JSON.stringify({ presence: [] }))).toEqual({ kind: 'presence', presence: [] });
  });

  test('anything else is ignored', () => {
    expect(parseStreamMessage('not json')).toBeNull();
    expect(parseStreamMessage(JSON.stringify({ hello: 1 }))).toBeNull();
    expect(parseStreamMessage('42')).toBeNull();
  });
});
