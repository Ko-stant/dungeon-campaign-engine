/**
 * Game state management and WebSocket handling
 */

import { ACTION_MODES } from './types.js';
import { restoreMovementStateFromSnapshot } from './movementPlanning.js';

class GameState {
  constructor() {
    this.snapshot = null;
    this.socketRef = null;
    this.redrawRef = null;
    this.patchCount = 0;

    // Entity tracking
    this.entityPositions = new Map();

    // Game objects
    this.thresholds = [];
    this.blockingWalls = [];
    this.furniture = [];
    this.monsters = [];

    // Region tracking
    this.revealedRegions = new Set();
    this.visibleNow = new Set();
    this.knownRegions = new Set();
    this.corridorRegionId = 0;

    // UI state
    this.selectedDoorId = null;
    this.selectedMonsterId = null;
    this.currentActionMode = ACTION_MODES.MOVE;

    // Image caches
    this.blockingWallImageCache = new Map();
    this.furnitureImageCache = new Map();
    this.monsterImageCache = new Map();

    // DOM elements (set during initialization)
    this.canvas = null;
    this.canvasContext = null;
    this.patchCountElement = null;
    this.toggleDoorButton = null;
  }

  /**
   * Initialize game state from snapshot data
   * @param {Snapshot} snapshot
   */
  initializeFromSnapshot(snapshot) {
    this.snapshot = snapshot;
    window.__SNAPSHOT__ = snapshot;

    // Initialize entity positions
    if (Array.isArray(snapshot?.entities)) {
      for (const e of snapshot.entities) {
        this.entityPositions.set(e.id, structuredClone(e.tile));
      }
    }

    // Initialize game objects
    this.thresholds = Array.isArray(snapshot?.thresholds) ? snapshot.thresholds.slice() : [];
    this.blockingWalls = Array.isArray(snapshot?.blockingWalls) ? snapshot.blockingWalls.slice() : [];
    this.furniture = Array.isArray(snapshot?.furniture) ? snapshot.furniture.slice() : [];
    this.monsters = Array.isArray(snapshot?.monsters) ? snapshot.monsters.slice() : [];

    // Initialize regions
    this.revealedRegions = new Set(Array.isArray(snapshot?.revealedRegionIds) ? snapshot.revealedRegionIds : []);
    this.visibleNow = new Set(Array.isArray(snapshot?.visibleRegionIds) ? snapshot.visibleRegionIds : []);
    this.knownRegions = new Set(Array.isArray(snapshot?.knownRegionIds) ? snapshot.knownRegionIds : []);
    this.corridorRegionId = typeof snapshot?.corridorRegionId === 'number' ? snapshot.corridorRegionId : 0;

    // Restore movement state from snapshot
    restoreMovementStateFromSnapshot(snapshot);
  }

  /**
   * Set DOM element references
   * @param {Object} elements
   */
  setDOMElements(elements) {
    this.canvas = elements.canvas;
    this.canvasContext = elements.canvasContext;
    this.patchCountElement = elements.patchCountElement;
    this.toggleDoorButton = elements.toggleDoorButton;
  }

  /**
   * Update patch count and UI
   */
  incrementPatchCount() {
    this.patchCount += 1;
    if (this.patchCountElement) {
      this.patchCountElement.textContent = String(this.patchCount);
    }
  }

  /**
   * Get current action mode
   */
  getCurrentActionMode() {
    return this.currentActionMode;
  }

  /**
   * Set current action mode
   * @param {string} mode
   */
  setCurrentActionMode(mode) {
    this.currentActionMode = mode;

    // Clear selections when switching modes
    if (mode === ACTION_MODES.ATTACK || mode === ACTION_MODES.SPELL) {
      this.selectedDoorId = null;
    } else {
      this.selectedMonsterId = null;
    }
  }

  /**
   * Get selected door ID
   */
  getSelectedDoorId() {
    return this.selectedDoorId;
  }

