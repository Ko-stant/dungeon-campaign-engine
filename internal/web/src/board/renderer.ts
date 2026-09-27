/**
 * Canvas renderer for a BoardView. Holds no game state: callers pass the view
 * to draw() every time. The pure maths lives in geometry.ts and model.ts; this
 * file only turns it into canvas calls, and is checked visually.
 */
import {
  computeGridMetrics,
  doorRect,
  edgeSegment,
  footprintRect,
  furnitureDrawBox,
  rotatedFootprint,
  tileRect,
  type Edge,
  type GridMetrics,
  type TileCoord,
} from './geometry.ts';
import { CORRIDOR, VOID, deriveWalls, tileAt, type BoardView, type PieceView, type TrapState } from './model.ts';

export interface BoardTheme {
  background: string;
  corridor: string;
  room: string;
  rock: string;
  grid: string;
  wall: string;
  doorClosed: string;
  doorOpen: string;
  doorSecret: string;
  doorGate: string;
  lock: string;
  blocked: string;
  furniture: string;
  monster: string;
  hero: string;
  label: string;
  undiscovered: string;
  highlight: string;
  preview: string;
  start: string;
  exit: string;
  note: string;
  trap: Record<TrapState, string>;
}

/** Reads the --rgb-* theme channels defined in styles/index.css. */
export function readTheme(el: Element = document.documentElement): BoardTheme {
  const css = getComputedStyle(el);
  const rgb = (name: string, fallback: string, alpha = 1): string => {
    const channels = css.getPropertyValue(`--rgb-${name}`).trim() || fallback;
    return alpha === 1 ? `rgb(${channels})` : `rgb(${channels} / ${alpha})`;
  };
  return {
    background: rgb('surface', '20 22 28'),
    corridor: rgb('surface', '20 22 28'),
    room: rgb('surface-2', '26 29 36'),
    rock: 'rgb(9 10 13)',
    grid: rgb('border', '95 104 123', 0.18),
    wall: rgb('brand', '130 166 255', 0.85),
    doorClosed: rgb('brand', '130 166 255'),
    doorOpen: rgb('positive', '74 222 128'),
    doorSecret: 'rgb(192 132 252)',
    doorGate: 'rgb(203 213 225)',
    lock: rgb('warning', '250 204 21'),
    blocked: rgb('danger', '248 113 113', 0.5),
    furniture: rgb('border', '95 104 123', 0.6),
    monster: 'rgb(220 20 60)',
    hero: rgb('accent', '255 160 122'),
    label: rgb('content', '231 236 243'),
    undiscovered: 'rgb(0 0 0 / 0.55)',
    highlight: rgb('warning', '250 204 21'),
    preview: rgb('warning', '250 204 21', 0.25),
    start: rgb('positive', '74 222 128', 0.22),
    exit: 'rgb(217 70 239 / 0.3)',
    note: rgb('warning', '250 204 21'),
    trap: {
      hidden: rgb('border', '95 104 123', 0.7),
      revealed: rgb('warning', '250 204 21'),
      triggered: rgb('danger', '248 113 113'),
      disarmed: rgb('positive', '74 222 128'),
    },
  };
}

/** Loads images once and asks for a redraw when each one arrives. */
export class ImageCache {
  readonly #images = new Map<string, HTMLImageElement | null>();
  readonly #onLoad: () => void;

  constructor(onLoad: () => void) {
    this.#onLoad = onLoad;
  }

  /** The loaded image, or undefined while loading or if it failed. */
  get(url: string): HTMLImageElement | undefined {
    const cached = this.#images.get(url);
    if (cached !== undefined) {
      return cached ?? undefined;
    }
    const img = new Image();
    this.#images.set(url, null);
    img.onload = () => {
      this.#images.set(url, img);
      this.#onLoad();
    };
    img.onerror = () => {
      console.warn(`board image failed to load: ${url}`);
    };
    img.src = url.startsWith('/') ? url : `/${url}`;
    return undefined;
  }
}

