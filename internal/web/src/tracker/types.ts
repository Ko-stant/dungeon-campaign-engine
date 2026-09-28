/** Session state and API shapes, mirroring internal/tracker and internal/app (Go). */
import type { TileCoord } from '../board/geometry.ts';
import type { DoorState, TrapState } from '../board/model.ts';
import type { Ability, BoardDoc, QuestDoc } from '../maps/types.ts';

export type HeroStatus = 'active' | 'dead' | 'escaped';
export type Visibility = 'hidden' | 'seen';

export interface Hero {
  id: string;
  name: string;
  player?: string;
  class: string;
  x: number;
  y: number;
  placed: boolean;
  body: number;
  maxBody: number;
  mind: number;
  maxMind: number;
  gold: number;
  equipment?: string;
  notes?: string;
  status: HeroStatus;
  /** Inventory; null or missing on sessions started before items existed. */
  items?: Item[] | null;
  /** Mana and abilities copied from the hero's class when the session started. */
  mana?: number;
  maxMana?: number;
  abilities?: Ability[] | null;
  /** Ability id -> round it is ready again, for abilities still cooling down. */
  cooldowns?: Record<string, number> | null;
}

/** Something a hero carries; nothing is equipped and there are no limits. */
export interface Item {
  id: string;
  name: string;
  quantity: number;
  notes?: string;
}

export interface Monster {
  id: string;
  type: string;
  name: string;
  x: number;
  y: number;
  body: number;
  maxBody: number;
  mind: number;
  visibility: Visibility;
  alive: boolean;
  notes?: string;
  /** Size in squares and color, copied from the monster type (missing on older sessions). */
  width?: number;
  height?: number;
  color?: string;
}

export interface LiveDoor {
  id: string;
  state: DoorState;
  found: boolean;
  locked: boolean;
}

/** A trap's live state: a quest state, or removed from the board during play. */
export type LiveTrapState = TrapState | 'removed';

export interface LiveTrap {
  id: string;
  state: LiveTrapState;
  /** Where the trap is now, once moved during play (a rolling boulder). */
  at?: TileCoord;
}

/** A map the session has left (the tracker only needs to name it). */
export interface OtherMap {
  questId: string;
  questName: string;
}

/** One chapter of a campaign: a quest on its map. */
export interface Chapter {
  number: number;
  questId: string;
  questName: string;
  boardId: string;
  boardName: string;
}

export interface SessionState {
  version: number;
  /** The active map's quest. Missing on sessions started before travel existed. */
  questId?: string;
  /** Maps the party has left; travelling back restores them. */
  otherMaps?: OtherMap[];
  board: BoardDoc;
  quest: QuestDoc;
  questName: string;
  round: number;
  heroes: Hero[];
  monsters: Monster[];
  doors: LiveDoor[];
  traps: LiveTrap[];
  /** Quest blocked squares removed during play. Missing on sessions started before this existed. */
  removedBlocks?: string[];
  consumedNotes: string[];
  discovered: number[];
}

export interface Command {
  type: string;
  payload: Record<string, unknown>;
}

export interface SessionEvent {
  seq: number;
  round: number;
  kind: string;
  summary: string;
  payload: unknown;
  createdAt: string;
}

export interface SessionResponse {
  id: string;
  campaignId: string;
  name: string;
  status: 'active' | 'completed';
  state: SessionState;
  eventSeq: number;
  updatedAt: string;
}

export interface CommandResponse {
  state: SessionState;
  event: SessionEvent;
  eventSeq: number;
}
