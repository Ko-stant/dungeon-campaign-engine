/**
 * A preview of the piece the GM is about to place, drawn the way the board
 * draws it: the footprint at the chosen rotation over a small grid, with the
 * anchor square (the square you click) outlined.
 */
import { piecePreviewLayout } from '../board/geometry.ts';
import { h } from './dom.ts';

export interface PreviewPiece {
  name: string;
  width: number;
  height: number;
  rotation: number;
  /** Catalog artwork; without it a colored block (custom monster) or a plain block is drawn. */
  image?: string | undefined;
  color?: string | undefined;
}

const MAX_PX = 176;

export function piecePreview(p: PreviewPiece): HTMLElement {
  const l = piecePreviewLayout(p.width, p.height, p.rotation, MAX_PX);
  const grid = `linear-gradient(to right, rgb(127 127 127 / 0.35) 1px, transparent 1px), linear-gradient(to bottom, rgb(127 127 127 / 0.35) 1px, transparent 1px)`;
  const art = p.image
    ? h('img', {
      src: p.image.startsWith('/') ? p.image : `/${p.image}`,
      alt: '',
      draggable: 'false',
      style: `position:absolute;left:50%;top:50%;width:${String(l.image.width)}px;height:${String(l.image.height)}px;max-width:none;transform:translate(-50%,-50%) rotate(${String(l.image.degrees)}deg)`,
    })
    : h('div', {
      class: 'absolute inset-1 flex items-center justify-center rounded text-center text-xs font-semibold',
      style: `background:${p.color ?? 'rgb(127 127 127 / 0.35)'}`,
    }, p.name);
  const turned = p.rotation === 90 || p.rotation === 270;
  const size = turned ? `${String(p.height)}×${String(p.width)}` : `${String(p.width)}×${String(p.height)}`;
  return h('figure', { class: 'space-y-1' },
    h('div', {
      class: 'relative overflow-hidden rounded border border-border/60',
      style: `width:${String(l.box.width + 1)}px;height:${String(l.box.height + 1)}px;background-image:${grid};background-size:${String(l.tile)}px ${String(l.tile)}px`,
      'aria-label': `Preview of ${p.name}`,
    },
    art,
    h('div', {
      class: 'pointer-events-none absolute border-2 border-amber-400',
      style: `left:${String(l.anchor.x)}px;top:${String(l.anchor.y)}px;width:${String(l.anchor.size + 1)}px;height:${String(l.anchor.size + 1)}px`,
      title: 'The square you click',
    })),
    h('figcaption', { class: 'text-xs opacity-60' }, `${size} squares. The outlined square is the one you click.`));
}