export interface Highlights {
  tile?: TileCoord | null;
  edge?: Edge | null;
  selectedId?: string | null;
  /** Inclusive rectangle preview (drag-fill), corner to corner. */
  rect?: { from: TileCoord; to: TileCoord } | null;
  /** Individual tiles to preview (drag-paint path). */
  tiles?: readonly TileCoord[] | null;
}

export class BoardRenderer {
  readonly #canvas: HTMLCanvasElement;
  readonly #ctx: CanvasRenderingContext2D;
  readonly #images: ImageCache;
  #theme: BoardTheme;
  #metrics: GridMetrics | null = null;

  constructor(canvas: HTMLCanvasElement, requestRedraw: () => void, theme: BoardTheme = readTheme()) {
    const ctx = canvas.getContext('2d');
    if (!ctx) {
      throw new Error('2D canvas context unavailable');
    }
    this.#canvas = canvas;
    this.#ctx = ctx;
    this.#images = new ImageCache(requestRedraw);
    this.#theme = theme;
  }

  /** Metrics from the most recent draw, for hit-testing pointer events. */
  get metrics(): GridMetrics | null {
    return this.#metrics;
  }

  setTheme(theme: BoardTheme): void {
    this.#theme = theme;
  }

  draw(view: BoardView, highlights: Highlights = {}): void {
    const { width, height } = this.#resizeToDisplay();
    const m = computeGridMetrics(width, height, view.cols, view.rows);
    this.#metrics = m;
    const ctx = this.#ctx;

    ctx.fillStyle = this.#theme.background;
    ctx.fillRect(0, 0, width, height);

    this.#drawTiles(view, m);
    this.#drawSquareMarks(view.startTiles, this.#theme.start, m);
    this.#drawSquareMarks(view.exitTiles, this.#theme.exit, m);
    this.#drawGrid(m);
    this.#drawBlockedSquares(view, m);
    this.#drawFurniture(view, m);
    this.#drawTraps(view, m);
    this.#drawWalls(view, m);
    this.#drawDoors(view, m, highlights.selectedId ?? null);
    this.#drawPieces(view.monsters, m, this.#theme.monster, highlights.selectedId ?? null);
    this.#drawPieces(view.heroes, m, this.#theme.hero, highlights.selectedId ?? null);
    this.#drawNotes(view, m, highlights.selectedId ?? null);
    this.#drawDiscovery(view, m);
    this.#drawHighlights(m, highlights);
  }