  /**
   * Set selected door ID
   * @param {string|null} doorId
   */
  setSelectedDoorId(doorId) {
    this.selectedDoorId = doorId;
  }

  /**
   * Get selected monster ID
   */
  getSelectedMonsterId() {
    return this.selectedMonsterId;
  }

  /**
   * Set selected monster ID
   * @param {string|null} monsterId
   */
  setSelectedMonsterId(monsterId) {
    this.selectedMonsterId = monsterId;
  }

  /**
   * Get monster by ID
   * @param {string} monsterId
   * @returns {MonsterLite|null}
   */
  getMonsterById(monsterId) {
    return this.monsters.find(m => m && m.id === monsterId) || null;
  }

  /**
   * Get threshold by ID
   * @param {string} thresholdId
   * @returns {ThresholdLite|null}
   */
  getThresholdById(thresholdId) {
    return this.thresholds.find(t => t.id === thresholdId) || null;
  }

  /**
   * Find threshold by position and orientation
   * @param {number} x
   * @param {number} y
   * @param {"vertical"|"horizontal"} orientation
   * @returns {ThresholdLite|null}
   */
  findThresholdByPosition(x, y, orientation) {
    return this.thresholds.find(d =>
      d.orientation === orientation && d.x === x && d.y === y,
    ) || null;
  }

  /**
   * Find monster at position
   * @param {number} x
   * @param {number} y
   * @returns {MonsterLite|null}
   */
  findMonsterAtPosition(x, y) {
    return this.monsters.find(m =>
      m && m.isVisible && m.isAlive &&
      m.tile && m.tile.x === x && m.tile.y === y,
    ) || null;
  }

  /**
   * Update entity position
   * @param {string} entityId
   * @param {TileAddress} newPosition
   */
  updateEntityPosition(entityId, newPosition) {
    this.entityPositions.set(entityId, newPosition);
  }

  /**
   * Get entity position
   * @param {string} entityId
   * @returns {TileAddress|null}
   */
  getEntityPosition(entityId) {
    return this.entityPositions.get(entityId) || null;
  }

  /**
   * Update threshold state
   * @param {string} thresholdId
   * @param {"open"|"closed"} state
   */
  updateThresholdState(thresholdId, state) {
    // console.log(`GAMESTATE-DOOR: Searching for threshold with ID ${thresholdId}`);
    // console.log(`GAMESTATE-DOOR: Available thresholds:`, this.thresholds.map(t => `${t.id}(${t.x},${t.y})`).join(', '));
    const idx = this.thresholds.findIndex(d => d.id === thresholdId);
    if (idx !== -1) {
      // console.log(`GAMESTATE-DOOR: Found threshold at index ${idx}, updating state from ${this.thresholds[idx].state} to ${state}`);
      this.thresholds[idx] = { ...this.thresholds[idx], state };
      // console.log(`GAMESTATE-DOOR: Updated threshold:`, this.thresholds[idx]);
    } else {
      // console.log(`GAMESTATE-DOOR: WARNING - Threshold with ID ${thresholdId} not found!`);
    }
  }

  /**
   * Add newly visible thresholds
   * @param {ThresholdLite[]} newThresholds
   */
  addVisibleThresholds(newThresholds) {
    for (const newThreshold of newThresholds) {
      const existingIndex = this.thresholds.findIndex(d => d.id === newThreshold.id);
      if (existingIndex === -1) {
        this.thresholds.push(newThreshold);
      } else {
        this.thresholds[existingIndex] = newThreshold;
      }
    }
  }

  /**
   * Add newly visible blocking walls
   * @param {BlockingWallLite[]} newWalls
   */
  addVisibleBlockingWalls(newWalls) {
    for (const newWall of newWalls) {
      const existingIndex = this.blockingWalls.findIndex(w => w.id === newWall.id);
      if (existingIndex === -1) {
        this.blockingWalls.push(newWall);
      } else {
        this.blockingWalls[existingIndex] = newWall;
      }
    }
  }

