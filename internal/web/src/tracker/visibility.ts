/** What the players see on the player screen (see internal/tracker/visibility.go). */
import type { SessionState } from './types.ts';

/**
 * Whether a piece is shown on the player screen: monsters, quest furniture,
 * quest blocked squares and doors as the GM marked them; blocked squares
 * added during play always. Null for anything that can't be shown this way
 * (traps show themselves once revealed; notes never).
 */
export function shownToPlayers(s: SessionState, id: string): boolean | null {
  const monster = s.monsters.find((m) => m.id === id);
  if (monster) {
    return monster.visibility === 'seen';
  }
  if (s.quest.furniture.some((f) => f.id === id)) {
    return (s.seenFurniture ?? []).includes(id);
  }
  if (s.quest.blockedSquares.some((b) => b.id === id)) {
    return (s.seenBlocks ?? []).includes(id);
  }
  if ((s.addedBlocks ?? []).some((b) => b.id === id)) {
    return true;
  }
  const door = s.doors.find((d) => d.id === id);
  if (door) {
    return door.seen ?? false;
  }
  return null;
}
