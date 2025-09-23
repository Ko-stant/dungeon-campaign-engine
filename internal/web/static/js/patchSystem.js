/**
 * Patch system - WebSocket patch handling and state updates
 */

import { gameState } from './gameState.js';
import { handleHeroActionResult } from './actionSystem.js';
import { updateMonsterDetailsUI } from './monsterSystem.js';

/**
 * Apply a patch received from the server
 * @param {Object} patch
 */
export function applyPatch(patch) {
  console.log('Parsed patch:', patch.type, patch);

  switch (patch.type) {
    case 'VariablesChanged':
      handleVariablesChanged(patch);
      break;

    case 'RegionsRevealed':
      handleRegionsRevealed(patch);
      break;

    case 'DoorStateChanged':
      handleDoorStateChanged(patch);
      break;

    case 'EntityUpdated':
      handleEntityUpdated(patch);
      break;

    case 'VisibleNow':
      handleVisibleNow(patch);
      break;

    case 'RegionsKnown':
      handleRegionsKnown(patch);
      break;

    case 'DoorsVisible':
      handleDoorsVisible(patch);
      break;

    case 'BlockingWallsVisible':
      handleBlockingWallsVisible(patch);
      break;

    case 'FurnitureVisible':
      handleFurnitureVisible(patch);
      break;

    case 'MonstersVisible':
      handleMonstersVisible(patch);
      break;

    case 'MonsterUpdate':
      handleMonsterUpdate(patch);
      break;

    case 'HeroActionResult':
      handleHeroActionResultPatch(patch);
      break;

    default:
      console.log('Unknown patch type:', patch.type);
  }
}

/**
 * Handle VariablesChanged patch
 * @param {Object} patch
 */
function handleVariablesChanged(patch) {
  if (patch.payload && patch.payload.entries) {
    gameState.incrementPatchCount();
  }
}

/**
 * Handle RegionsRevealed patch
 * @param {Object} patch
 */