  /** Matches the canvas backing store to its CSS size and the device pixel ratio. */
  #resizeToDisplay(): { width: number; height: number } {
    const rect = this.#canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    const w = Math.max(1, Math.floor(rect.width * dpr));
    const h = Math.max(1, Math.floor(rect.height * dpr));
    if (this.#canvas.width !== w || this.#canvas.height !== h) {
      this.#canvas.width = w;
      this.#canvas.height = h;
    }
    this.#ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return { width: rect.width, height: rect.height };
  }

  #drawTiles(view: BoardView, m: GridMetrics): void {
    const ctx = this.#ctx;
    view.regions.forEach((region, i) => {
      ctx.fillStyle = region === VOID ? this.#theme.rock : region === CORRIDOR ? this.#theme.corridor : this.#theme.room;
      const r = tileRect(m, tileAt(view.cols, i));
      ctx.fillRect(r.x, r.y, r.w, r.h);
      const color = region > CORRIDOR ? view.roomColors?.get(region) : undefined;
      if (color) {
        // A translucent wash keeps the grid, walls and artwork readable.
        ctx.save();
        ctx.globalAlpha = 0.35;
        ctx.fillStyle = color;
        ctx.fillRect(r.x, r.y, r.w, r.h);
        ctx.restore();
      }
    });
  }

  /** Fills marked squares (start or exit) with a translucent color. */
  #drawSquareMarks(tiles: readonly TileCoord[] | undefined, color: string, m: GridMetrics): void {
    if (!tiles?.length) {
      return;
    }
    const ctx = this.#ctx;
    ctx.save();
    ctx.fillStyle = color;
    for (const t of tiles) {
      const r = tileRect(m, t);
      ctx.fillRect(r.x, r.y, r.w, r.h);
    }
    ctx.restore();
  }

  #drawNotes(view: BoardView, m: GridMetrics, selectedId: string | null): void {
    if (!view.notes?.length) {
      return;
    }
    const ctx = this.#ctx;
    for (const n of view.notes) {
      const r = tileRect(m, n.at);
      const radius = Math.max(5, m.tile * 0.22);
      const cx = r.x + r.w - radius - 1;
      const cy = r.y + radius + 1;
      ctx.save();
      ctx.beginPath();
      ctx.arc(cx, cy, radius, 0, Math.PI * 2);
      ctx.fillStyle = this.#theme.note;
      ctx.fill();
      if (n.id === selectedId) {
        ctx.strokeStyle = this.#theme.highlight;
        ctx.lineWidth = 2;
        ctx.stroke();
      }
      ctx.restore();
      this.#label(n.label, cx, cy, radius * 1.2, 'rgb(0 0 0)');
    }
  }

  #drawGrid(m: GridMetrics): void {
    const ctx = this.#ctx;
    ctx.save();
    ctx.strokeStyle = this.#theme.grid;
    ctx.lineWidth = 1;
    ctx.beginPath();
    for (let x = 0; x <= m.cols; x++) {
      const px = m.originX + x * m.tile + 0.5;
      ctx.moveTo(px, m.originY);
      ctx.lineTo(px, m.originY + m.rows * m.tile);
    }
    for (let y = 0; y <= m.rows; y++) {
      const py = m.originY + y * m.tile + 0.5;
      ctx.moveTo(m.originX, py);
      ctx.lineTo(m.originX + m.cols * m.tile, py);
    }
    ctx.stroke();
    ctx.restore();
  }

  #drawWalls(view: BoardView, m: GridMetrics): void {
    const ctx = this.#ctx;
    ctx.save();
    ctx.strokeStyle = this.#theme.wall;
    ctx.lineWidth = Math.max(1, Math.round(m.tile / 15));
    ctx.lineCap = 'square';
    ctx.beginPath();
    for (const e of deriveWalls(view.cols, view.rows, view.regions, view.drawnWalls)) {
      const s = edgeSegment(m, e);
      ctx.moveTo(s.x1 + 0.5, s.y1 + 0.5);
      ctx.lineTo(s.x2 + 0.5, s.y2 + 0.5);
    }
    ctx.stroke();
    ctx.restore();
  }

  #drawDoors(view: BoardView, m: GridMetrics, selectedId: string | null): void {
    const ctx = this.#ctx;
    for (const door of view.doors) {
      const r = doorRect(m, door.edge);
      const color =
        door.kind === 'secret' ? this.#theme.doorSecret
          : door.kind === 'gate' ? this.#theme.doorGate
            : door.state === 'open' ? this.#theme.doorOpen : this.#theme.doorClosed;
      ctx.save();
      if (door.kind === 'gate' && door.state !== 'open') {
        // A closed gate: bars across the opening.
        ctx.fillStyle = this.#theme.background;
        ctx.fillRect(r.x, r.y, r.w, r.h);
        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;
        ctx.strokeRect(r.x, r.y, r.w, r.h);
        ctx.beginPath();
        const vertical = r.h > r.w;
        for (let i = 1; i <= 3; i++) {
          if (vertical) {
            const y = r.y + (r.h * i) / 4;
            ctx.moveTo(r.x - 2, y);
            ctx.lineTo(r.x + r.w + 2, y);
          } else {
            const x = r.x + (r.w * i) / 4;
            ctx.moveTo(x, r.y - 2);
            ctx.lineTo(x, r.y + r.h + 2);
          }
        }
        ctx.stroke();
      } else if (door.state === 'open') {
        ctx.fillStyle = this.#theme.background;
        ctx.fillRect(r.x, r.y, r.w, r.h);
        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;
        ctx.strokeRect(r.x, r.y, r.w, r.h);
      } else {
        ctx.fillStyle = color;
        ctx.fillRect(r.x, r.y, r.w, r.h);
      }
      if (door.kind === 'secret') {
        ctx.setLineDash([2, 2]);
        ctx.strokeStyle = this.#theme.label;
        ctx.strokeRect(r.x, r.y, r.w, r.h);
        ctx.setLineDash([]);
      }
      if (door.locked) {
        const size = Math.max(4, m.tile * 0.22);
        const cx = r.x + r.w / 2;
        const cy = r.y + r.h / 2;
        ctx.fillStyle = this.#theme.lock;
        ctx.strokeStyle = 'rgb(0 0 0 / 0.7)';
        ctx.lineWidth = 1;
        ctx.fillRect(cx - size / 2, cy - size / 2, size, size);
        ctx.strokeRect(cx - size / 2, cy - size / 2, size, size);
      }
      if (door.id === selectedId) {
        ctx.setLineDash([]);
        ctx.strokeStyle = this.#theme.highlight;
        ctx.lineWidth = 2;
        ctx.strokeRect(r.x - 2, r.y - 2, r.w + 4, r.h + 4);
      }
      ctx.restore();
    }
  }

  #drawBlockedSquares(view: BoardView, m: GridMetrics): void {
    const ctx = this.#ctx;
    const img = this.#images.get('assets/tiles_cleaned/general/blocked_tile_1x1.png');
    for (const b of view.blockedSquares) {
      for (let y = b.y; y < b.y + b.h; y++) {
        for (let x = b.x; x < b.x + b.w; x++) {
          const r = tileRect(m, { x, y });
          if (img) {
            ctx.drawImage(img, r.x, r.y, r.w, r.h);
          } else {
            ctx.fillStyle = this.#theme.blocked;
            ctx.fillRect(r.x + 1, r.y + 1, r.w - 2, r.h - 2);
          }
        }
      }
      if (b.hiddenDoor) {
        // GM marker: a dashed purple outline in the secret-door color.
        const r = footprintRect(m, { x: b.x, y: b.y }, b.w, b.h);
        ctx.save();
        ctx.strokeStyle = this.#theme.doorSecret;
        ctx.lineWidth = Math.max(2, m.tile / 10);
        ctx.setLineDash([4, 3]);
        ctx.strokeRect(r.x + 1.5, r.y + 1.5, r.w - 3, r.h - 3);
        ctx.restore();
      }
    }
  }

  #drawFurniture(view: BoardView, m: GridMetrics): void {
    const ctx = this.#ctx;
    for (const f of view.furniture) {
      const img = f.image ? this.#images.get(f.image) : undefined;
      if (img) {
        const box = furnitureDrawBox(m, f.at, f.width, f.height, f.rotation);
        ctx.save();
        ctx.translate(box.cx, box.cy);
        ctx.rotate(box.radians);
        ctx.drawImage(img, -box.width / 2, -box.height / 2, box.width, box.height);
        ctx.restore();
        continue;
      }
      const size = rotatedFootprint(f.width, f.height, f.rotation);
      const r = footprintRect(m, f.at, size.width, size.height);
      ctx.fillStyle = this.#theme.furniture;
      ctx.fillRect(r.x + 2, r.y + 2, r.w - 4, r.h - 4);
      this.#label(f.type.replaceAll('_', ' '), r.cx, r.cy, m.tile * 0.3);
    }
  }

  #drawTraps(view: BoardView, m: GridMetrics): void {
    const ctx = this.#ctx;
    for (const trap of view.traps) {
      const r = tileRect(m, trap.at);
      const s = m.tile * 0.3;
      ctx.save();
      ctx.strokeStyle = this.#theme.trap[trap.state];
      ctx.fillStyle = this.#theme.trap[trap.state];
      ctx.lineWidth = 2;
      if (trap.state === 'hidden') {
        ctx.setLineDash([3, 3]);
      }
      ctx.beginPath();
      ctx.moveTo(r.cx, r.cy - s);
      ctx.lineTo(r.cx + s, r.cy + s * 0.8);
      ctx.lineTo(r.cx - s, r.cy + s * 0.8);
      ctx.closePath();
      if (trap.state === 'triggered') {
        ctx.fill();
      } else {
        ctx.stroke();
      }
      ctx.restore();
    }
  }

  #drawPieces(pieces: readonly PieceView[], m: GridMetrics, color: string, selectedId: string | null): void {
    const ctx = this.#ctx;
    for (const p of pieces) {
      const r = tileRect(m, p.at);
      const img = p.image ? this.#images.get(p.image) : undefined;
      ctx.save();
      if (p.dim) {
        ctx.globalAlpha = 0.4;
      }
      if (img) {
        ctx.drawImage(img, r.x, r.y, r.w, r.h);
      } else {
        ctx.beginPath();
        ctx.arc(r.cx, r.cy, Math.max(2, m.tile * 0.35), 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.fill();
        if (p.label) {
          this.#label(p.label.slice(0, 2), r.cx, r.cy, m.tile * 0.35, 'rgb(0 0 0)');
        }
      }
      ctx.restore();
      if (p.id === selectedId) {
        ctx.save();
        ctx.strokeStyle = this.#theme.highlight;
        ctx.lineWidth = 2;
        ctx.strokeRect(r.x + 1, r.y + 1, r.w - 2, r.h - 2);
        ctx.restore();
      }
    }
  }

  #drawDiscovery(view: BoardView, m: GridMetrics): void {
    if (!view.discovered) {
      return;
    }
    const ctx = this.#ctx;
    ctx.fillStyle = this.#theme.undiscovered;
    for (let i = 0; i < view.regions.length; i++) {
      if (!view.discovered.has(i)) {
        const r = tileRect(m, tileAt(view.cols, i));
        ctx.fillRect(r.x, r.y, r.w, r.h);
      }
    }
  }

  #drawHighlights(m: GridMetrics, h: Highlights): void {
    const ctx = this.#ctx;
    ctx.save();
    ctx.strokeStyle = this.#theme.highlight;
    ctx.lineWidth = 2;
    if (h.tile) {
      const r = tileRect(m, h.tile);
      ctx.strokeRect(r.x + 1, r.y + 1, r.w - 2, r.h - 2);
    }
    if (h.edge) {
      const r = doorRect(m, h.edge);
      ctx.strokeRect(r.x, r.y, r.w, r.h);
    }
    if (h.tiles) {
      ctx.fillStyle = this.#theme.preview;
      for (const t of h.tiles) {
        const r = tileRect(m, t);
        ctx.fillRect(r.x, r.y, r.w, r.h);
      }
    }
    if (h.rect) {
      const x0 = Math.min(h.rect.from.x, h.rect.to.x);
      const y0 = Math.min(h.rect.from.y, h.rect.to.y);
      const x1 = Math.max(h.rect.from.x, h.rect.to.x);
      const y1 = Math.max(h.rect.from.y, h.rect.to.y);
      const a = footprintRect(m, { x: x0, y: y0 }, x1 - x0 + 1, y1 - y0 + 1);
      ctx.fillStyle = this.#theme.preview;
      ctx.fillRect(a.x, a.y, a.w, a.h);
      ctx.setLineDash([4, 3]);
      ctx.strokeRect(a.x + 0.5, a.y + 0.5, a.w - 1, a.h - 1);
    }
    ctx.restore();
  }

  #label(text: string, x: number, y: number, size: number, color = this.#theme.label): void {
    const ctx = this.#ctx;
    ctx.save();
    ctx.fillStyle = color;
    ctx.font = `600 ${Math.max(8, Math.round(size))}px ui-sans-serif, system-ui, sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(text, x, y);
    ctx.restore();
  }
}
