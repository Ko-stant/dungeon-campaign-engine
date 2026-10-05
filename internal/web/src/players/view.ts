/** Builds the renderer's view of the player screen's state (see internal/tracker/player.go). */
import type { BoardView, PieceView, TrapView } from '../board/model.ts';
import { withShape } from '../board/model.ts';
import type { PlayerCatalog, PlayerState } from './types.ts';

/** The whole layout is drawn; discovered squares get the "seen" color; only what the players know is on the board. */
export function playerBoardView(pv: PlayerState, catalog: PlayerCatalog): BoardView {
  const view: BoardView = {
    cols: pv.width,
    rows: pv.height,
    regions: pv.regions,
    doors: pv.doors.map((d) => ({ id: d.id, edge: d.edge, kind: d.kind, state: d.state, locked: d.locked ?? false, ...(d.span ? { span: d.span } : {}) })),
    blockedSquares: pv.blocks.map((b) => ({ id: b.id, x: b.x, y: b.y, w: b.w, h: b.h })),
    furniture: pv.furniture.map((f) => {
      const def = catalog.furniture.find((d) => d.id === f.type);
      return { id: f.id, type: f.type, at: { x: f.x, y: f.y }, width: def?.width ?? 1, height: def?.height ?? 1, rotation: f.rotation, ...(def?.image ? { image: def.image } : {}) };
    }),
    traps: pv.traps.map((t) => {
      const view: TrapView = { id: t.id, kind: t.kind, at: { x: t.x, y: t.y }, state: t.state };
      const def = catalog.traps.find((d) => d.id === t.kind);
      if (def) {
        view.width = def.width;
        view.height = def.height;
        view.rotation = t.rotation ?? 0;
        if (def.image) {
          view.image = def.image;
        }
      }
      return view;
    }),
    monsters: pv.monsters.map((m) => {
      const piece: PieceView = { id: m.id, type: m.type, at: { x: m.x, y: m.y }, label: m.name };
      const image = catalog.monsters.find((d) => d.id === m.type)?.image;
      if (image) {
        piece.image = image;
      }
      return withShape(piece, m);
    }),
    heroes: pv.heroes.filter((h) => h.placed).map((h) => ({ id: h.id, type: h.class, at: { x: h.x, y: h.y }, label: h.name })),
    seenTiles: new Set(pv.discovered),
  };
  if (pv.drawnWalls?.length) {
    view.drawnWalls = pv.drawnWalls;
  }
  return view;
}
