/**
 * Parity fixtures between the TypeScript simulator and the Go rules engine.
 * The TS code is the reference: these cases are computed here and checked
 * into the Go packages' testdata, where Go tests replay them. Regenerate with
 *   bun run parity:gen
 * and scripts/parity/gen.test.ts fails if the checked-in files go stale.
 */

import { doorCovers, edgeBetween, footprintTiles, type Edge, type TileCoord } from '../../internal/web/src/board/geometry.ts';
import { deriveWalls } from '../../internal/web/src/board/model.ts';
import { walkDistances } from '../../internal/web/src/combat/encounters.ts';
import type { HeroAttacker, HeroDefender, MonsterAttacker } from '../../internal/web/src/combat/odds.ts';
import { heroStrike, monsterStrike, seededRoller, type Roller, type StrikeOptions } from '../../internal/web/src/combat/simulate.ts';
import { formatDice, parseDice, type DiceExpr } from '../../internal/web/src/dice/dice.ts';
import type { BoardDoc, DoorDoc, QuestDoc, RectDoc } from '../../internal/web/src/maps/types.ts';

/** Fixture file paths, relative to the repository root. */
export const FIXTURE_FILES = {
  mulberry: 'internal/combat/testdata/parity/mulberry.json',
  heroStrikes: 'internal/combat/testdata/parity/hero_strikes.json',
  monsterStrikes: 'internal/combat/testdata/parity/monster_strikes.json',
  geometry: 'internal/maps/testdata/parity/geometry.json',
} as const;

const SIDES = [4, 6, 8, 10, 12, 20];

