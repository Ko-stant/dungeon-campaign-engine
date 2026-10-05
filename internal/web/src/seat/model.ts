/**
 * The player's game screen as data: which hero the player acts for, what
 * may be clicked on the board, the buttons, and where the game stands. Pure;
 * pages/seat.ts draws it.
 */

import type { TileCoord } from '../board/geometry.ts';
import type { PlayerState } from '../players/types.ts';
import type { Action, SeatHero, SeatState } from './types.ts';

/** The hero the screen acts for: the one whose turn it is, else the one the player picked, else the first. */
export function activeHero(seat: SeatState, picked: string | null): SeatHero | null {
  const byId = (id: string | null | undefined): SeatHero | undefined => seat.heroes.find((h) => h.id === id);
  return byId(seat.turnHero) ?? byId(picked) ?? seat.heroes[0] ?? null;
}

/** A hero's actions, sorted for the screen: moves by square and attacks by monster for board clicks, and every other action as a button (attacks too). */
export interface ActionGroups {
  moves: Map<string, Action>;
  attacks: Map<string, Action>;
  buttons: Action[];
}

export const tileKey = (t: TileCoord): string => `${String(t.x)},${String(t.y)}`;

export function groupActions(h: SeatHero): ActionGroups {
  const g: ActionGroups = { moves: new Map(), attacks: new Map(), buttons: [] };
  for (const a of h.actions) {
    const p = a.command.payload;
    if (a.command.type === 'turn.move') {
      const to = p.to as TileCoord | undefined;
      if (to) {
        g.moves.set(tileKey(to), a);
      }
      continue;
    }
    if (a.command.type === 'turn.attack' && typeof p.target === 'string') {
      g.attacks.set(p.target, a);
    }
    g.buttons.push(a);
  }
  return g;
}

/** What clicking a square does: move there if it is highlighted, or attack a monster in reach standing on it. */
export function clickAction(g: ActionGroups, pv: PlayerState, t: TileCoord): Action | null {
  const move = g.moves.get(tileKey(t));
  if (move) {
    return move;
  }
  const m = pv.monsters.find((m) => t.x >= m.x && t.x < m.x + Math.max(m.width, 1) && t.y >= m.y && t.y < m.y + Math.max(m.height, 1));
  return (m && g.attacks.get(m.id)) ?? null;
}

/** One line on where the game stands for this player. */
export function statusLine(seat: SeatState, pv: PlayerState): string {
  if (seat.phase === 'over') {
    return seat.outcome === 'won' ? 'The quest is won!' : 'The quest is lost.';
  }
  if (!seat.phase) {
    return 'Waiting for the GM to start the game';
  }
  const round = `Round ${String(seat.round)}`;
  if (seat.phase === 'monsters') {
    return `${round}: the monsters' turn`;
  }
  if (seat.turnHero) {
    const mine = seat.heroes.find((h) => h.id === seat.turnHero);
    if (mine) {
      return `${round}: your turn (${mine.name})`;
    }
    const name = pv.heroes.find((h) => h.id === seat.turnHero)?.name ?? 'Another hero';
    return `${round}: ${name}'s turn`;
  }
  const canStart = seat.heroes.some((h) => h.actions.some((a) => a.command.type === 'turn.start'));
  return `${round}: the heroes' turns. ${canStart ? "Start a turn when you're ready." : 'Waiting for the others.'}`;
}

/** What is left of a hero's turn, or "" when it is not their turn. */
export function turnDetail(h: SeatHero): string {
  const t = h.turn;
  if (!t) {
    return '';
  }
  let move: string;
  if (t.moveDone) {
    move = 'Movement over';
  } else if (t.moveRoll) {
    move = `Rolled ${String(t.moveRoll)}: ${String(t.moveLeft ?? 0)} square${t.moveLeft === 1 ? '' : 's'} left`;
  } else {
    move = 'Move (roll first)';
  }
  if (!t.acted && !t.moveRoll && !t.moveDone) {
    return 'Move (roll first) and act, in either order';
  }
  return `${move} · ${t.acted ? 'action used' : 'action ready'}`;
}
