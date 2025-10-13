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

    case 'InstantActionResult':
      handleInstantActionResultPatch(patch);
      break;

    case 'TurnStateChanged':
      handleTurnStateChanged(patch);
      break;

    case 'MovementHistorySync':
      handleMovementHistorySync(patch);
      break;

    default:
      console.error('Unknown patch type:', patch.type);
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
  // console.log(`DOOR-PATCH: Received DoorStateChanged patch:`, patch);
  if (patch.payload) {
    const { thresholdId, state } = patch.payload;
    // console.log(`DOOR-PATCH: Updating threshold ${thresholdId} to state ${state}`);
    gameState.updateThresholdState(thresholdId, state);
    gameState.incrementPatchCount();

    // Recalculate movement range if in planning mode (door opening/closing affects accessible areas)
    import('./movementPlanning.js').then(movementModule => {
      movementModule.refreshMovementRange();
    });

    scheduleRedraw();
  } else {
    // console.log(`DOOR-PATCH: No payload in DoorStateChanged patch`);
  }
}

/**
 * Handle EntityUpdated patch
 * @param {Object} patch
 */
function handleEntityUpdated(patch) {
  if (patch.payload && patch.payload.tile) {
    const { id, tile } = patch.payload;

    // Get previous position before updating for movement cost calculation
    const previousPosition = gameState.getEntityPosition(id);

    gameState.updateEntityPosition(id, tile);
    gameState.incrementPatchCount();

    // Update player stats panel if hero entity updated
    if (gameState.playerStatsPanelController && id.startsWith('hero')) {
      gameState.playerStatsPanelController.updateFromSnapshot(gameState.snapshot);
    }

    // Update movement planning if hero moved
    if (id === 'hero-1') {
      import('./movementPlanning.js').then(module => {
        module.handleEntityMovement(id, tile, previousPosition);
      });

      // Update movement UI after manual movement
      Promise.all([
        import('./actionSystem.js'),
        import('./movementPlanning.js')
      ]).then(([actionModule, movementModule]) => {
        const movementState = movementModule.getMovementState();
        actionModule.updateMovementStatusUI(movementState);
      });
    }

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
    handleHeroActionResult(patch.payload);
    gameState.incrementPatchCount();

    scheduleRedraw();
  }
}

/**
 * Handle InstantActionResult patch (for movement dice, etc.)
 * @param {Object} patch
 */
function handleInstantActionResultPatch(patch) {
  if (patch.payload) {
    // Use the same handler for instant actions (like movement dice)
    handleHeroActionResult(patch.payload);
    gameState.incrementPatchCount();

    scheduleRedraw();
  }
}

/**
 * Handle TurnStateChanged patch
 * @param {Object} patch
 */
function handleTurnStateChanged(patch) {
  if (patch.payload) {
    // Update turn counter in UI using controller if available
    if (gameState.turnCounterController && gameState.snapshot) {
      // Update turn number in snapshot
      if (patch.payload.turnNumber !== undefined) {
        gameState.snapshot.turn = patch.payload.turnNumber;
      }
      // Refresh turn counter display
      gameState.turnCounterController.updateFromSnapshot(gameState.snapshot);
    } else {
      // Fallback to direct DOM update
      const turnCounter = document.getElementById('turnCounter');
      if (turnCounter && patch.payload.turnNumber !== undefined) {
        turnCounter.textContent = patch.payload.turnNumber;
      }
    }

    // Update player stats panel with new turn state
    if (gameState.playerStatsPanelController && gameState.snapshot) {
      gameState.playerStatsPanelController.updateFromSnapshot(gameState.snapshot);
    }

    // Handle movement clearing after actions
    if (patch.payload.movementLeft !== undefined && patch.payload.movementLeft === 0) {
      // Movement was cleared (likely due to taking an action), update movement planning state
      import('./movementPlanning.js').then(movementModule => {
        // Clear flood effect and end movement planning if active
        movementModule.turnMovementState.movementUsedThisTurn = movementModule.turnMovementState.maxMovementForTurn;

        // End movement planning if it's active
        const movementState = movementModule.getMovementState();
        if (movementState.isPlanning) {
          movementModule.endMovementPlanning();
        }

        // Update UI to reflect no movement remaining
        import('./actionSystem.js').then(actionModule => {
          actionModule.updateMovementStatusUI({
            isPlanning: false,
            usedMovement: movementModule.turnMovementState.maxMovementForTurn,
            maxMovement: movementModule.turnMovementState.maxMovementForTurn,
            availableMovement: 0,
            pathLength: 0,
            canExecute: false
          });
        });

        scheduleRedraw();
      });
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle MovementHistorySync patch
 * @param {Object} patch
 */
function handleMovementHistorySync(patch) {
  if (patch.payload) {
    // Sync movement history with server data
    import('./movementPlanning.js').then(movementModule => {
      movementModule.syncMovementHistoryFromServer(patch.payload);
    });

    gameState.incrementPatchCount();
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
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  const url = `${scheme}://${location.host}/stream`;

  const socket = new WebSocket(url);
  gameState.setSocket(socket);

  socket.onmessage = (event) => {
    try {
      const patch = JSON.parse(event.data);
      applyPatch(patch);
    } catch (err) {
      console.error('Failed to parse WebSocket message:', err, event.data);
    }
  };

  socket.onclose = () => {
    gameState.setSocket(null);

    // Clear any existing timeout
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }

    // Schedule reconnection
    reconnectTimeout = setTimeout(openWebSocket, 2000);
  };

  socket.onopen = () => {
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