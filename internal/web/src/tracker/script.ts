/** The Read aloud panel's logic: which script section fits the current map, and stepping through passages. */
import type { NoteDoc } from '../maps/types.ts';
import type { Chapter, ScriptPassage, ScriptSection } from './types.ts';

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

/** A note matches when it holds one run of this many words from the start of the passage (fewer could match by chance). */
const RUN_WORDS = 8;
/** How many runs from the start of the passage are tried, so one differing word (a spelling the note has differently) does not stop a match. */
const RUNS = 4;

/** Lower case, apostrophes dropped, everything but letters and digits as single spaces. */
function normalize(text: string): string {
  return ` ${text.toLowerCase().replace(/['’]/g, '').replace(/[^\p{L}\p{N}]+/gu, ' ').trim()} `;
}

/**
 * The quest notes that hold a passage: a note whose text starts with the
 * passage id ("Q1-N1 ..."), or one that contains a run of eight words from
 * the start of the passage's spoken text (the GM often pastes the passage
 * into the note, sometimes with a word changed).
 */
export function passageNotes(passage: ScriptPassage, notes: readonly NoteDoc[]): NoteDoc[] {
  const words = normalize(passage.parts.flatMap((p) => p.paragraphs).join(' ')).trim().split(' ');
  const runs: string[] = [];
  for (let i = 0; i + RUN_WORDS <= words.length && runs.length < RUNS; i += RUN_WORDS) {
    runs.push(` ${words.slice(i, i + RUN_WORDS).join(' ')} `);
  }
  const idPrefix = new RegExp(`^${passage.id.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(\\s|$)`, 'i');
  return notes.filter((n) => {
    if (idPrefix.test(n.text.trim())) {
      return true;
    }
    const text = normalize(n.text);
    return runs.some((run) => text.includes(run));
  });
}

/**
 * Each passage id with quest notes -> their labels ("Q1-N1" -> ["B"]). The
 * notes are the active map's, so with section (the active quest's section,
 * from currentSection) only its passages are matched; -1 matches them all.
 */
export function noteLabels(sections: readonly ScriptSection[], notes: readonly NoteDoc[], section = -1): Map<string, string[]> {
  const out = new Map<string, string[]>();
  const passages = section >= 0 ? sections[section]?.passages ?? [] : sections.flatMap((s) => s.passages);
  for (const passage of passages) {
    const labels = passageNotes(passage, notes).map((n) => n.label);
    if (labels.length) {
      out.set(passage.id, labels);
    }
  }
  return out;
}
