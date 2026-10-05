import { describe, expect, test } from 'bun:test';
import { DOC_VERSION } from '../maps/types.ts';
import type { Action } from '../seat/types.ts';
import { gmConsole, monsterMoveAt, monsterMoveTiles, rulesLines } from './rules.ts';
import type { SessionState } from './types.ts';

/** A 4x2 corridor with Grom and Vex, and two orcs; the rules on, round 2. */
function state(): SessionState {
  return {
    version: 2,
    board: { version: DOC_VERSION, width: 4, height: 2, regions: [0, 0, 0, 0, 0, 0, 0, 0], rooms: [] },
    quest: { version: DOC_VERSION, boardChecksum: '', doors: [], blockedSquares: [], furniture: [], monsters: [], traps: [], notes: [], startTiles: [] },
    questName: 'The Trial',
    round: 2,
    heroes: [
      { id: 'hero-1', name: 'Grom', class: 'barbarian', x: 1, y: 1, placed: true, body: 8, maxBody: 8, mind: 2, maxMind: 2, status: 'active' },
      { id: 'hero-2', name: 'Vex', class: 'elf', x: 1, y: 2, placed: true, body: 6, maxBody: 6, mind: 4, maxMind: 4, status: 'active' },
      { id: 'hero-3', name: 'Ilsa', class: 'wizard', x: 0, y: 0, placed: false, body: 4, maxBody: 4, mind: 6, maxMind: 6, status: 'active' },
    ],
    monsters: [
      { id: 'monster-1', type: 'orc', name: 'Orc', x: 3, y: 1, body: 5, maxBody: 5, mind: 2, visibility: 'seen', alive: true },
      { id: 'monster-2', type: 'orc', name: 'Orc', x: 4, y: 1, body: 5, maxBody: 5, mind: 2, visibility: 'hidden', alive: true },
    ],
    doors: [],
    traps: [],
    consumedNotes: [],
    discovered: [],
    rules: { ruleset: 'three-plagues/1', phase: 'heroes' },
  };
}

const act = (label: string, type: string, payload: Record<string, unknown>): Action => ({ label, command: { type, payload } });

const monsterActions: Action[] = [
  act('Orc (monster-1): move to (2,1)', 'monster.move', { monster: 'monster-1', to: { x: 2, y: 1 } }),
  act('Orc (monster-1): move to (2,2)', 'monster.move', { monster: 'monster-1', to: { x: 2, y: 2 } }),
  act('Orc (monster-1): attack Grom', 'monster.attack', { monster: 'monster-1', target: 'hero-1' }),
  act('Orc (monster-2): move to (4,2)', 'monster.move', { monster: 'monster-2', to: { x: 4, y: 2 } }),
  act('End the round', 'phase.end', {}),
];

describe('rulesLines', () => {
  test("the heroes' phase: the turn under way and who is still to act", () => {
    const s = state();
    s.rules = { ruleset: 'three-plagues/1', phase: 'heroes', acted: ['hero-2'], turn: { heroId: 'hero-1', moveRoll: 7, moveLeft: 3 }, skipNext: ['hero-2'] };
    expect(rulesLines(s)).toEqual([
      "Round 2: the heroes' turns",
      "Grom's turn: rolled 7, 3 squares left · action ready",
      'Done: Vex',
      'Loses their next turn: Vex',
    ]);
    s.rules = { ruleset: 'three-plagues/1', phase: 'heroes', turn: { heroId: 'hero-2', moveDone: true, acted: true } };
    expect(rulesLines(s)).toEqual(["Round 2: the heroes' turns", "Vex's turn: movement over · action used", 'Still to act: Grom']);
    s.rules = { ruleset: 'three-plagues/1', phase: 'heroes', turn: { heroId: 'hero-1' } };
    expect(rulesLines(s)[1]).toBe("Grom's turn: no roll yet · action ready");
  });

  test('between turns, the monsters, and the end', () => {
    const s = state();
    expect(rulesLines(s)).toEqual(["Round 2: the heroes' turns", 'No turn under way', 'Still to act: Grom, Vex']);
    s.rules = { ruleset: 'three-plagues/1', phase: 'monsters', monstersMoved: ['monster-1'], monstersActed: ['monster-1'] };
    expect(rulesLines(s)).toEqual(["Round 2: the monsters' turn", 'Done: Orc (monster-1)']);
    s.rules = { ruleset: 'three-plagues/1', phase: 'over', outcome: 'won' };
    expect(rulesLines(s)).toEqual(['The quest is won']);
    delete s.rules;
    expect(rulesLines(s)).toEqual([]);
  });
});

describe('gmConsole', () => {
  test('sorts the actions into phase buttons and each monster with its moves and attacks', () => {
    const s = state();
    s.rules = { ruleset: 'three-plagues/1', phase: 'monsters' };
    const c = gmConsole(s, monsterActions);
    expect(c.phase.map((a) => a.label)).toEqual(['End the round']);
    expect(c.monsters.map((m) => [m.id, m.label, m.moves.size, m.attacks.map((a) => a.label)])).toEqual([
      ['monster-1', 'Orc (monster-1)', 2, ['Attack Grom']],
      ['monster-2', 'Orc (monster-2) · hidden', 1, []],
    ]);
  });

  test("the selected monster's moves are highlighted and clicked", () => {
    const s = state();
    s.rules = { ruleset: 'three-plagues/1', phase: 'monsters' };
    const c = gmConsole(s, monsterActions);
    expect(monsterMoveTiles(c, 'monster-1')).toEqual([{ x: 2, y: 1 }, { x: 2, y: 2 }]);
    expect(monsterMoveTiles(c, 'hero-1')).toEqual([]);
    expect(monsterMoveTiles(c, null)).toEqual([]);
    expect(monsterMoveAt(c, 'monster-1', { x: 2, y: 2 })?.command.payload).toEqual({ monster: 'monster-1', to: { x: 2, y: 2 } });
    expect(monsterMoveAt(c, 'monster-1', { x: 4, y: 2 })).toBeNull();
    expect(monsterMoveAt(c, null, { x: 2, y: 2 })).toBeNull();
  });
});
