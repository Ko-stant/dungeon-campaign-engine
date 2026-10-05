/** The player screen's data, mirroring internal/tracker/player.go and internal/app/player_api.go (Go). */
import type { Edge, Rotation } from '../board/geometry.ts';
import type { DoorState, TrapState } from '../board/model.ts';
import type { FurnitureDef, TrapDef } from '../maps/types.ts';
import type { Effect, HeroCombat, HeroStatus, MonsterCombat } from '../tracker/types.ts';

export interface PlayerDoor {
  id: string;
  edge: Edge;
  span?: number;
  kind: 'normal' | 'gate' | 'exit';
  state: DoorState;
  locked?: boolean;
}

export interface PlayerBlock {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface PlayerFurniture {
  id: string;
  type: string;
  x: number;
  y: number;
  rotation: Rotation;
}

export interface PlayerTrap {
  id: string;
  kind: string;
  x: number;
  y: number;
  rotation?: Rotation;
  state: TrapState;
}

export interface PlayerMonster {
  id: string;
  type: string;
  name: string;
  x: number;
  y: number;
  width: number;
  height: number;
  color?: string;
  /** Missing while the GM keeps monsters' Body off the screen. */
  body?: number;
  maxBody?: number;
  /** Hurt and at a quarter of its Body or less (Faltering). */
  wounded?: boolean;
  combat?: MonsterCombat;
  effects?: Effect[] | null;
}

export interface PlayerHero {
  id: string;
  name: string;
  class: string;
  x: number;
  y: number;
  placed: boolean;
  body: number;
  maxBody: number;
  mind: number;
  maxMind: number;
  mana?: number;
  manaCap?: number;
  status: HeroStatus;
  effects?: Effect[] | null;
  /** Class stats plus equipped items. */
  combat?: HeroCombat | null;
  /** Accuracy bonus from misses in a row. */
  determination?: number;
}

export interface PlayerState {
  questName: string;
  /** The quest's aim as the players hear it (rules mode); missing when none. */
  goal?: string;
  round: number;
  fight?: boolean;
  width: number;
  height: number;
  regions: number[];
  drawnWalls?: Edge[] | null;
  discovered: number[];
  doors: PlayerDoor[];
  furniture: PlayerFurniture[];
  blocks: PlayerBlock[];
  traps: PlayerTrap[];
  monsters: PlayerMonster[];
  heroes: PlayerHero[];
  showMonsterBody: boolean;
}

export interface PlayerMonsterDef {
  id: string;
  name: string;
  movement?: number;
  image?: string;
  width?: number;
  height?: number;
  color?: string;
}

export interface PlayerCatalog {
  furniture: FurnitureDef[];
  traps: TrapDef[];
  monsters: PlayerMonsterDef[];
  heroes: { id: string; name: string; color?: string }[];
}

export interface PlayerEvent {
  seq: number;
  round: number;
  summary: string;
  createdAt: string;
}

export interface PlayerResponse {
  state: PlayerState;
  events: PlayerEvent[];
  eventSeq: number;
  catalog: PlayerCatalog;
}

/**
 * Pushed on every change; event is null when the players hear nothing about it. feed (null
 * otherwise) replaces the whole feed when earlier lines changed: a sighting taken back.
 */
export interface PlayerUpdate {
  state: PlayerState;
  event: PlayerEvent | null;
  eventSeq: number;
  feed: PlayerEvent[] | null;
}
