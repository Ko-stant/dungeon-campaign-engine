import { describe, expect, test } from 'bun:test';
import { calmFrom, calmFromStorage, fightHint, type FightView } from './fightHint.ts';

const goblin = (id: string, more: Partial<FightView['monsters'][number]> = {}) =>
  ({ id, name: 'Goblin', body: 3, maxBody: 3, alive: true, visibility: 'seen' as const, ...more });
const hero = (id: string, determination = 0) => ({ id, name: id === 'h1' ? 'Mordecai' : 'Papi', determination });
const view = (more: Partial<FightView> = {}): FightView => ({ fight: false, monsters: [], heroes: [hero('h1'), hero('h2')], ...more });

describe('fightHint: start a fight', () => {
  test('a monster the players can see, out of a fight, suggests starting one', () => {
    expect(fightHint(view({ monsters: [goblin('m1')] }), null)).toEqual({ kind: 'start', reason: 'Goblin in sight' });
    expect(fightHint(view({ monsters: [goblin('m1'), goblin('m2', { name: 'Orc' })] }), null)?.reason).toBe('2 monsters in sight');
  });

  test('hidden or dead monsters do not', () => {
    expect(fightHint(view({ monsters: [goblin('m1', { visibility: 'hidden' }), goblin('m2', { alive: false })] }), null)).toBeNull();
  });

  test('monsters still in sight when the fight ended (or the GM said "not a fight") are calm', () => {
    const s = view({ monsters: [goblin('m1')] });
    expect(fightHint(s, calmFrom(s))).toBeNull();
    // A new one comes into sight.
    expect(fightHint(view({ monsters: [goblin('m1'), goblin('m2', { name: 'Orc' })] }), calmFrom(s))).toEqual({ kind: 'start', reason: 'Orc in sight' });
  });

  test('a monster losing Body, seen or not, suggests a fight even if it was calm', () => {
    const s = view({ monsters: [goblin('m1'), goblin('m2', { visibility: 'hidden' })] });
    const calm = calmFrom(s);
    expect(fightHint(view({ monsters: [goblin('m1', { body: 1 }), goblin('m2', { visibility: 'hidden' })] }), calm)).toEqual({ kind: 'start', reason: 'Goblin took damage' });
    expect(fightHint(view({ monsters: [goblin('m1'), goblin('m2', { visibility: 'hidden', body: 2 })] }), calm)?.reason).toBe('Goblin took damage');
  });

  test('a hero missing an attack (Determination going up) suggests a fight', () => {
    expect(fightHint(view({ heroes: [hero('h1', 2), hero('h2')] }), null)).toEqual({ kind: 'start', reason: 'Mordecai missed an attack' });
    const s = view({ heroes: [hero('h1', 2)] });
    expect(fightHint(s, calmFrom(s))).toBeNull();
  });
});

describe('fightHint: end the fight', () => {
  test('in a fight with no living monster in sight, suggests ending it', () => {
    expect(fightHint(view({ fight: true, monsters: [goblin('m1', { alive: false }), goblin('m2', { visibility: 'hidden' })] }), null))
      .toEqual({ kind: 'end', reason: 'No monsters left in sight' });
  });

  test('not while one is still in sight', () => {
    expect(fightHint(view({ fight: true, monsters: [goblin('m1')] }), null)).toBeNull();
  });
});

describe('calmFromStorage', () => {
  test('reads what calmFrom stored, and nothing else', () => {
    const calm = calmFrom(view({ monsters: [goblin('m1', { body: 2 })], heroes: [hero('h1', 2)] }));
    expect(calmFromStorage(JSON.stringify(calm))).toEqual(calm);
    expect(calmFromStorage(null)).toBeNull();
    expect(calmFromStorage('nope')).toBeNull();
    expect(calmFromStorage('{"monsters":{"m1":"x"},"determination":{}}')).toBeNull();
  });
});
