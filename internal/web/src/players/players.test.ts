import { describe, expect, test } from 'bun:test';
import { cardFor, pickAt } from './cards.ts';
import { addEvent } from './feed.ts';
import type { PlayerCatalog, PlayerEvent, PlayerState } from './types.ts';
import { playerBoardView } from './view.ts';

const catalog: PlayerCatalog = {
  furniture: [{ id: 'table', name: 'Table', width: 2, height: 1, blocksMovement: true, blocksLineOfSight: false, image: 'assets/table.png' }],
  traps: [{ id: 'pit', name: 'Pit Trap', width: 1, height: 1, image: 'assets/pit.png' }],
  monsters: [{ id: 'orc', name: 'Orc', movement: 8, image: 'assets/orc.png' }],
  heroes: [{ id: 'custom-rogue', name: 'Rogue', color: '#4b5563' }],
};

/** A 4x2 corridor: a table on (1,1)-(2,1), a pit on (3,2), an orc on (4,1), blocked squares on (4,2), the Rogue on (1,2). */
function view(): PlayerState {
  return {
    questName: 'Crumbling Halls',
    round: 3,
    fight: true,
    width: 4,
    height: 2,
    regions: [0, 0, 0, 0, 0, 0, 0, 0],
    discovered: [0, 1, 4],
    doors: [{ id: 'door-1', edge: { x: 3, y: 1, orientation: 'vertical' }, kind: 'normal', state: 'open' }],
    furniture: [{ id: 'furniture-1', type: 'table', x: 1, y: 1, rotation: 0 }],
    blocks: [{ id: 'added-1', x: 4, y: 2, w: 1, h: 1 }],
    traps: [{ id: 'trap-1', kind: 'pit', x: 3, y: 2, state: 'triggered' }],
    monsters: [{
      id: 'monster-1', type: 'orc', name: 'Orc', x: 4, y: 1, width: 1, height: 1, body: 4, maxBody: 22, wounded: true,
      combat: { avoidance: 8, hitDice: '2d8', damage: 9 }, effects: [{ id: 'effect-1', name: 'Poisoned', rounds: 2 }],
    }],
    heroes: [{
      id: 'hero-1', name: 'Papi Ponzi', class: 'custom-rogue', x: 1, y: 2, placed: true, body: 20, maxBody: 28, mind: 3, maxMind: 3, status: 'active',
      combat: { hitDice: '2d10', accuracy: 2, critFrom: 15, damage: 6, defenseDice: '1d6', avoidance: 7, mitigation: 0 },
    }],
    showMonsterBody: true,
  };
}

describe('playerBoardView', () => {
  test('draws what the players know: seen squares tinted, pieces with their artwork, no GM marks', () => {
    const v = playerBoardView(view(), catalog);
    expect(v.cols).toBe(4);
    expect([...(v.seenTiles ?? [])]).toEqual([0, 1, 4]);
    expect(v.discovered).toBeUndefined();
    expect(v.doors).toEqual([{ id: 'door-1', edge: { x: 3, y: 1, orientation: 'vertical' }, kind: 'normal', state: 'open', locked: false }]);
    expect(v.furniture).toEqual([{ id: 'furniture-1', type: 'table', at: { x: 1, y: 1 }, width: 2, height: 1, rotation: 0, image: 'assets/table.png' }]);
    expect(v.traps).toEqual([{ id: 'trap-1', kind: 'pit', at: { x: 3, y: 2 }, state: 'triggered', width: 1, height: 1, rotation: 0, image: 'assets/pit.png' }]);
    expect(v.monsters).toEqual([{ id: 'monster-1', type: 'orc', at: { x: 4, y: 1 }, label: 'Orc', image: 'assets/orc.png' }]);
    expect(v.heroes).toEqual([{ id: 'hero-1', type: 'custom-rogue', at: { x: 1, y: 2 }, label: 'Papi Ponzi' }]);
    expect(v.blockedSquares).toEqual([{ id: 'added-1', x: 4, y: 2, w: 1, h: 1 }]);
    expect(v.notes).toBeUndefined();
  });
});

describe('pickAt', () => {
  test('heroes, then monsters, traps, furniture and blocked squares', () => {
    const pv = view();
    expect(pickAt(pv, catalog, { x: 1, y: 2 })).toEqual({ kind: 'hero', id: 'hero-1' });
    expect(pickAt(pv, catalog, { x: 4, y: 1 })).toEqual({ kind: 'monster', id: 'monster-1' });
    expect(pickAt(pv, catalog, { x: 3, y: 2 })).toEqual({ kind: 'trap', id: 'trap-1' });
    expect(pickAt(pv, catalog, { x: 2, y: 1 })).toEqual({ kind: 'furniture', id: 'furniture-1' });
    expect(pickAt(pv, catalog, { x: 4, y: 2 })).toEqual({ kind: 'block', id: 'added-1' });
    expect(pickAt(pv, catalog, { x: 3, y: 1 })).toBeNull();
  });
});

describe('cardFor', () => {
  test('a monster: Body, Wounded, stats, Move and effects', () => {
    expect(cardFor(view(), catalog, { kind: 'monster', id: 'monster-1' })).toEqual({
      title: 'Orc',
      tags: ['Wounded'],
      lines: ['Body 4 / 22', 'Avoid 8 · Hit 2d8 · Damage 9', 'Move 8'],
      effects: ['Poisoned (2 rounds)'],
    });
  });

  test("a monster's Body stays off the card while the GM hides it", () => {
    const pv = view();
    pv.showMonsterBody = false;
    const m = pv.monsters[0];
    if (m) {
      delete m.body;
      delete m.maxBody;
    }
    expect(cardFor(pv, catalog, { kind: 'monster', id: 'monster-1' })?.lines).toEqual(['Avoid 8 · Hit 2d8 · Damage 9', 'Move 8']);
  });

  test('a hero: class, stats and totals', () => {
    expect(cardFor(view(), catalog, { kind: 'hero', id: 'hero-1' })).toEqual({
      title: 'Papi Ponzi',
      subtitle: 'Rogue',
      tags: [],
      lines: ['Body 20 / 28 · Mind 3 / 3', 'Hit 2d10+2 · Crit 15-20 · Damage 6 · Avoid 7+1d6'],
      effects: [],
    });
  });

  test('furniture, traps and blocked squares get a name', () => {
    expect(cardFor(view(), catalog, { kind: 'furniture', id: 'furniture-1' })?.title).toBe('Table');
    expect(cardFor(view(), catalog, { kind: 'trap', id: 'trap-1' })).toMatchObject({ title: 'Pit Trap', tags: ['Triggered'] });
    expect(cardFor(view(), catalog, { kind: 'block', id: 'added-1' })?.title).toBe('Blocked squares');
  });

  test('a piece that is gone has no card', () => {
    expect(cardFor(view(), catalog, { kind: 'monster', id: 'monster-9' })).toBeNull();
  });
});

describe('addEvent', () => {
  const ev = (seq: number): PlayerEvent => ({ seq, round: 1, summary: `line ${String(seq)}`, createdAt: '' });
  test('keeps the newest lines, without repeats', () => {
    let feed = [ev(1), ev(2)];
    feed = addEvent(feed, ev(3), 3);
    feed = addEvent(feed, ev(3), 3);
    feed = addEvent(feed, ev(4), 3);
    expect(feed.map((e) => e.seq)).toEqual([2, 3, 4]);
  });
});
