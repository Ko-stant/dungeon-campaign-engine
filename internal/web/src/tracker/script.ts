/** The Read aloud panel's logic: which script section fits the current map, and stepping through passages. */
import type { Chapter, ScriptSection } from './types.ts';

/**
 * The index of the script section for the active quest: a "Quest N ..." section
 * where N is the quest's chapter number, else a section whose title names the
 * quest. -1 when nothing matches.
 */
export function currentSection(sections: readonly ScriptSection[], state: { questId?: string; questName: string }, chapters: readonly Chapter[]): number {
  const chapter = chapters.find((c) => c.questId === state.questId);
  if (chapter) {
    const i = sections.findIndex((s) => Number(/^quest\s+(\d+)\b/i.exec(s.title)?.[1]) === chapter.number);
    if (i >= 0) {
      return i;
    }
  }
  const name = state.questName.trim().toLowerCase();
  return name ? sections.findIndex((s) => s.title.toLowerCase().includes(name)) : -1;
}

/** The passages before and after id, in script order across sections. */
export function passageNeighbors(sections: readonly ScriptSection[], id: string): { prev: string | null; next: string | null } {
  const ids = sections.flatMap((s) => s.passages.map((p) => p.id));
  const i = ids.indexOf(id);
  if (i < 0) {
    return { prev: null, next: null };
  }
  return { prev: ids[i - 1] ?? null, next: ids[i + 1] ?? null };
}

/** "2/5 read" for a section. */
export function sectionProgress(section: ScriptSection, read: readonly string[] | null | undefined): string {
  const done = section.passages.filter((p) => read?.includes(p.id)).length;
  return `${String(done)}/${String(section.passages.length)} read`;
}

export interface Clip {
  id: string;
  /** "Clip" for the passage's own clip, else its letter ("a"). */
  label: string;
  url: string;
}

/** A passage's audio clips: the clip named after it, then its lettered clips ("Q3-09a") in order. */
export function passageClips(passageId: string, clips: Readonly<Record<string, string>>): Clip[] {
  const out: Clip[] = [];
  const own = clips[passageId];
  if (own) {
    out.push({ id: passageId, label: 'Clip', url: own });
  }
  for (const [id, url] of Object.entries(clips).sort(([a], [b]) => a.localeCompare(b))) {
    const rest = id.slice(passageId.length);
    if (id.startsWith(passageId) && /^[a-z]$/.test(rest)) {
      out.push({ id, label: rest, url });
    }
  }
  return out;
}
