import { describe, expect, test } from 'bun:test';
import type { PlayerState } from '../players/types.ts';
import { activeHero, clickAction, groupActions, seatFromUpdate, statusLine, turnDetail } from './model.ts';
import type { Action, SeatHero, SeatState, SeatUpdate } from './types.ts';

const act = (label: string, type: string, payload: Record<string, unknown> = {}): Action => ({ label, command: { type, payload } });

function hero(over: Partial<SeatHero> = {}): SeatHero {
  return { id: 'hero-1', name: 'Grom', className: 'Barbarian', placed: true, x: 1, y: 1, status: 'active', body: 40, maxBody: 40, mind: 3, maxMind: 3, items: [], actions: [], ...over };
}

const pv = {
  monsters: [{ id: 'orc', type: 'orc', name: 'Orc', x: 3, y: 3, width: 1, height: 1 }, { id: 'ogre', type: 'ogre', name: 'Ogre', x: 5, y: 1, width: 2, height: 2 }],
  heroes: [{ id: 'hero-1', name: 'Grom' }, { id: 'hero-2', name: 'Ilsa' }, { id: 'hero-3', name: 'Vex' }],
} as unknown as PlayerState;

describe('activeHero', () => {
  test('is the one whose turn it is, then the one picked, then the first', () => {
    const seat: SeatState = { round: 1, phase: 'heroes', heroes: [hero(), hero({ id: 'hero-3', name: 'Vex' })] };
    expect(activeHero(seat, null)?.id).toBe('hero-1');
    expect(activeHero(seat, 'hero-3')?.id).toBe('hero-3');
    expect(activeHero({ ...seat, turnHero: 'hero-3' }, 'hero-1')?.id).toBe('hero-3');
    expect(activeHero({ ...seat, turnHero: 'hero-2' }, 'hero-1')?.id).toBe('hero-1');
    expect(activeHero({ round: 1, heroes: [] }, null)).toBeNull();
  });
});

describe('groupActions', () => {
  test('sorts what a hero may do: moves go on the board, the rest are buttons', () => {
    const g = groupActions(hero({
      actions: [
        act('Roll for movement', 'turn.roll-move'),
        act('Move to (2,1)', 'turn.move', { to: { x: 2, y: 1 } }),
        act('Move to (1,2)', 'turn.move', { to: { x: 1, y: 2 } }),
        act('Open d1', 'turn.door', { door: 'd1' }),
        act('Attack Orc (orc)', 'turn.attack', { target: 'orc' }),
        act('Search for traps', 'turn.search', { kind: 'traps' }),
        act('Disarm Pit Trap pit', 'turn.disarm', { trap: 'pit' }),
        act('End the turn', 'turn.end'),
      ],
    }));
    expect(g.moves.size).toBe(2);
    expect(g.moves.get('2,1')?.label).toBe('Move to (2,1)');
    expect(g.attacks.get('orc')?.label).toBe('Attack Orc (orc)');
    expect(g.buttons.map((a) => a.label)).toEqual(['Roll for movement', 'Open d1', 'Attack Orc (orc)', 'Search for traps', 'Disarm Pit Trap pit', 'End the turn']);
  });
});

describe('clickAction', () => {
  const g = groupActions(hero({ actions: [act('Move to (2,1)', 'turn.move', { to: { x: 2, y: 1 } }), act('Attack Ogre (ogre)', 'turn.attack', { target: 'ogre' })] }));
  test('a highlighted square moves there; a monster in reach is attacked anywhere on it', () => {
    expect(clickAction(g, pv, { x: 2, y: 1 })?.command.type).toBe('turn.move');
    expect(clickAction(g, pv, { x: 6, y: 2 })?.command.type).toBe('turn.attack');
    expect(clickAction(g, pv, { x: 3, y: 3 })).toBeNull(); // the orc, out of reach
    expect(clickAction(g, pv, { x: 9, y: 9 })).toBeNull();
  });
});

describe('statusLine', () => {
  const mine = [hero()];
  test('tells the player where the game stands', () => {
    expect(statusLine({ round: 1, heroes: mine }, pv)).toBe('Waiting for the GM to start the game');
    expect(statusLine({ round: 2, phase: 'heroes', heroes: [hero({ actions: [act("Start Grom's turn", 'turn.start')] })] }, pv)).toBe("Round 2: the heroes' turns. Start a turn when you're ready.");
    expect(statusLine({ round: 2, phase: 'heroes', heroes: [hero({ acted: true })] }, pv)).toBe("Round 2: the heroes' turns. Waiting for the others.");
    expect(statusLine({ round: 2, phase: 'heroes', turnHero: 'hero-1', heroes: mine }, pv)).toBe('Round 2: your turn (Grom)');
    expect(statusLine({ round: 2, phase: 'heroes', turnHero: 'hero-2', heroes: mine }, pv)).toBe("Round 2: Ilsa's turn");
    expect(statusLine({ round: 2, phase: 'monsters', heroes: mine }, pv)).toBe("Round 2: the monsters' turn");
    expect(statusLine({ round: 5, phase: 'over', outcome: 'won', heroes: mine }, pv)).toBe('The quest is won!');
    expect(statusLine({ round: 5, phase: 'over', outcome: 'lost', heroes: mine }, pv)).toBe('The quest is lost.');
  });
});

describe('turnDetail', () => {
  test('says what is left of the turn', () => {
    expect(turnDetail(hero())).toBe('');
    expect(turnDetail(hero({ turn: { heroId: 'hero-1' } }))).toBe('Move (roll first) and act, in either order');
    expect(turnDetail(hero({ turn: { heroId: 'hero-1', moveRoll: 7, moveLeft: 4 } }))).toBe('Rolled 7: 4 squares left · action ready');
    expect(turnDetail(hero({ turn: { heroId: 'hero-1', moveRoll: 7, moveLeft: 1, acted: true, moveDone: true } }))).toBe('Movement over · action used');
    expect(turnDetail(hero({ turn: { heroId: 'hero-1', acted: true } }))).toBe('Move (roll first) · action used');
  });
});

describe('seatFromUpdate', () => {
  const seat = (names: string[]): SeatState => ({ round: 1, heroes: names.map((name, i) => hero({ id: `hero-${String(i)}`, name })) });
  const update = (eventSeq: number, names: string[]): SeatUpdate => ({ player: { state: pv, event: null, eventSeq, feed: null }, seat: seat(names) });

  test('a new event brings the new seat', () => {
    expect(seatFromUpdate(update(6, ['Grom']), 5)?.heroes.map((h) => h.name)).toEqual(['Grom']);
  });

  test('a seat sent again for the same event is taken: the GM handed this player a hero', () => {
    expect(seatFromUpdate(update(5, ['Grom', 'Ilsa']), 5)?.heroes.map((h) => h.name)).toEqual(['Grom', 'Ilsa']);
  });

  test('a seat from an older event, or no seat at all, changes nothing', () => {
    expect(seatFromUpdate(update(4, ['Grom']), 5)).toBeNull();
    expect(seatFromUpdate({ presence: [] }, 5)).toBeNull();
  });
});