function dice(s: string): DiceExpr {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

/** Passes dice through and records them. */
function recording(inner: Roller): Roller & { dice: number[] } {
  const out: number[] = [];
  return {
    dice: out,
    die(sides: number): number {
      const v = inner.die(sides);
      out.push(v);
      return v;
    },
  };
}

/** Hands out fixed dice in order. */
function scripted(values: readonly number[]): Roller {
  let i = 0;
  return {
    die(sides: number): number {
      const v = values[i++];
      if (v === undefined || v < 1 || v > sides) {
        throw new Error(`no scripted d${sides}`);
      }
      return v;
    },
  };
}

/** Picks from a seeded roller; the fixtures themselves must be repeatable. */
function picker(seed: number) {
  const r = seededRoller(seed);
  return {
    int: (lo: number, hi: number): number => lo - 1 + r.die(hi - lo + 1),
    chance: (percent: number): boolean => r.die(100) <= percent,
    one: <T>(xs: readonly T[]): T => xs[r.die(xs.length) - 1] as T,
  };
}

// --- Mulberry32 ---

function mulberry() {
  const seeds = [0, 1, 2, 42, 1234567, 2147483648, 4294967295];
  return {
    sides: SIDES,
    cases: seeds.map((seed) => {
      const r = seededRoller(seed);
      return { seed, draws: Array.from({ length: 240 }, (_, i) => r.die(SIDES[i % SIDES.length] ?? 20)) };
    }),
  };
}

// --- Hero strikes ---

interface HeroCase {
  attacker: { hitDice: string; accuracy: number; critFrom: number; critMultiplier: number; nearMiss: number; damage: number };
  avoidance: number;
  options: StrikeOptions;
  seed?: number;
  dice: number[];
  result: ReturnType<typeof heroStrike>;
}

function heroCase(a: HeroAttacker, avoidance: number, options: StrikeOptions, roller: Roller, seed?: number): HeroCase {
  const rec = recording(roller);
  const result = heroStrike(a, avoidance, rec, options);
  return {
    attacker: { hitDice: formatDice(a.hitDice), accuracy: a.accuracy, critFrom: a.critFrom, critMultiplier: a.critMultiplier, nearMiss: a.nearMiss, damage: a.damage },
    avoidance,
    options,
    ...(seed === undefined ? {} : { seed }),
    dice: rec.dice,
    result,
  };
}

function heroStrikes() {
  const cases: HeroCase[] = [];
  // Every throw of small dice, so critical misses and special crits all show up.
  const small: HeroAttacker = { hitDice: dice('2d4'), accuracy: 1, critFrom: 18, critMultiplier: 2, nearMiss: 2, damage: 5 };
  for (let a = 1; a <= 4; a++) {
    for (let b = 1; b <= 4; b++) {
      for (let c = 1; c <= 20; c++) {
        cases.push(heroCase(small, 8, {}, scripted([a, b, c])));
      }
    }
  }
  const advantage: HeroAttacker = { hitDice: dice('1d6'), accuracy: 2, critFrom: 19, critMultiplier: 3, nearMiss: 1, damage: 4 };
  for (let a = 1; a <= 6; a++) {
    for (let b = 1; b <= 6; b++) {
      for (const c of [1, 2, 18, 19, 20]) {
        cases.push(heroCase(advantage, 7, { advantage: true }, scripted([a, b, c])));
      }
    }
  }
  const flat: HeroAttacker = { hitDice: dice('3'), accuracy: 0, critFrom: 20, critMultiplier: 2, nearMiss: 2, damage: 3 };
  for (let c = 1; c <= 20; c++) {
    cases.push(heroCase(flat, 4, {}, scripted([c])));
  }
  // Seeded random attacks across the option space.
  const p = picker(7);
  const hitDice = ['1d20', '2d10', '3d6', '1d12+1d8', '2d6+2', '1d20-1', '1d8+1d4'];
  for (let i = 0; i < 700; i++) {
    const a: HeroAttacker = {
      hitDice: dice(p.one(hitDice)),
      accuracy: p.int(-2, 5),
      critFrom: p.int(15, 20),
      critMultiplier: p.int(2, 3),
      nearMiss: p.int(0, 3),
      damage: p.int(1, 15),
    };
    const o: StrikeOptions = {};
    if (p.chance(40)) {
      o.bonus = p.int(-2, 4);
    }
    if (p.chance(20)) {
      o.advantage = true;
    }
    if (p.chance(10)) {
      o.sure = true;
    }
    if (p.chance(20)) {
      o.extraDamage = p.int(1, 4);
    }
    if (p.chance(10)) {
      o.damageFactor = p.int(1, 3);
    }
    const seed = 1000 + i;
    cases.push(heroCase(a, p.int(4, 24), o, seededRoller(seed), seed));
  }
  return { cases };
}

// --- Monster strikes ---

interface MonsterCase {
  monster: { hitDice: string; damage: number };
  defender: { defenseDice: string; baseAvoidance: number; mitigation: number };
  canDefend: boolean;
  seed?: number;
  dice: number[];
  result: ReturnType<typeof monsterStrike>;
}

function monsterCase(m: MonsterAttacker, d: HeroDefender, canDefend: boolean, roller: Roller, seed?: number): MonsterCase {
  const rec = recording(roller);
  const result = monsterStrike(m, d, rec, canDefend);
  return {
    monster: { hitDice: formatDice(m.hitDice), damage: m.damage },
    defender: { defenseDice: formatDice(d.defenseDice), baseAvoidance: d.baseAvoidance, mitigation: d.mitigation },
    canDefend,
    ...(seed === undefined ? {} : { seed }),
    dice: rec.dice,
    result,
  };
}

function monsterStrikes() {
  const cases: MonsterCase[] = [];
  const m: MonsterAttacker = { hitDice: dice('1d6'), damage: 6 };
  const d: HeroDefender = { defenseDice: dice('1d4'), baseAvoidance: 1, mitigation: 2 };
  const crits: [number, number][] = [
    [20, 20],
    [20, 19],
    [1, 20],
    [5, 5],
  ];
  for (let a = 1; a <= 6; a++) {
    for (let b = 1; b <= 4; b++) {
      for (const [c1, c2] of crits) {
        cases.push(monsterCase(m, d, true, scripted([a, b, c1, c2])));
      }
    }
    for (const [c1, c2] of crits) {
      cases.push(monsterCase(m, d, false, scripted([a, c1, c2])));
    }
  }
  const p = picker(11);
  const hitDice = ['2d8', '1d20', '3d6', '2d10+2', '1d12'];
  const defenseDice = ['1d6', '2d6', '1d8', '1d4+1d6', '1d10'];
  for (let i = 0; i < 400; i++) {
    const mm: MonsterAttacker = { hitDice: dice(p.one(hitDice)), damage: p.int(2, 16) };
    const dd: HeroDefender = { defenseDice: dice(p.one(defenseDice)), baseAvoidance: p.int(0, 8), mitigation: p.int(0, 5) };
    const seed = 5000 + i;
    cases.push(monsterCase(mm, dd, !p.chance(15), seededRoller(seed), seed));
  }
  return { cases };
}

// --- Geometry ---

function randomBoard(p: ReturnType<typeof picker>, width: number, height: number): { board: BoardDoc; quest: QuestDoc } {
  const regions: number[] = [];
  for (let i = 0; i < width * height; i++) {
    regions.push(p.one([-1, 0, 0, 1, 1, 2, 3]));
  }
  const board: BoardDoc = {
    version: 2,
    width,
    height,
    regions,
    rooms: [1, 2, 3].map((id) => ({ id, name: `Room ${id}` })),
    drawnWalls: [],
  };
  const walls = deriveWalls(width, height, regions);
  const interior = walls.filter((e) => (e.orientation === 'vertical' ? e.x > 1 && e.x <= width : e.y > 1 && e.y <= height));
  const doors: DoorDoc[] = [];
  for (let i = 0; i < 6 && interior.length > 0; i++) {
    const edge = p.one(interior);
    doors.push({ id: `d${i}`, edge, kind: 'normal', state: p.one(['open', 'closed'] as const), ...(p.chance(30) ? { span: 2 } : {}) });
  }
  const blocked: RectDoc[] = [];
  for (let i = 0; i < 3; i++) {
    blocked.push({ id: `b${i}`, x: p.int(1, width), y: p.int(1, height), w: p.int(1, 2), h: p.int(1, 2), ...(p.chance(30) ? { hiddenDoor: true } : {}) });
  }
  const quest: QuestDoc = {
    version: 2,
    boardChecksum: '',
    doors,
    blockedSquares: blocked,
    furniture: [],
    monsters: [],
    traps: [],
    notes: [],
    startTiles: [
      { x: p.int(1, width), y: p.int(1, height) },
      { x: p.int(1, width), y: p.int(1, height) },
    ],
    exitTiles: [],
    teleports: [],
  };
  return { board, quest };
}

function geometry() {
  const p = picker(23);
  const boards = Array.from({ length: 12 }, () => {
    const { board, quest } = randomBoard(p, p.int(4, 12), p.int(3, 9));
    const dist = walkDistances(board, quest);
    const distances = [...dist.entries()]
      .map(([k, v]): [number, number, number] => {
        const [x = 0, y = 0] = k.split(',').map(Number);
        return [x, y, v];
      })
      .sort((a, b) => a[1] - b[1] || a[0] - b[0]);
    return { board, quest, walls: deriveWalls(board.width, board.height, board.regions), distances };
  });

  const edges: { a: TileCoord; b: TileCoord; edge: Edge | null }[] = [];
  for (let dx = -2; dx <= 2; dx++) {
    for (let dy = -2; dy <= 2; dy++) {
      const a = { x: 5, y: 5 };
      const b = { x: 5 + dx, y: 5 + dy };
      edges.push({ a, b, edge: edgeBetween(a, b) });
    }
  }

  const footprints: { origin: TileCoord; width: number; height: number; rotation: number; tiles: TileCoord[] }[] = [];
  for (const [width, height] of [
    [1, 1],
    [2, 1],
    [2, 3],
    [3, 2],
  ] as const) {
    for (const rotation of [0, 90, 180, 270]) {
      const origin = { x: 3, y: 4 };
      footprints.push({ origin, width, height, rotation, tiles: footprintTiles(origin, width, height, rotation) });
    }
  }

  const covers: { door: { edge: Edge; span?: number }; edge: Edge; covers: boolean }[] = [];
  for (const door of [
    { edge: { x: 4, y: 2, orientation: 'vertical' as const }, span: 2 },
    { edge: { x: 4, y: 2, orientation: 'horizontal' as const }, span: 2 },
    { edge: { x: 4, y: 2, orientation: 'horizontal' as const } },
  ]) {
    for (const orientation of ['vertical', 'horizontal'] as const) {
      for (let x = 3; x <= 6; x++) {
        for (let y = 1; y <= 4; y++) {
          const edge = { x, y, orientation };
          covers.push({ door, edge, covers: doorCovers(door, edge) });
        }
      }
    }
  }

  return { boards, edges, footprints, covers };
}

/** Every fixture, keyed by file path. */
export function buildFixtures(): Record<string, unknown> {
  return {
    [FIXTURE_FILES.mulberry]: mulberry(),
    [FIXTURE_FILES.heroStrikes]: heroStrikes(),
    [FIXTURE_FILES.monsterStrikes]: monsterStrikes(),
    [FIXTURE_FILES.geometry]: geometry(),
  };
}

/**
 * Serializes a fixture with one case per line, so diffs stay readable and
 * files stay small: top-level arrays are written an element per line.
 */
export function formatFixture(data: unknown): string {
  if (typeof data !== 'object' || data === null || Array.isArray(data)) {
    return JSON.stringify(data) + '\n';
  }
  const parts = Object.entries(data).map(([k, v]) => {
    if (Array.isArray(v)) {
      return `  ${JSON.stringify(k)}: [\n${v.map((x) => '    ' + JSON.stringify(x)).join(',\n')}\n  ]`;
    }
    return `  ${JSON.stringify(k)}: ${JSON.stringify(v)}`;
  });
  return `{\n${parts.join(',\n')}\n}\n`;
}
