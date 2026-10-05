/** Builds the renderer's view of a live session for the GM. */
import { withShape, type BoardView, type DoorView, type PieceView } from '../board/model.ts';
import { toBoardView, trapView } from '../editor/model.ts';
import type { Catalog } from '../maps/types.ts';
import type { SessionState } from './types.ts';

export interface ViewOptions {
  /**
   * The heroes' view: darken squares the heroes have not discovered and fade
   * furniture, doors and blocked squares the players have not been shown.
   */
  fog: boolean;
}

export function trackerView(s: SessionState, catalog: Catalog, opts: ViewOptions): BoardView {
  const consumed = new Set(s.consumedNotes);
  // Furniture, blocked squares and notes come from the frozen quest; pieces,
  // doors and traps from the live state.
  const base = toBoardView(s.board, { ...s.quest, monsters: [], startTiles: [], notes: s.quest.notes.filter((n) => !consumed.has(n.id)) }, catalog);

  const liveDoors = new Map(s.doors.map((d) => [d.id, d]));
  const liveTraps = new Map(s.traps.map((t) => [t.id, t]));

  const monsters: PieceView[] = s.monsters
    .filter((m) => m.alive)
    .map((m) => {
      const view: PieceView = { id: m.id, type: m.type, at: { x: m.x, y: m.y }, label: m.name };
      const image = catalog.monsters.find((d) => d.id === m.type)?.image;
      if (image) {
        view.image = image;
      }
      if (m.visibility === 'hidden') {
        view.dim = true;
      }
      return withShape(view, m);
    });

  const heroes: PieceView[] = s.heroes.filter((h) => h.placed).map((h) => ({ id: h.id, type: h.class, at: { x: h.x, y: h.y }, label: h.name }));

  const removed = new Set(s.removedBlocks ?? []);
  const seenFurniture = new Set(s.seenFurniture ?? []);
  const seenBlocks = new Set(s.seenBlocks ?? []);
  const view: BoardView = {
    ...base,
    furniture: base.furniture.map((f) => (opts.fog && !seenFurniture.has(f.id) ? { ...f, dim: true } : f)),
    blockedSquares: [
      ...base.blockedSquares.filter((b) => !removed.has(b.id)).map((b) => (opts.fog && !seenBlocks.has(b.id) ? { ...b, dim: true } : b)),
      ...(s.addedBlocks ?? []),
    ],
    doors: s.quest.doors.map((d) => {
      const live = liveDoors.get(d.id);
      const hiddenSecret = d.kind === 'secret' && !(live?.found ?? false);
      const kind = hiddenSecret ? 'secret' : d.kind === 'gate' || d.kind === 'exit' ? d.kind : 'normal';
      const door: DoorView = { id: d.id, edge: d.edge, kind, state: live?.state ?? d.state, locked: live?.locked ?? d.locked ?? false, span: d.span };
      if (opts.fog && !(live?.seen ?? false)) {
        door.dim = true;
      }
      return door;
    }),
    traps: s.quest.traps.flatMap((t) => {
      const live = liveTraps.get(t.id);
      const state = live?.state ?? t.state;
      return state === 'removed' ? [] : [trapView(catalog, t, state, live?.at)];
    }),
    monsters,
    heroes,
  };
  if (opts.fog) {
    view.discovered = new Set(s.discovered);
  }
  return view;
}
