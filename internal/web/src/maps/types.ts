/**
 * Board and quest documents, mirroring internal/maps (Go) JSON exactly, plus
 * the API response shapes from internal/app.
 */
import type { Edge, Rotation, TileCoord } from '../board/geometry.ts';
import type { DoorKind, DoorState, TrapState } from '../board/model.ts';

export interface Room {
  id: number;
  name: string;
  /** Optional "#rrggbb" fill, to tell rooms apart on screen. */
  color?: string;
}

/**
 * Document version written by this client and the server. Version 2 counts
 * squares from (1, 1) at the bottom-left (see board/geometry.ts).
 */
export const DOC_VERSION = 2;

export interface BoardDoc {
  version: number;
  width: number;
  height: number;
  /** Row-major from the bottom row up: -1 void, 0 corridor, >0 room id. */
  regions: number[];
  rooms: Room[];
  /** Walls the GM drew on interior edges, on top of the derived ones. Omitted when empty. */
  drawnWalls?: Edge[];
}

export interface DoorDoc {
  id: string;
  edge: Edge;
  kind: DoorKind;
  /** Starts locked. Omitted when false. Nothing enforces it. */
  locked?: boolean;
  state: DoorState;
  /** 2 for a two-wide door or gate: the edge and the next one along the wall. Omitted when 1. */
  span?: number;
  /** The item that unlocks it in rules mode (internal/maps Door.Key). Omitted when none. */
  key?: string;
}

export interface RectDoc {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  /** The block hides a secret door; finding it removes the block. Omitted when false. */
  hiddenDoor?: boolean;
}

export interface FurnitureDoc {
  id: string;
  type: string;
  x: number;
  y: number;
  rotation: Rotation;
}

export interface MonsterDoc {
  id: string;
  type: string;
  x: number;
  y: number;
  body?: number;
  mind?: number;
  notes?: string;
}

export interface TrapDoc {
  id: string;
  kind: string;
  x: number;
  y: number;
  furnitureId?: string;
  state: TrapState;
  /** Turns a catalog trap's footprint; absent means 0. */
  rotation?: Rotation;
  /** Short label (at most MAX_TRAP_LABEL characters) to tell traps apart, e.g. trigger "1". */
  label?: string;
}

export const MAX_TRAP_LABEL = 8;

/** A teleport square; the optional short label pairs squares up. */
export interface TeleportDoc {
  id: string;
  x: number;
  y: number;
  label?: string;
}

export interface NoteDoc {
  id: string;
  label: string;
  x: number;
  y: number;
  text: string;
}

export interface QuestDoc {
  version: number;
  boardChecksum: string;
  description?: string;
  wanderingMonster?: string;
  doors: DoorDoc[];
  blockedSquares: RectDoc[];
  furniture: FurnitureDoc[];
  monsters: MonsterDoc[];
  traps: TrapDoc[];
  notes: NoteDoc[];
  startTiles: TileCoord[];
  /** Where the heroes leave the dungeon. Missing or null on quests saved before exits existed. */
  exitTiles?: TileCoord[] | null;
  /** Teleport squares. Missing or null on quests saved before teleports existed. */
  teleports?: TeleportDoc[] | null;
  /** The quest's aim as the players hear it. Omitted when none. */
  goal?: string;
  /** What wins the quest in rules mode (internal/maps Objective); omitted when none. */
  objectives?: ObjectiveDoc[];
}

/** One objective: kill the named monsters (all when none are named), carry an item, or escape by the exits. */
export interface ObjectiveDoc {
  kind: 'kill' | 'collect' | 'escape';
  monsters?: string[];
  item?: string;
}

export interface Issue {
  code: string;
  itemId?: string;
  message: string;
}

export interface FurnitureDef {
  id: string;
  name: string;
  width: number;
  height: number;
  blocksMovement: boolean;
  blocksLineOfSight: boolean;
  image?: string;
}

/** A trap kind with artwork; quest traps whose kind has no entry are single-square markers. */
export interface TrapDef {
  id: string;
  name: string;
  width: number;
  height: number;
  image?: string;
  /** The GM can move it during play (a rolling boulder). */
  movable?: boolean;
}

export interface MonsterDef {
  id: string;
  name: string;
  body: number;
  mind: number;
  attack: number;
  defense: number;
  movement: number;
  image?: string;
  /** Custom monsters (made on the Monsters page): size in squares, color, notes. */
  width?: number;
  height?: number;
  color?: string;
  notes?: string;
  custom?: boolean;
}

/** A monster's name for pickers; custom and multi-square monsters show their size. */
export function monsterOptionLabel(m: MonsterDef): string {
  const w = m.width ?? 1;
  const h = m.height ?? 1;
  if (m.custom) {
    return `${m.name} (custom ${w}×${h})`;
  }
  return w > 1 || h > 1 ? `${m.name} (${w}×${h})` : m.name;
}

/** A monster's name for pickers during play: custom monsters read like any other. */
export function monsterPlayLabel(m: MonsterDef): string {
  const w = m.width ?? 1;
  const h = m.height ?? 1;
  return w > 1 || h > 1 ? `${m.name} (${w}×${h})` : m.name;
}

export type AbilityKind = 'active' | 'passive' | 'reaction' | 'spell';

/** A hero class ability. Cooldown counts rounds: used in round R, ready again in round R + cooldown. */
export interface Ability {
  id: string;
  name: string;
  kind: AbilityKind;
  manaCost?: number;
  cooldown?: number;
  text?: string;
}

export interface HeroDef {
  id: string;
  name: string;
  description?: string;
  body: number;
  mind: number;
  attack: number;
  defense: number;
  movementDice: number;
  /** Custom classes (made on the Classes page): dice expressions, accuracy, mana, exclusives, abilities. */
  custom?: boolean;
  color?: string;
  attackDice?: string;
  defenseDice?: string;
  movement?: string;
  accuracy?: number;
  mana?: number;
  exclusives?: string[];
  abilities?: Ability[];
  /** Combat (custom classes): crit range start on the d20, the class's own damage, avoidance and mitigation, mana per fight round. */
  critFrom?: number;
  damage?: number;
  avoidance?: number;
  mitigation?: number;
  manaRegen?: number;
}

export interface Catalog {
  furniture: FurnitureDef[];
  monsters: MonsterDef[];
  heroes: HeroDef[];
  traps: TrapDef[];
}

export interface BoardResponse {
  id: string;
  name: string;
  board: BoardDoc;
  updatedAt: string;
}

export interface QuestSummary {
  id: string;
  boardId: string;
  name: string;
  updatedAt: string;
}

export interface QuestResponse {
  id: string;
  boardId: string;
  name: string;
  quest: QuestDoc;
  issues: Issue[];
  updatedAt: string;
}