function handleRegionsRevealed(patch) {
  if (patch.payload && Array.isArray(patch.payload.ids)) {
    gameState.addRevealedRegions(patch.payload.ids);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle DoorStateChanged patch
 * @param {Object} patch
 */
function handleDoorStateChanged(patch) {
  if (patch.payload) {
    const { thresholdId, state } = patch.payload;
    gameState.updateThresholdState(thresholdId, state);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle EntityUpdated patch
 * @param {Object} patch
 */
function handleEntityUpdated(patch) {
  if (patch.payload && patch.payload.tile) {
    const { id, tile } = patch.payload;
    console.log('DEBUG: EntityUpdated patch received for', id, 'new position:', tile,
      'entityPositions size before:', gameState.entityPositions.size);

    gameState.updateEntityPosition(id, tile);
    gameState.incrementPatchCount();

    console.log('DEBUG: entityPositions size after:', gameState.entityPositions.size,
      'contents:', Array.from(gameState.entityPositions.entries()));

    scheduleRedraw();
  }
}

/**
 * Handle VisibleNow patch
 * @param {Object} patch
 */
function handleVisibleNow(patch) {
  if (patch.payload && Array.isArray(patch.payload.ids)) {
    gameState.updateVisibleRegions(patch.payload.ids);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle RegionsKnown patch
 * @param {Object} patch
 */
function handleRegionsKnown(patch) {
  if (patch.payload && Array.isArray(patch.payload.ids)) {
    gameState.addKnownRegions(patch.payload.ids);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle DoorsVisible patch
 * @param {Object} patch
 */
function handleDoorsVisible(patch) {
  if (patch.payload && Array.isArray(patch.payload.doors)) {
    console.log('Received DoorsVisible patch:', patch.payload.doors.length, 'new doors');
    gameState.addVisibleThresholds(patch.payload.doors);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle BlockingWallsVisible patch
 * @param {Object} patch
 */
function handleBlockingWallsVisible(patch) {
  if (patch.payload && Array.isArray(patch.payload.blockingWalls)) {
    console.log('Received BlockingWallsVisible patch:', patch.payload.blockingWalls.length, 'new blocking walls');
    gameState.addVisibleBlockingWalls(patch.payload.blockingWalls);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle FurnitureVisible patch
 * @param {Object} patch
 */
function handleFurnitureVisible(patch) {
  if (patch.payload && Array.isArray(patch.payload.furniture)) {
    console.log('Received FurnitureVisible patch:', patch.payload.furniture.length, 'new furniture');
    gameState.addVisibleFurniture(patch.payload.furniture);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle MonstersVisible patch
 * @param {Object} patch
 */
function handleMonstersVisible(patch) {
  if (patch.payload && Array.isArray(patch.payload.monsters)) {
    console.log('Received MonstersVisible patch:', patch.payload.monsters.length, 'new monsters');
    gameState.addVisibleMonsters(patch.payload.monsters);
    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle MonsterUpdate patch
 * @param {Object} patch
 */
function handleMonsterUpdate(patch) {
  if (patch.payload && patch.payload.monster) {
    console.log('Received MonsterUpdate patch:', patch.payload.monster);
    gameState.updateMonster(patch.payload.monster);
    gameState.incrementPatchCount();
    updateMonsterDetailsUI();
    scheduleRedraw();
  }
}

/**
 * Handle HeroActionResult patch
 * @param {Object} patch
 */
function handleHeroActionResultPatch(patch) {
  if (patch.payload) {
    console.log('DEBUG: Received HeroActionResult:', patch.payload,
      'entityPositions size before:', gameState.entityPositions.size);

    handleHeroActionResult(patch.payload);
    gameState.incrementPatchCount();

    console.log('DEBUG: About to call scheduleRedraw() after HeroActionResult, entityPositions size:',
      gameState.entityPositions.size);

    scheduleRedraw();
  }
}

/**
 * WebSocket connection management
 */
let reconnectTimeout = null;

/**
 * Open WebSocket connection
 */
export function openWebSocket() {
  console.log('openStream() called');
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  const url = `${scheme}://${location.host}/stream`;
  console.log('Attempting WebSocket connection to:', url);

  const socket = new WebSocket(url);
  gameState.setSocket(socket);

  socket.onmessage = (event) => {
    try {
      console.log('Raw WebSocket message:', event.data);
      const patch = JSON.parse(event.data);
      applyPatch(patch);
    } catch (err) {
      console.error('Failed to parse WebSocket message:', err, event.data);
    }
  };

  socket.onclose = () => {
    console.log('WebSocket connection closed, will reconnect in 2 seconds');
    gameState.setSocket(null);

    // Clear any existing timeout
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }

    // Schedule reconnection
    reconnectTimeout = setTimeout(openWebSocket, 2000);
  };

  socket.onopen = () => {
    console.log('WebSocket connection opened');
    // Clear reconnect timeout if connection succeeds
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
  };

  socket.onerror = (error) => {
    console.error('WebSocket error:', error);
  };
}

/**
 * Close WebSocket connection
 */
export function closeWebSocket() {
  const socket = gameState.getSocket();
  if (socket) {
    socket.close();
    gameState.setSocket(null);
  }

  // Clear any pending reconnection
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
}

/**
 * Get WebSocket connection state
 * @returns {Object}
 */
export function getWebSocketState() {
  const socket = gameState.getSocket();
  return {
    connected: socket && socket.readyState === WebSocket.OPEN,
    readyState: socket ? socket.readyState : null,
    url: socket ? socket.url : null,
  };
}

/**
 * Schedule a redraw (will be overridden by main app)
 */
let scheduleRedraw = () => {
  console.warn('scheduleRedraw not set - main app should override this');
};

/**
 * Set the redraw function
 * @param {Function} fn
 */
export function setRedrawFunction(fn) {
  scheduleRedraw = fn;
}

/**
 * Manual patch processing for testing
 * @param {Object} patch
 */
export function processPatchForTesting(patch) {
  applyPatch(patch);
}