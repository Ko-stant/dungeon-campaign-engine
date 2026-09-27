/** Session state and API shapes, mirroring internal/tracker and internal/app (Go). */
import type { DoorState, TrapState } from '../board/model.ts';
import type { BoardDoc, QuestDoc } from '../maps/types.ts';

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
}

export interface LiveDoor {
  id: string;
  state: DoorState;
  found: boolean;
  locked: boolean;
}

export interface LiveTrap {
  id: string;
  state: TrapState;
}

export interface SessionState {
  version: number;
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
