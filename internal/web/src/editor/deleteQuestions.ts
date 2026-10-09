/** The questions the map editor's Yes/No dialog asks before a delete. */

/** "A", "A and B" or "A, B and C". */
function joinNames(names: readonly string[]): string {
  if (names.length <= 1) {
    return names[0] ?? '';
  }
  return `${names.slice(0, -1).join(', ')} and ${names[names.length - 1] ?? ''}`;
}

export function questDeleteQuestion(name: string): string {
  const quest = name.trim() ? `the quest ${name.trim()}` : 'this quest';
  return `Delete ${quest} for good? It is removed from any campaign's chapters. Sessions already started keep their copy.`;
}

export function boardDeleteQuestion(name: string, quests: readonly string[]): string {
  let q = `Delete the board ${name} for good?`;
  if (quests.length === 1) {
    q += ` Its quest (${joinNames(quests)}) goes with it, and is removed from any campaign's chapters.`;
  } else if (quests.length > 1) {
    q += ` Its ${String(quests.length)} quests (${joinNames(quests)}) go with it, and are removed from any campaign's chapters.`;
  }
  return `${q} Sessions already started keep their copy of the map.`;
}
