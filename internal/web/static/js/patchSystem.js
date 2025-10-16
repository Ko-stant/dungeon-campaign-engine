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

    case 'LobbyStateChanged':
      // Ignore lobby messages on game page (shouldn't normally receive these)
      console.log('Ignoring LobbyStateChanged message (game page)');
      break;

    case 'PlayerIDAssigned':
      // Ignore player ID assignment on game page (already assigned)
      console.log('Ignoring PlayerIDAssigned message (game page)');
      break;

    case 'GameStarting':
      // Game is starting, might want to show loading screen
      console.log('Game starting:', patch.payload);
      break;

    case 'TurnPhaseChanged':
      handleTurnPhaseChanged(patch);
      break;

    case 'PlayerElected':
      handlePlayerElected(patch);
      break;

    case 'ElectionCancelled':
      handleElectionCancelled(patch);
      break;

    case 'HeroTurnStarted':
      handleHeroTurnStarted(patch);
      break;

    case 'HeroTurnCompleted':
      handleHeroTurnCompleted(patch);
      break;

    case 'GMTurnCompleted':
      handleGMTurnCompleted(patch);
      break;

    case 'MonsterTurnStateChanged':
      handleMonsterTurnStateChanged(patch);
      break;

    case 'StartingPositionSelected':
      handleStartingPositionSelected(patch);
      break;

    case 'AvailableStartingPositions':
      handleAvailableStartingPositions(patch);
      break;

    case 'QuestSetupStateChanged':
      handleQuestSetupStateChanged(patch);
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

      // Update hero turn states if heroID is present (new format)
      if (patch.payload.heroId && patch.payload.playerId) {
        // Ensure heroTurnStates map exists
        if (!gameState.snapshot.heroTurnStates) {
          gameState.snapshot.heroTurnStates = {};
        }

        // Update or create hero turn state
        gameState.snapshot.heroTurnStates[patch.payload.heroId] = {
          heroId: patch.payload.heroId,
          playerId: patch.payload.playerId,
          turnNumber: patch.payload.turnNumber,
          movementDiceRolled: patch.payload.movementDiceRolled || false,
          movementDiceResults: patch.payload.movementDiceResults || [],
          movementTotal: patch.payload.movementTotal || 0,
          movementUsed: patch.payload.movementUsed || 0,
          movementRemaining: patch.payload.movementLeft || 0,
          hasMoved: patch.payload.hasMoved || false,
          actionTaken: patch.payload.actionTaken || false,
          actionType: '',
          turnFlags: {},
          activitiesCount: 0,
          activeEffectsCount: 0,
          activeEffects: [],
          locationSearches: {},
          turnStartPosition: null,
          currentPosition: null
        };
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

/**
 * Handle TurnPhaseChanged patch
 * @param {Object} patch
 */
function handleTurnPhaseChanged(patch) {
  if (patch.payload) {
    // Update snapshot with new turn phase data
    if (gameState.snapshot) {
      gameState.snapshot.turnPhase = patch.payload.turnPhase;
      gameState.snapshot.cycleNumber = patch.payload.cycleNumber;
      gameState.snapshot.activeHeroPlayerID = patch.payload.activeHeroPlayerID || '';
      gameState.snapshot.electedPlayerID = patch.payload.electedPlayerID || '';
      gameState.snapshot.heroesActedIDs = patch.payload.heroesActedIDs || [];
    }

    // Update GM controls if present
    if (gameState.gmControlsController) {
      gameState.gmControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Update hero turn controls if present
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle PlayerElected patch
 * @param {Object} patch
 */
function handlePlayerElected(patch) {
  if (patch.payload) {
    // Update elected player ID in snapshot
    if (gameState.snapshot) {
      gameState.snapshot.electedPlayerID = patch.payload.playerID;
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      gameState.gmControlsController.logEvent(`Hero elected: ${patch.payload.playerName || patch.payload.playerID}`);
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle ElectionCancelled patch
 * @param {Object} patch
 */
function handleElectionCancelled(patch) {
  if (patch.payload) {
    // Clear elected player in snapshot
    if (gameState.snapshot) {
      gameState.snapshot.electedPlayerID = '';
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      gameState.gmControlsController.logEvent('Election cancelled');
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle HeroTurnStarted patch
 * @param {Object} patch
 */
function handleHeroTurnStarted(patch) {
  if (patch.payload) {
    // Update active hero player ID in snapshot
    if (gameState.snapshot) {
      gameState.snapshot.activeHeroPlayerID = patch.payload.playerID;
      gameState.snapshot.turnPhase = 'hero_active';
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      gameState.gmControlsController.logEvent(`Hero turn started: ${patch.payload.playerName || patch.payload.playerID}`);
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle HeroTurnCompleted patch
 * @param {Object} patch
 */
function handleHeroTurnCompleted(patch) {
  if (patch.payload) {
    // Add player to heroes acted list
    if (gameState.snapshot) {
      if (!gameState.snapshot.heroesActedIDs) {
        gameState.snapshot.heroesActedIDs = [];
      }
      if (!gameState.snapshot.heroesActedIDs.includes(patch.payload.playerID)) {
        gameState.snapshot.heroesActedIDs.push(patch.payload.playerID);
      }
      gameState.snapshot.activeHeroPlayerID = '';
      gameState.snapshot.turnPhase = 'hero_election';
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      gameState.gmControlsController.logEvent(`Hero turn completed: ${patch.payload.playerName || patch.payload.playerID}`);
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle GMTurnCompleted patch
 * @param {Object} patch
 */
function handleGMTurnCompleted(patch) {
  if (patch.payload) {
    // Update turn phase and cycle
    if (gameState.snapshot) {
      gameState.snapshot.turnPhase = 'hero_election';
      gameState.snapshot.cycleNumber = patch.payload.cycleNumber;
      gameState.snapshot.heroesActedIDs = [];
    }

    // Update GM controls
    if (gameState.gmControlsController) {
      gameState.gmControlsController.updateFromSnapshot(gameState.snapshot);
      gameState.gmControlsController.logEvent(`GM turn completed. Starting cycle ${patch.payload.cycleNumber}`);
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle MonsterTurnStateChanged patch
 * @param {Object} patch
 */
function handleMonsterTurnStateChanged(patch) {
  if (patch.payload && patch.payload.monsterID) {
    // Update monster in game state
    const monster = gameState.monsters.get(patch.payload.monsterID);
    if (monster) {
      monster.hasMoved = patch.payload.hasMoved;
      monster.actionTaken = patch.payload.actionTaken;
      monster.movement = patch.payload.movement;
    }

    // Update GM monster list if present
    if (gameState.gmControlsController) {
      gameState.gmControlsController.updateMonsterList();
      if (gameState.gmControlsController.selectedMonsterID === patch.payload.monsterID) {
        gameState.gmControlsController.updateSelectedMonsterPanel();
      }
    }

    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}

/**
 * Handle StartingPositionSelected patch
 * @param {Object} patch
 */
function handleStartingPositionSelected(patch) {
  if (patch.payload) {
    // Update player's starting position in snapshot
    if (gameState.snapshot && gameState.snapshot.players) {
      const player = gameState.snapshot.players.find(p => p.id === patch.payload.playerID);
      if (player) {
        player.startingPosition = patch.payload.position;
        player.hasSelectedPosition = true;
      }
    }

    // Update hero turn controls
    if (gameState.heroTurnControlsController) {
      gameState.heroTurnControlsController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      gameState.gmControlsController.logEvent(
        `Starting position selected: ${patch.payload.playerName || patch.payload.playerID} at (${patch.payload.position.x}, ${patch.payload.position.y})`
      );
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle AvailableStartingPositions patch
 * @param {Object} patch
 */
function handleAvailableStartingPositions(patch) {
  if (patch.payload && Array.isArray(patch.payload.positions)) {
    // Update available positions in hero turn controls
    if (gameState.heroTurnControlsController) {
      // This would be handled by the controller
      // For now, just store in game state
      gameState.availableStartingPositions = patch.payload.positions;
    }

    gameState.incrementPatchCount();
  }
}

/**
 * Handle QuestSetupStateChanged patch
 * @param {Object} patch
 */
function handleQuestSetupStateChanged(patch) {
  if (patch.payload) {
    console.log('QUEST-SETUP-PATCH: Received QuestSetupStateChanged', patch.payload);
    console.log('QUEST-SETUP-PATCH: My player ID:', gameState.snapshot?.viewerPlayerId);
    console.log('QUEST-SETUP-PATCH: Player start positions:', patch.payload.playerStartPositions);

    // Update quest setup state in snapshot
    if (gameState.snapshot) {
      gameState.snapshot.playersReady = patch.payload.playersReady || {};
      gameState.snapshot.playerStartPositions = patch.payload.playerStartPositions || {};
      console.log('QUEST-SETUP-PATCH: Updated snapshot positions:', gameState.snapshot.playerStartPositions);
    }

    // Update quest setup controller
    if (gameState.questSetupController) {
      gameState.questSetupController.updateFromSnapshot(gameState.snapshot);
    }

    // Log event if GM
    if (gameState.gmControlsController) {
      const readyCount = Object.values(patch.payload.playersReady || {}).filter(r => r).length;
      const totalPlayers = Object.keys(patch.payload.playersReady || {}).length;
      gameState.gmControlsController.logEvent(`Quest setup updated: ${readyCount}/${totalPlayers} players ready`);
    }

    gameState.incrementPatchCount();
    scheduleRedraw();
  }
}