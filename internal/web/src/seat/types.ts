/** A player's seat online, mirroring internal/tracker/seat.go and internal/app/seat.go (Go). */
import type { Effect, HeroCombat, Item } from '../tracker/types.ts';
import type { PlayerResponse, PlayerUpdate } from '../players/types.ts';

/** A command the seat may send now, ready to post (tracker.Action). */
export interface Action {
  label: string;
  command: { type: string; payload: Record<string, unknown> };
}

/** The turn under way (tracker.Turn). */
export interface Turn {
  heroId: string;
  moveRoll?: number;
  moveLeft?: number;
  acted?: boolean;
  moveDone?: boolean;
}

export interface SeatAbility {
  id: string;
  name: string;
  kind: string;
  manaCost?: number;
  cooldown?: number;
  text?: string;
  /** Rounds until it is ready again; missing when ready. */
  readyIn?: number;
}

export interface SeatHero {
  id: string;
  name: string;
  className: string;
  placed: boolean;
  x: number;
  y: number;
  status: string;
  body: number;
  maxBody: number;
  mind: number;
  maxMind: number;
  mana?: number;
  maxMana?: number;
  movement?: string;
  determination?: number;
  combat?: HeroCombat | null;
  abilities?: SeatAbility[] | null;
  items: Item[];
  effects?: Effect[] | null;
  turn?: Turn;
  acted?: boolean;
  skipNext?: boolean;
  actions: Action[];
}

export interface SeatState {
  round: number;
  /** heroes, monsters or over; missing until the GM starts online play. */
  phase?: string;
  turnHero?: string;
  outcome?: string;
  heroes: SeatHero[];
}

/** A player connected to the session, with the heroes they play. */
export interface Present {
  name: string;
  heroes: string[];
}

/** GET /api/sessions/{id}/seat. */
export interface SeatResponse extends PlayerResponse {
  seat: SeatState;
  presence: Present[];
}

/** One message on the seat stream: a change, or who is here. */
export interface SeatUpdate {
  player?: PlayerUpdate;
  seat?: SeatState;
  presence?: Present[];
}
