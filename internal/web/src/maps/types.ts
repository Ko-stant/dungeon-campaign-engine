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
}

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

/** A monster's name for pickers; custom monsters show their size. */
export function monsterOptionLabel(m: MonsterDef): string {
  return m.custom ? `${m.name} (custom ${m.width ?? 1}×${m.height ?? 1})` : m.name;
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
}

export interface Catalog {
  furniture: FurnitureDef[];
  monsters: MonsterDef[];
  heroes: HeroDef[];
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
