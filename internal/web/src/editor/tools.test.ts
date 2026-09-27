import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID } from '../board/model.ts';
import { DOC_VERSION, type BoardDoc, type Catalog } from '../maps/types.ts';
import { emptyQuest, placeMonster } from './model.ts';
import { applyClick, applyDrag, isDragTool, isQuestTool, lineTiles, type EditorDoc, type Tool } from './tools.ts';

const catalog: Catalog = {
  furniture: [{ id: 'chest', name: 'Chest', width: 1, height: 1, blocksMovement: true, blocksLineOfSight: false }],
  monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8 }],
  heroes: [],
};

function doc(withQuest = true): EditorDoc {
  const board: BoardDoc = { version: DOC_VERSION, width: 4, height: 3, regions: new Array<number>(12).fill(CORRIDOR), rooms: [] };
  return { board, quest: withQuest ? emptyQuest() : null };
}

const tile = (x: number, y: number) => ({ tile: { x, y }, edge: null });
const edge = (x: number, y: number, orientation: 'vertical' | 'horizontal') => ({ tile: null, edge: { x, y, orientation } });

describe('applyDrag (board tools)', () => {
  test('paint paints the visited tiles with the chosen region', () => {
    const out = applyDrag(doc(), { kind: 'paint', region: VOID }, [{ x: 1, y: 1 }, { x: 2, y: 1 }]);
    expect(out.board.regions.slice(0, 3)).toEqual([VOID, VOID, CORRIDOR]); // bottom row
  });

  test('fill paints the rectangle between the first and last points', () => {
    const out = applyDrag(doc(), { kind: 'fill', region: VOID }, [{ x: 2, y: 2 }, { x: 6, y: 6 }, { x: 3, y: 3 }]);
    const voids = out.board.regions.map((r, i) => (r === VOID ? i : -1)).filter((i) => i >= 0);
    expect(voids).toEqual([5, 6, 9, 10]); // x 2..3 on rows 2 and 3
  });

  test('blocked adds one blocked-square rectangle from the drag', () => {
    const out = applyDrag(doc(), { kind: 'blocked' }, [{ x: 2, y: 1 }, { x: 1, y: 2 }]);
    expect(out.quest?.blockedSquares).toEqual([{ id: 'blocked-1', x: 1, y: 1, w: 2, h: 2 }]);
  });

  test('quest tools do nothing when no quest is open', () => {
    const before = doc(false);
    expect(applyDrag(before, { kind: 'blocked' }, [{ x: 1, y: 1 }])).toBe(before);
  });

  test('an empty drag changes nothing', () => {
    const before = doc();
    expect(applyDrag(before, { kind: 'paint', region: VOID }, [])).toBe(before);
  });
});

describe('applyClick (quest tools)', () => {
  test('door cycles the clicked edge and ignores tile clicks', () => {
    const d1 = applyClick(doc(), { kind: 'door' }, edge(1, 1, 'vertical'), catalog);
    expect(d1.quest?.doors).toHaveLength(1);
    expect(applyClick(d1, { kind: 'door' }, tile(1, 1), catalog)).toBe(d1);
  });

  test('placement tools add pieces on the clicked tile', () => {
    const tools: Tool[] = [
      { kind: 'furniture', type: 'chest', rotation: 90 },
      { kind: 'monster', type: 'orc' },
      { kind: 'trap', trapKind: 'pit' },
      { kind: 'note' },
      { kind: 'start' },
      { kind: 'exit' },
    ];
    let d = doc();
    for (const t of tools) {
      d = applyClick(d, t, tile(2, 1), catalog);
    }
    expect(d.quest?.furniture).toEqual([{ id: 'furniture-1', type: 'chest', x: 2, y: 1, rotation: 90 }]);
    expect(d.quest?.monsters).toHaveLength(1);
    expect(d.quest?.traps).toHaveLength(1);
    expect(d.quest?.notes[0]?.label).toBe('A');
    expect(d.quest?.startTiles).toEqual([{ x: 2, y: 1 }]);
    expect(d.quest?.exitTiles).toEqual([{ x: 2, y: 1 }]);
  });

  test('erase removes the door under an edge, else the topmost item under a tile', () => {
    let d = applyClick(doc(), { kind: 'door' }, edge(1, 1, 'vertical'), catalog);
    d = { ...d, quest: placeMonster(d.quest ?? emptyQuest(), 'orc', { x: 3, y: 2 }) };
    d = applyClick(d, { kind: 'furniture', type: 'chest', rotation: 0 }, tile(3, 2), catalog);

    d = applyClick(d, { kind: 'erase' }, edge(1, 1, 'vertical'), catalog);
    expect(d.quest?.doors).toEqual([]);
    d = applyClick(d, { kind: 'erase' }, tile(3, 2), catalog);
    expect(d.quest?.monsters).toEqual([]);
    expect(d.quest?.furniture).toHaveLength(1);
    d = applyClick(d, { kind: 'erase' }, tile(3, 2), catalog);
    expect(d.quest?.furniture).toEqual([]);
  });

  test('clicks off the board or without a quest change nothing', () => {
    const d = doc();
    expect(applyClick(d, { kind: 'monster', type: 'orc' }, { tile: null, edge: null }, catalog)).toBe(d);
    const noQuest = doc(false);
    expect(applyClick(noQuest, { kind: 'monster', type: 'orc' }, tile(1, 1), catalog)).toBe(noQuest);
  });

  test('the wall tool toggles a drawn wall on the clicked edge, with or without a quest', () => {
    const noQuest = doc(false);
    const walled = applyClick(noQuest, { kind: 'wall' }, edge(2, 1, 'vertical'), catalog);
    expect(walled.board.drawnWalls).toEqual([{ x: 2, y: 1, orientation: 'vertical' }]);
    expect(applyClick(walled, { kind: 'wall' }, edge(2, 1, 'vertical'), catalog).board.drawnWalls).toEqual([]);
    expect(applyClick(noQuest, { kind: 'wall' }, tile(2, 1), catalog)).toBe(noQuest);
    expect(isQuestTool({ kind: 'wall' })).toBe(false);
    expect(isDragTool({ kind: 'wall' })).toBe(false);
  });

  test('the paint tool also works as a single click', () => {
    const out = applyClick(doc(), { kind: 'paint', region: VOID }, tile(4, 3), catalog); // top-right
    expect(out.board.regions[11]).toBe(VOID);
  });
});

describe('lineTiles', () => {
  test('includes both ends and every tile between, with no diagonal gaps', () => {
    expect(lineTiles({ x: 0, y: 0 }, { x: 3, y: 0 })).toEqual([{ x: 0, y: 0 }, { x: 1, y: 0 }, { x: 2, y: 0 }, { x: 3, y: 0 }]);
    const diag = lineTiles({ x: 0, y: 0 }, { x: 2, y: 2 });
    expect(diag[0]).toEqual({ x: 0, y: 0 });
    expect(diag[diag.length - 1]).toEqual({ x: 2, y: 2 });
    for (let i = 1; i < diag.length; i++) {
      const a = diag[i - 1];
      const b = diag[i];
      if (!a || !b) {
        throw new Error('unreachable');
      }
      // Each step moves to an orthogonal neighbour, so a brush never leaves diagonal gaps.
      expect(Math.abs(a.x - b.x) + Math.abs(a.y - b.y)).toBe(1);
    }
  });

  test('a zero-length line is the single tile', () => {
    expect(lineTiles({ x: 4, y: 5 }, { x: 4, y: 5 })).toEqual([{ x: 4, y: 5 }]);
  });
});
