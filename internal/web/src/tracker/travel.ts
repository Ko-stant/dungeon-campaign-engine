/** Choices for travelling the party to another map mid-game. */
import type { Chapter, OtherMap } from './types.ts';

export interface TravelOption {
  questId: string;
  label: string;
}

/**
 * The maps the party can travel to: the campaign's other chapters in order,
 * then any other map the session has visited. Maps already visited are marked
 * "(return)" because travelling there restores them as they were left.
 */
export function travelOptions(state: { questId?: string; otherMaps?: readonly OtherMap[] }, chapters: readonly Chapter[]): TravelOption[] {
  const visited = new Set((state.otherMaps ?? []).map((m) => m.questId));
  const out: TravelOption[] = [];
  for (const ch of chapters) {
    if (ch.questId === state.questId) {
      continue;
    }
    out.push({ questId: ch.questId, label: `Chapter ${ch.number}: ${ch.questName}${visited.has(ch.questId) ? ' (return)' : ''}` });
  }
  for (const m of state.otherMaps ?? []) {
    if (!chapters.some((ch) => ch.questId === m.questId)) {
      out.push({ questId: m.questId, label: `${m.questName} (return)` });
    }
  }
  return out;
}
