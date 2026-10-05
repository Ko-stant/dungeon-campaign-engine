/** Session state and API shapes, mirroring internal/tracker and internal/app (Go). */
import type { TileCoord } from '../board/geometry.ts';
import type { DoorState, TrapState } from '../board/model.ts';
import type { Ability, BoardDoc, QuestDoc, RectDoc } from '../maps/types.ts';

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
  equipment?: string;
  notes?: string;
  status: HeroStatus;
  /** Inventory; null or missing on sessions started before items existed. */
  items?: Item[] | null;
  /** Mana and abilities copied from the hero's class when the session started; equipped items raise the maximum (manaCap). */
  mana?: number;
  maxMana?: number;
  abilities?: Ability[] | null;
  /** Ability id -> round it is ready again, for abilities still cooling down. */
  cooldowns?: Record<string, number> | null;
  /** Accuracy bonus from misses in a row (+2 a miss, up to +4; 0 after a hit or a fight's end). */
  determination?: number;
  /** The class's combat stats, frozen at session start (equipped items add to them: combatTotals); missing for built-in classes and older sessions. */
  combat?: HeroCombat | null;
  effects?: Effect[] | null;
}

/** A named condition with an optional countdown in fight rounds, mirroring tracker.Effect (Go). */
export interface Effect {
  id: string;
  name: string;
  /** Rounds left, counted down each fight round; missing or 0 lasts until removed. */
  rounds?: number;
  note?: string;
}

/** A hero's combat stats (The Three Plagues rules), mirroring tracker.Combat (Go). */
export interface HeroCombat {
  hitDice: string;
  accuracy: number;
  critFrom: number;
  damage: number;
  defenseDice: string;
  avoidance: number;
  mitigation: number;
  manaRegen?: number;
}

/** An item's bonuses to its hero's combat stats while equipped, mirroring tracker.ItemStats (Go). */
export interface ItemStats {
  damage?: number;
  accuracy?: number;
  avoidance?: number;
  mitigation?: number;
  mana?: number;
  manaRegen?: number;
}

/** Something a hero carries; no limits. An equipped item adds its stats to the hero's totals. */
export interface Item extends ItemStats {
  id: string;
  name: string;
  quantity: number;
  notes?: string;
  /** What it is (weapon, bow, chest...); free text, no slots. */
  kind?: string;
  equipped?: boolean;
  /** Using one (item.use) spends it and heals this much Body or restores this much mana. */
  healBody?: number;
  restoreMana?: number;
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
  /** The campaign's combat stats for this monster type, frozen when set up or added; missing when the campaign has none. */
  combat?: MonsterCombat | null;
  effects?: Effect[] | null;
}

/** A monster's combat stats (The Three Plagues rules), mirroring content.MonsterCombat (Go). */
export interface MonsterCombat {
  avoidance: number;
  hitDice: string;
  damage: number;
  ranged?: boolean;
  reach?: boolean;
  /** Each attack also strikes this many more heroes in a straight line. */
  line?: number;
  /** Each attack also blasts splashTargets heroes beside the target for splashDamage. */
  splashDamage?: number;
  splashTargets?: number;
  undead?: boolean;
  /** What the monster can do, in the GM's words (shown on the player screen). */
  abilities?: string;
}

export interface LiveDoor {
  id: string;
  state: DoorState;
  found: boolean;
  locked: boolean;
  /** Shown on the player screen. */
  seen?: boolean;
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
  /** Maps the party has left; traveling back restores them. */
  otherMaps?: OtherMap[];
  board: BoardDoc;
  quest: QuestDoc;
  questName: string;
  round: number;
  /** The party's purse; missing on sessions saved before it existed. */
  gold?: number;
  /** True while a fight is on: rounds then finish cooldowns, regenerate mana and count effects down. */
  fight?: boolean;
  heroes: Hero[];
  monsters: Monster[];
  doors: LiveDoor[];
  traps: LiveTrap[];
  /** Quest blocked squares removed during play. Missing on sessions started before this existed. */
  removedBlocks?: string[];
  /** Quest furniture and blocked squares shown on the player screen. */
  seenFurniture?: string[] | null;
  seenBlocks?: string[] | null;
  /** Blocked squares put down during play (a falling block, a stopped boulder). */
  addedBlocks?: RectDoc[] | null;
  /** Player screen settings. */
  players?: { hideMonsterBody?: boolean };
  consumedNotes: string[];
  discovered: number[];
  /** Read-aloud passage ids already read at the table. */
  readPassages?: string[] | null;
  /** The rules engine (online play); missing while the rules are off. */
  rules?: RulesState;
}

/** The rules engine's state, mirroring tracker.RulesState (Go). */
export interface RulesState {
  ruleset: string;
  /** heroes, monsters or over. */
  phase: string;
  /** Heroes whose turn this round is over. */
  acted?: string[] | null;
  turn?: Turn;
  /** Heroes who lose their next turn (a critical miss). */
  skipNext?: string[] | null;
  monstersMoved?: string[] | null;
  monstersActed?: string[] | null;
  /** won or lost, once the quest is over. */
  outcome?: string;
}

/** The hero's turn under way, mirroring tracker.Turn (Go). */
export interface Turn {
  heroId: string;
  moveRoll?: number;
  moveLeft?: number;
  acted?: boolean;
  moveDone?: boolean;
}

/** A campaign's read-aloud script, mirroring internal/script (Go). */
export interface ScriptNote {
  label: string;
  text: string;
}

export interface ScriptPart {
  speaker?: string;
  aside?: string;
  paragraphs: string[];
}

export interface ScriptPassage {
  id: string;
  title: string;
  notes?: ScriptNote[];
  parts: ScriptPart[];
  goalsTitle?: string;
  goals?: string[];
}

export interface ScriptSection {
  title: string;
  intro?: string;
  passages: ScriptPassage[];
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
