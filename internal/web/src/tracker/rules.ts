/**
 * The GM's rules console as data (online play, docs/ONLINE_AND_RULES_PLAN.md
 * Phase 5c): where the round stands, and what the GM may do by the rules,
 * from the actions the server lists (GET /api/sessions/{id}/actions; the
 * rules live only in Go). Pure; pages/tracker.ts draws it.
 */
import type { TileCoord } from '../board/geometry.ts';
import { tileKey } from '../seat/model.ts';
import type { Action } from '../seat/types.ts';
import type { SessionState } from './types.ts';

/** One monster's moves (by square) and attacks in the monsters' phase. */
export interface MonsterPlan {
  id: string;
  /** Name and id, as the GM's lists show monsters. */
  label: string;
  moves: Map<string, Action>;
  /** Labeled "Attack <hero>". */
  attacks: Action[];
}

export interface GmConsole {
  /** Phase changes: the monsters' turn, ending the round. */
  phase: Action[];
  /** Monsters that can still move or attack, in the session's order. */
  monsters: MonsterPlan[];
}

const monsterLabel = (s: SessionState, id: string): string => {
  const m = s.monsters.find((x) => x.id === id);
  return m ? `${m.name} (${m.id})` : id;
};

const heroName = (s: SessionState, id: string): string => s.heroes.find((h) => h.id === id)?.name ?? id;

/** Sorts the GM's legal actions for the console. */
export function gmConsole(s: SessionState, actions: Action[]): GmConsole {
  const phase: Action[] = [];
  const plans = new Map<string, MonsterPlan>();
  const plan = (id: string): MonsterPlan => {
    let p = plans.get(id);
    if (!p) {
      const hidden = s.monsters.find((m) => m.id === id)?.visibility === 'hidden';
      p = { id, label: `${monsterLabel(s, id)}${hidden ? ' · hidden' : ''}`, moves: new Map(), attacks: [] };
      plans.set(id, p);
    }
    return p;
  };
  for (const a of actions) {
    const p = a.command.payload;
    const monster = typeof p.monster === 'string' ? p.monster : null;
    if (a.command.type === 'monster.move' && monster) {
      const to = p.to as TileCoord | undefined;
      if (to) {
        plan(monster).moves.set(tileKey(to), a);
      }
    } else if (a.command.type === 'monster.attack' && monster && typeof p.target === 'string') {
      plan(monster).attacks.push({ ...a, label: `Attack ${heroName(s, p.target)}` });
    } else {
      phase.push(a);
    }
  }
  const order = s.monsters.map((m) => m.id);
  const monsters = [...plans.values()].sort((a, b) => order.indexOf(a.id) - order.indexOf(b.id));
  return { phase, monsters };
}

/** The squares the selected monster may move to (none unless a monster with moves is selected). */
export function monsterMoveTiles(c: GmConsole, selectedId: string | null): TileCoord[] {
  const p = c.monsters.find((m) => m.id === selectedId);
  return p ? [...p.moves.values()].map((a) => a.command.payload.to as TileCoord) : [];
}

/** The selected monster's move to a clicked square, if it is one of its moves. */
export function monsterMoveAt(c: GmConsole, selectedId: string | null, t: TileCoord): Action | null {
  return c.monsters.find((m) => m.id === selectedId)?.moves.get(tileKey(t)) ?? null;
}

/** Where the round stands, a line each; empty while the rules are off. */
export function rulesLines(s: SessionState): string[] {
  const r = s.rules;
  if (!r) {
    return [];
  }
  if (r.phase === 'over') {
    return [`The quest is ${r.outcome ?? 'over'}`];
  }
  const names = (ids: string[]): string => ids.map((id) => heroName(s, id)).join(', ');
  if (r.phase === 'monsters') {
    const done = s.monsters.filter((m) => m.alive && (r.monstersMoved ?? []).includes(m.id) && (r.monstersActed ?? []).includes(m.id));
    return [`Round ${String(s.round)}: the monsters' turn`, ...(done.length ? [`Done: ${done.map((m) => `${m.name} (${m.id})`).join(', ')}`] : [])];
  }
  const lines = [`Round ${String(s.round)}: the heroes' turns`];
  const t = r.turn;
  if (t) {
    const move = t.moveDone ? 'movement over' : t.moveRoll ? `rolled ${String(t.moveRoll)}, ${String(t.moveLeft ?? 0)} ${t.moveLeft === 1 ? 'square' : 'squares'} left` : 'no roll yet';
    lines.push(`${heroName(s, t.heroId)}'s turn: ${move} · ${t.acted ? 'action used' : 'action ready'}`);
  } else {
    lines.push('No turn under way');
  }
  const acted = r.acted ?? [];
  const waiting = s.heroes.filter((h) => h.status === 'active' && h.placed && !acted.includes(h.id) && h.id !== t?.heroId);
  if (waiting.length) {
    lines.push(`Still to act: ${names(waiting.map((h) => h.id))}`);
  }
  if (acted.length) {
    lines.push(`Done: ${names(acted)}`);
  }
  if (r.skipNext?.length) {
    lines.push(`Loses their next turn: ${names(r.skipNext)}`);
  }
  return lines;
}
