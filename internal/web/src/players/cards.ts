/** What a click on the player screen picks, and the card it shows. */
import { footprintTiles, type TileCoord } from '../board/geometry.ts';
import { covers } from '../board/model.ts';
import { combatLine, monsterLine } from '../tracker/combat.ts';
import { effectLabel } from '../tracker/effects.ts';
import type { PlayerCatalog, PlayerState } from './types.ts';

export type PickKind = 'hero' | 'monster' | 'trap' | 'furniture' | 'block';

export interface Pick {
  kind: PickKind;
  id: string;
}

export interface Card {
  title: string;
  subtitle?: string;
  /** Short highlighted states, e.g. "Wounded", "Triggered". */
  tags: string[];
  lines: string[];
  effects: string[];
}

const on = (t: TileCoord) => (p: TileCoord): boolean => p.x === t.x && p.y === t.y;

/** The topmost piece on a square: heroes, monsters, traps, furniture, then blocked squares. */
export function pickAt(pv: PlayerState, catalog: PlayerCatalog, t: TileCoord): Pick | null {
  const hero = pv.heroes.find((h) => h.placed && h.x === t.x && h.y === t.y);
  if (hero) {
    return { kind: 'hero', id: hero.id };
  }
  const monster = pv.monsters.find((m) => covers(m.x, m.y, m.width, m.height, t));
  if (monster) {
    return { kind: 'monster', id: monster.id };
  }
  const trap = pv.traps.find((tr) => {
    const def = catalog.traps.find((d) => d.id === tr.kind);
    return footprintTiles({ x: tr.x, y: tr.y }, def?.width ?? 1, def?.height ?? 1, tr.rotation ?? 0).some(on(t));
  });
  if (trap) {
    return { kind: 'trap', id: trap.id };
  }
  const furniture = pv.furniture.find((f) => {
    const def = catalog.furniture.find((d) => d.id === f.type);
    return footprintTiles({ x: f.x, y: f.y }, def?.width ?? 1, def?.height ?? 1, f.rotation).some(on(t));
  });
  if (furniture) {
    return { kind: 'furniture', id: furniture.id };
  }
  const block = pv.blocks.find((b) => covers(b.x, b.y, b.w, b.h, t));
  return block ? { kind: 'block', id: block.id } : null;
}

const capitalized = (s: string): string => s.charAt(0).toUpperCase() + s.slice(1);

/** The card for a picked piece; null when it is no longer on the screen. */
export function cardFor(pv: PlayerState, catalog: PlayerCatalog, pick: Pick): Card | null {
  switch (pick.kind) {
    case 'monster': {
      const m = pv.monsters.find((x) => x.id === pick.id);
      if (!m) {
        return null;
      }
      const lines: string[] = [];
      if (m.body !== undefined && m.maxBody !== undefined) {
        lines.push(`Body ${String(m.body)} / ${String(m.maxBody)}`);
      }
      if (m.combat) {
        lines.push(monsterLine(m.combat));
      }
      const move = catalog.monsters.find((d) => d.id === m.type)?.movement ?? 0;
      if (move > 0) {
        lines.push(`Move ${String(move)}`);
      }
      return { title: m.name, tags: m.wounded ? ['Wounded'] : [], lines, effects: (m.effects ?? []).map(effectLabel) };
    }
    case 'hero': {
      const h = pv.heroes.find((x) => x.id === pick.id);
      if (!h) {
        return null;
      }
      const stats = [`Body ${String(h.body)} / ${String(h.maxBody)}`, `Mind ${String(h.mind)} / ${String(h.maxMind)}`];
      if ((h.manaCap ?? 0) > 0) {
        stats.push(`Mana ${String(h.mana ?? 0)} / ${String(h.manaCap ?? 0)}`);
      }
      const lines = [stats.join(' · ')];
      if (h.combat) {
        lines.push(combatLine(h.combat));
      }
      const card: Card = { title: h.name, tags: h.status === 'dead' ? ['Fallen'] : h.status === 'escaped' ? ['Escaped'] : [], lines, effects: (h.effects ?? []).map(effectLabel) };
      const cls = catalog.heroes.find((c) => c.id === h.class)?.name;
      if (cls) {
        card.subtitle = cls;
      }
      return card;
    }
    case 'trap': {
      const t = pv.traps.find((x) => x.id === pick.id);
      if (!t) {
        return null;
      }
      const name = catalog.traps.find((d) => d.id === t.kind)?.name ?? `${capitalized(t.kind.replaceAll('_', ' '))} trap`;
      return { title: name, tags: [capitalized(t.state)], lines: [], effects: [] };
    }
    case 'furniture': {
      const f = pv.furniture.find((x) => x.id === pick.id);
      if (!f) {
        return null;
      }
      return { title: catalog.furniture.find((d) => d.id === f.type)?.name ?? capitalized(f.type.replaceAll('_', ' ')), tags: [], lines: [], effects: [] };
    }
    case 'block':
      return pv.blocks.some((b) => b.id === pick.id) ? { title: 'Blocked squares', tags: [], lines: [], effects: [] } : null;
  }
}