  /**
   * Add newly visible furniture
   * @param {FurnitureLite[]} newFurniture
   */
  addVisibleFurniture(newFurniture) {
    for (const item of newFurniture) {
      const existingIndex = this.furniture.findIndex(f => f.id === item.id);
      if (existingIndex === -1) {
        this.furniture.push(item);
      } else {
        this.furniture[existingIndex] = item;
      }
    }
  }

  /**
   * Add newly visible monsters
   * @param {MonsterLite[]} newMonsters
   */
  addVisibleMonsters(newMonsters) {
    for (const newMonster of newMonsters) {
      if (!newMonster || !newMonster.id || !newMonster.tile) {
        continue;
      }

      const existingIndex = this.monsters.findIndex(m => m && m.id === newMonster.id);
      if (existingIndex === -1) {
        this.monsters.push(newMonster);
      } else {
        this.monsters[existingIndex] = newMonster;
      }
    }
  }

  /**
   * Update monster data
   * @param {MonsterLite} updatedMonster
   */
  updateMonster(updatedMonster) {
    if (!updatedMonster || !updatedMonster.id) {
      return;
    }

    const existingIndex = this.monsters.findIndex(m => m && m.id === updatedMonster.id);
    if (existingIndex !== -1) {
      const existingMonster = this.monsters[existingIndex];

      // Preserve client-side visibility state - only update stats, not visibility
      // Visibility is managed separately via MonstersVisible patches and region revelation
      const preservedMonster = {
        ...updatedMonster,
        isVisible: existingMonster.isVisible,
        tile: existingMonster.tile || updatedMonster.tile
      };

      this.monsters[existingIndex] = preservedMonster;

      // Clear selection if selected monster died
      if (!updatedMonster.isAlive && this.selectedMonsterId === updatedMonster.id) {
        this.selectedMonsterId = null;
      }
    }
  }

  /**
   * Add regions to revealed set
   * @param {number[]} regionIds
   */
  addRevealedRegions(regionIds) {
    for (const id of regionIds) {
      this.revealedRegions.add(id);
    }
  }

  /**
   * Add regions to known set
   * @param {number[]} regionIds
   */
  addKnownRegions(regionIds) {
    for (const id of regionIds) {
      this.knownRegions.add(id);
    }
  }

  /**
   * Update visible regions
   * @param {number[]} regionIds
   */
  updateVisibleRegions(regionIds) {
    this.visibleNow = new Set(regionIds);
  }

  /**
   * Clean up invalid monsters from array
   */
  cleanupMonsters() {
    const validMonsters = this.monsters.filter(monster => monster && monster.tile);
    if (validMonsters.length !== this.monsters.length) {
      this.monsters.length = 0;
      this.monsters.push(...validMonsters);
    }
  }

  /**
   * Set WebSocket reference
   * @param {WebSocket} socket
   */
  setSocket(socket) {
    this.socketRef = socket;
  }

  /**
   * Get WebSocket reference
   * @returns {WebSocket|null}
   */
  getSocket() {
    return this.socketRef;
  }

  /**
   * Check if socket is ready
   * @returns {boolean}
   */
  isSocketReady() {
    return this.socketRef && this.socketRef.readyState === WebSocket.OPEN;
  }

  /**
   * Send message through WebSocket
   * @param {Object} message
   */
  sendMessage(message) {
    if (this.isSocketReady()) {
      this.socketRef.send(JSON.stringify(message));
      return true;
    }
    return false;
  }

  /**
   * Set redraw function reference
   * @param {Function} redrawFn
   */
  setRedrawFunction(redrawFn) {
    this.redrawRef = redrawFn;
  }

  /**
   * Get redraw function reference
   * @returns {Function|null}
   */
  getRedrawFunction() {
    return this.redrawRef;
  }

  /**
   * Trigger a redraw if function is available
   */
  requestRedraw() {
    if (this.redrawRef) {
      this.redrawRef();
    }
  }
}

// Create and export singleton instance
export const gameState = new GameState();