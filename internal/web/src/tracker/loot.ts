/**
 * The campaign's loot list in play: items with their kind and stats ready
 * (GET /api/campaigns/{id}/loot), handed to a hero in one step, and potions
 * that heal or restore mana when used (item.use).
 */
import { itemStatsLine } from './combat.ts';
import type { Command, Item, ItemStats } from './types.ts';

const STAT_KEYS: (keyof ItemStats)[] = ['damage', 'accuracy', 'avoidance', 'mitigation', 'mana', 'manaRegen'];

/** item.add for one of a loot item, unequipped, with its kind, stats, use and notes. */
export function lootAddCommand(heroId: string, loot: Item): Command {
  const payload: Record<string, unknown> = { heroId, name: loot.name, quantity: 1 };
  if (loot.kind) {
    payload.kind = loot.kind;
  }
  if (loot.notes) {
    payload.notes = loot.notes;
  }
  for (const k of STAT_KEYS) {
    if (loot[k]) {
      payload[k] = loot[k];
    }
  }
  if (loot.healBody) {
    payload.healBody = loot.healBody;
  }
  if (loot.restoreMana) {
    payload.restoreMana = loot.restoreMana;
  }
  return { type: 'item.add', payload };
}

/** What using an item does, e.g. "heals 8 Body" ("" for nothing). */
export function itemUseText(it: Pick<Item, 'healBody' | 'restoreMana'>): string {
  const parts: string[] = [];
  if ((it.healBody ?? 0) > 0) {
    parts.push(`heals ${String(it.healBody)} Body`);
  }
  if ((it.restoreMana ?? 0) > 0) {
    parts.push(`restores ${String(it.restoreMana)} mana`);
  }
  return parts.join(', ');
}

/** An item's stats, then what using it does. Mirrors Item.Summary (Go). */
export function itemSummary(it: ItemStats & Pick<Item, 'healBody' | 'restoreMana'>): string {
  return [itemStatsLine(it), itemUseText(it)].filter(Boolean).join(', ');
}

/** An item does something when used (item.use). */
export function usable(it: Pick<Item, 'healBody' | 'restoreMana'>): boolean {
  return (it.healBody ?? 0) > 0 || (it.restoreMana ?? 0) > 0;
}
