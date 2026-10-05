/**
 * A quest's goal and objectives, and door keys: what the rules engine checks
 * to end a quest online (docs/campaigns/three-plagues/ONLINE_RULES.md, "Quest
 * goals and objectives"). Pure edits of the quest document; empty values are
 * left out, as the Go side writes them.
 */

import type { Catalog, ObjectiveDoc, QuestDoc } from '../maps/types.ts';

/** Mirrors internal/maps MaxGoal and MaxKeyName. */
export const MAX_GOAL = 500;
export const MAX_KEY_NAME = 80;

/** Sets the goal the players see; an empty goal is removed. */
export function setGoal(q: QuestDoc, text: string): QuestDoc {
  const goal = text.trim().slice(0, MAX_GOAL);
  const next = { ...q };
  delete next.goal;
  return goal ? { ...next, goal } : next;
}

function withObjectives(q: QuestDoc, objectives: ObjectiveDoc[]): QuestDoc {
  const next = { ...q };
  delete next.objectives;
  return objectives.length ? { ...next, objectives } : next;
}

function cleanItem(item: string | undefined): string {
  return (item ?? '').trim().slice(0, MAX_KEY_NAME);
}

/** Adds an objective; a collect objective without an item is not added. */
export function addObjective(q: QuestDoc, o: ObjectiveDoc): QuestDoc {
  let next: ObjectiveDoc;
  switch (o.kind) {
    case 'kill':
      next = o.monsters?.length ? { kind: 'kill', monsters: [...o.monsters] } : { kind: 'kill' };
      break;
    case 'collect': {
      const item = cleanItem(o.item);
      if (!item) {
        return q;
      }
      next = { kind: 'collect', item };
      break;
    }
    case 'escape':
      next = { kind: 'escape' };
      break;
  }
  return withObjectives(q, [...(q.objectives ?? []), next]);
}

export function removeObjective(q: QuestDoc, index: number): QuestDoc {
  return withObjectives(q, (q.objectives ?? []).filter((_, i) => i !== index));
}

/** Names the monsters a kill objective needs dead; none means every monster. */
export function setKillTargets(q: QuestDoc, index: number, monsters: readonly string[]): QuestDoc {
  return withObjectives(q, (q.objectives ?? []).map((o, i) => (i !== index || o.kind !== 'kill' ? o : monsters.length ? { kind: 'kill', monsters: [...monsters] } : { kind: 'kill' })));
}

/** Renames a collect objective's item; an empty name keeps the old one. */
export function setCollectItem(q: QuestDoc, index: number, item: string): QuestDoc {
  const name = cleanItem(item);
  if (!name) {
    return q;
  }
  return withObjectives(q, (q.objectives ?? []).map((o, i) => (i === index && o.kind === 'collect' ? { kind: 'collect', item: name } : o)));
}

/**
 * Drops monsters no longer in the quest from kill objectives; a kill
 * objective left naming nobody is removed (it must not turn into "every
 * monster").
 */
export function pruneKillTargets(q: QuestDoc): QuestDoc {
  if (!q.objectives?.length) {
    return q;
  }
  const ids = new Set(q.monsters.map((m) => m.id));
  const kept: ObjectiveDoc[] = [];
  for (const o of q.objectives) {
    if (o.kind !== 'kill' || !o.monsters?.length) {
      kept.push(o);
      continue;
    }
    const monsters = o.monsters.filter((id) => ids.has(id));
    if (monsters.length) {
      kept.push({ kind: 'kill', monsters });
    }
  }
  return withObjectives(q, kept);
}

/** One line for the GM: "Kill every monster", "Carry the Soul Gem". */
export function objectiveText(o: ObjectiveDoc, q: QuestDoc, catalog: Catalog): string {
  switch (o.kind) {
    case 'kill': {
      if (!o.monsters?.length) {
        return 'Kill every monster';
      }
      const names = o.monsters.map((id) => {
        const m = q.monsters.find((x) => x.id === id);
        const name = m ? (catalog.monsters.find((d) => d.id === m.type)?.name ?? m.type) : 'missing monster';
        return `${name} (${id})`;
      });
      return `Kill ${names.join(', ')}`;
    }
    case 'collect':
      return `Carry the ${o.item ?? ''}`;
    case 'escape':
      return 'Leave by the exits';
  }
}

/** Names the item that unlocks a locked door; unlocked doors have no key, and an empty name removes it. */
export function setDoorKey(q: QuestDoc, doorId: string, key: string): QuestDoc {
  const name = cleanItem(key);
  return {
    ...q,
    doors: q.doors.map((d) => {
      if (d.id !== doorId) {
        return d;
      }
      const next = { ...d };
      delete next.key;
      return name && d.locked ? { ...next, key: name } : next;
    }),
  };
}
