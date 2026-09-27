/**
 * Throwaway comparison page (/dev/board): draws the base board and quest-01
 * through the new renderer. Hovering shows the tile and edge hit-tests.
 * Deleted in Phase 8.
 */
import { pixelToEdge, pixelToTile, type Edge, type TileCoord } from '../board/geometry.ts';
import type { BoardView } from '../board/model.ts';
import { BoardRenderer } from '../board/renderer.ts';

async function main(): Promise<void> {
  const canvas = document.querySelector<HTMLCanvasElement>('#board');
  const info = document.querySelector<HTMLElement>('#hover-info');
  if (!canvas || !info) {
    throw new Error('dev board page is missing #board or #hover-info');
  }

  const response = await fetch('/dev/board.json');
  if (!response.ok) {
    info.textContent = `failed to load board: ${response.status}`;
    return;
  }
  const view = (await response.json()) as BoardView;

  let hoverTile: TileCoord | null = null;
  let hoverEdge: Edge | null = null;
  let frame = 0;
  const redraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      renderer.draw(view, { tile: hoverTile, edge: hoverEdge });
    });
  };
  const renderer = new BoardRenderer(canvas, redraw);

  canvas.addEventListener('pointermove', (ev) => {
    const m = renderer.metrics;
    if (!m) {
      return;
    }
    const rect = canvas.getBoundingClientRect();
    const px = ev.clientX - rect.left;
    const py = ev.clientY - rect.top;
    hoverEdge = pixelToEdge(m, px, py);
    hoverTile = hoverEdge ? null : pixelToTile(m, px, py);
    info.textContent = hoverEdge
      ? `edge ${hoverEdge.orientation} (${hoverEdge.x},${hoverEdge.y})`
      : hoverTile
        ? `tile (${hoverTile.x},${hoverTile.y}) region ${view.regions[hoverTile.y * view.cols + hoverTile.x] ?? '?'}`
        : 'off board';
    redraw();
  });

  new ResizeObserver(redraw).observe(canvas);
  redraw();
}

void main();
