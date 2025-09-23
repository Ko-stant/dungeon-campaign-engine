/**
 * @typedef {Object} TileAddress
 * @property {string} segmentId
 * @property {number} x
 * @property {number} y
 */

/**
 * @typedef {Object} EntityLite
 * @property {string} id
 * @property {string} kind
 * @property {TileAddress} tile
 */

/**
 * @typedef {Object} Snapshot
 * @property {string} mapId
 * @property {string} packId
 * @property {number} turn
 * @property {number} lastEventId
 * @property {number} mapWidth
 * @property {number} mapHeight
 * @property {number} regionsCount
 * @property {number[]} tileRegionIds
 * @property {number[]} revealedRegionIds
 * @property {EntityLite[]} entities
 * @property {string} protocolVersion
 * @property {ThresholdLite[]} thresholds
 * @property {BlockingWallLite[]} blockingWalls
 * @property {FurnitureLite[]} furniture
 * @property {MonsterLite[]} monsters
 * @property {number[]} visibleRegionIds
 * @property {number} corridorRegionId
 * @property {number[]} knownRegionIds
 */

/**
 * @typedef {Object} ThresholdLite
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {"vertical"|"horizontal"} orientation
 * @property {"DoorSocket"} kind
 * @property {"open"|"closed"} [state]
 */

/**
 * @typedef {Object} BlockingWallLite
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {"vertical"|"horizontal"} orientation
 * @property {number} size
 */

/**
 * @typedef {Object} FurnitureLite
 * @property {string} id
 * @property {string} type
 * @property {TileAddress} tile
 * @property {{width: number, height: number}} gridSize
 * @property {number} [rotation] - 0, 90, 180, 270 degrees
 * @property {boolean} [swapAspectOnRotate] - Whether to swap width/height for 90/270 rotations
 * @property {string} tileImage
 * @property {string} tileImageCleaned
 * @property {{width: number, height: number}} pixelDimensions
 * @property {boolean} blocksLineOfSight
 * @property {boolean} blocksMovement
 * @property {string[]} [contains]
 */

/**
 * @typedef {Object} MonsterLite
 * @property {string} id
 * @property {string} type
 * @property {TileAddress} tile
 * @property {number} body
 * @property {number} MaxBody
 * @property {number} mind
 * @property {number} maxMind
 * @property {number} attackDice
 * @property {number} defenseDice
 * @property {boolean} isVisible
 * @property {boolean} isAlive
 */

/**
 * @typedef {Object} DiceRoll
 * @property {string} die
 * @property {number} result
 * @property {string} [combatResult]
 */

/**
 * @typedef {Object} HeroActionResult
 * @property {string} action
 * @property {boolean} success
 * @property {string} message
 * @property {DiceRoll[]} [diceRolls]
 * @property {number} [damage]
 */

// Action mode constants
export const ACTION_MODES = {
  MOVE: 'move',
  ATTACK: 'attack',
  SPELL: 'spell',
  SEARCH_TREASURE: 'search_treasure',
  SEARCH_TRAPS: 'search_traps',
  SEARCH_HIDDEN_DOORS: 'search_hidden_doors',
};

export const ACTION_NAMES = {
  [ACTION_MODES.MOVE]: 'Move',
  [ACTION_MODES.ATTACK]: 'Attack',
  [ACTION_MODES.SPELL]: 'Cast Spell',
  [ACTION_MODES.SEARCH_TREASURE]: 'Search Treasure',
  [ACTION_MODES.SEARCH_TRAPS]: 'Search Traps',
  [ACTION_MODES.SEARCH_HIDDEN_DOORS]: 'Search Hidden Doors',
};

// Monster colors for fallback rendering
export const MONSTER_COLORS = {
  goblin: { fill: 'rgb(34, 139, 34)', stroke: 'rgb(0, 100, 0)' },
  orc: { fill: 'rgb(139, 69, 19)', stroke: 'rgb(101, 67, 33)' },
  skeleton: { fill: 'rgb(245, 245, 220)', stroke: 'rgb(211, 211, 211)' },
  zombie: { fill: 'rgb(128, 128, 0)', stroke: 'rgb(85, 107, 47)' },
  mummy: { fill: 'rgb(222, 184, 135)', stroke: 'rgb(139, 69, 19)' },
  dread_warrior: { fill: 'rgb(105, 105, 105)', stroke: 'rgb(47, 79, 79)' },
  gargoyle: { fill: 'rgb(112, 128, 144)', stroke: 'rgb(47, 79, 79)' },
  abomination: { fill: 'rgb(139, 0, 139)', stroke: 'rgb(75, 0, 130)' },
  default: { fill: 'rgb(220, 20, 60)', stroke: 'rgb(139, 0, 0)' },
};

// Furniture type colors for fallback rendering
export const FURNITURE_COLORS = {
  stairwell: { fill: '--color-accent', stroke: '--color-content' },
  chest: { fill: '--color-positive', stroke: '--color-content' },
  table: { fill: '--color-surface-2', stroke: '--color-border-rgb' },
  alchemists_table: { fill: '--color-surface-2', stroke: '--color-border-rgb' },
  sorcerers_table: { fill: '--color-surface-2', stroke: '--color-border-rgb' },
  default: { fill: '--color-surface-3', stroke: '--color-border-rgb' },
};

// Image path constants
export const IMAGE_PATHS = {
  BLOCKING_WALL_1X1: 'assets/tiles_cleaned/general/blocked_tile_1x1.png',
  BLOCKING_WALL_2X1: 'assets/tiles_cleaned/general/blocked_tile_2x1.png',
  MONSTER_BASE: 'assets/tiles_cleaned/monsters/monster_',
};