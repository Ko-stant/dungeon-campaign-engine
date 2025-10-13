/**
 * Action system - action modes, dice rolling, combat execution
 */

import { ACTION_MODES, ACTION_NAMES } from './types.js';
import { gameState } from './gameState.js';
import { startMovementPlanning, endMovementPlanning, resetMovementPlan, executeMovementPlan, getMovementState, setMovementDiceRoll, getRemainingMovement, turnMovementState } from './movementPlanning.js';
import {
  isMonsterSelected,
  clearMonsterSelection,
  updateMonsterDetailsUI,
} from './monsterSystem.js';
import { clearDoorSelection } from './doorSystem.js';

/**
 * Set the current action mode
 * @param {string} mode
 */
export function setActionMode(mode) {
  gameState.setCurrentActionMode(mode);

  // Update UI indicators
  updateActionModeDisplay(mode);
  updateActionButtons();

  // Clear selections when switching modes
  if (mode === ACTION_MODES.ATTACK || mode === ACTION_MODES.SPELL) {
    clearDoorSelection();
  } else {
    clearMonsterSelection();
  }

  // Update UI displays
  updateMonsterDetailsUI();
}

/**
 * Get the current action mode
 * @returns {string}
 */
export function getCurrentActionMode() {
  return gameState.getCurrentActionMode();
}

/**
 * Update action mode display in UI
 * @param {string} mode
 */
function updateActionModeDisplay(mode) {
  const actionModeDisplay = document.getElementById('actionMode');
  if (actionModeDisplay) {
    actionModeDisplay.textContent = ACTION_NAMES[mode] || mode;
  }
}

/**
 * Toggle actions menu visibility
 */
export function toggleActionsMenu() {
  const menu = document.getElementById('actionsMenu');
  if (menu) {
    const isVisible = menu.style.display !== 'none';
    menu.style.display = isVisible ? 'none' : 'block';
  }
}

/**
 * Select action by number (1-6)
 * @param {number} actionNumber
 */
export function selectActionByNumber(actionNumber) {
  const actions = [
    ACTION_MODES.MOVE,
    ACTION_MODES.ATTACK,
    ACTION_MODES.SPELL,
    ACTION_MODES.SEARCH_TREASURE,
    ACTION_MODES.SEARCH_TRAPS,
    ACTION_MODES.SEARCH_HIDDEN_DOORS,
  ];

  if (actionNumber >= 1 && actionNumber <= actions.length) {
    setActionMode(actions[actionNumber - 1]);
  }
}

/**
 * Check if current action can be executed
 * @returns {boolean}
 */
export function canExecuteCurrentAction() {
  const mode = getCurrentActionMode();

  switch (mode) {
    case ACTION_MODES.MOVE:
      return true; // Movement is always available
    case ACTION_MODES.ATTACK:
    case ACTION_MODES.SPELL:
      return isMonsterSelected();
    case ACTION_MODES.SEARCH_TREASURE:
    case ACTION_MODES.SEARCH_TRAPS:
    case ACTION_MODES.SEARCH_HIDDEN_DOORS:
      return true; // Search actions don't need targets
    default:
      return false;
  }
}

/**
 * Execute the current action
 * @returns {boolean} True if action was sent
 */
export function executeCurrentAction() {
  const mode = getCurrentActionMode();

  if (!canExecuteCurrentAction() || !gameState.isSocketReady()) {
    return false;
  }

  const msg = createActionMessage(mode);
  if (msg) {
    gameState.sendMessage(msg);
    return true;
  }

  return false;
}

/**
 * Create action message for server
 * @param {string} mode
 * @returns {Object|null}
 */
function createActionMessage(mode) {
  const selectedMonsterId = gameState.getSelectedMonsterId();

  switch (mode) {
    case ACTION_MODES.ATTACK:
      if (selectedMonsterId) {
        return {
          type: 'HeroAction',
          payload: {
            playerId: 'player-1',
            entityId: 'hero-1',
            action: 'attack',
            parameters: { targetId: selectedMonsterId },
          },
        };
      }
      break;

    case ACTION_MODES.SPELL:
      if (selectedMonsterId) {
        return {
          type: 'HeroAction',
          payload: {
            playerId: 'player-1',
            entityId: 'hero-1',
            action: 'cast_spell',
            parameters: {
              spellId: 'fireball', // TODO: Add spell selection
              targetId: selectedMonsterId,
            },
          },
        };
      }
      break;

    case ACTION_MODES.SEARCH_TREASURE:
      return {
        type: 'HeroAction',
        payload: {
          playerId: 'player-1',
          entityId: 'hero-1',
          action: 'search_treasure',
          parameters: {},
        },
      };

    case ACTION_MODES.SEARCH_TRAPS:
      return {
        type: 'HeroAction',
        payload: {
          playerId: 'player-1',
          entityId: 'hero-1',
          action: 'search_traps',
          parameters: {},
        },
      };

    case ACTION_MODES.SEARCH_HIDDEN_DOORS:
      return {
        type: 'HeroAction',
        payload: {
          playerId: 'player-1',
          entityId: 'hero-1',
          action: 'search_hidden_doors',
          parameters: {},
        },
      };
  }

  return null;
}

/**
 * Roll movement dice (server-side processing)
 */
export function rollMovementDice() {
  const socket = gameState.getSocket();
  if (!socket) {
    console.error('WebSocket not connected');
    return;
  }

  const instantActionRequest = {
    playerID: 'player-1', // TODO: Get actual player ID
    entityID: 'hero-1',   // TODO: Get actual entity ID
    action: 'roll_movement',
    parameters: {}
  };

  // Send the instant action request
  socket.send(JSON.stringify({
    type: 'InstantActionRequest',
    payload: instantActionRequest
  }));
}

/**
 * Execute attack action (server-side dice rolling)
 */
export function rollAttackDice() {
  if (getCurrentActionMode() !== ACTION_MODES.ATTACK || !isMonsterSelected()) {
    return false;
  }

  const msg = {
    type: 'HeroAction',
    payload: {
      playerId: 'player-1', // TODO: Get actual player ID
      entityId: 'hero-1', // TODO: Get actual hero entity ID
      action: 'attack',
      parameters: {
        targetId: gameState.getSelectedMonsterId(),
      },
    },
  };

  if (gameState.sendMessage(msg)) {
    displayProcessingMessage('Processing attack...');
    disableActionButtons();
    return true;
  }

  return false;
}

/**
 * Defense dice info (handled automatically by server)
 */
export function rollDefenseDice() {
  displayProcessingMessage(
    'Defense is handled automatically by the server during attack.',
  );
}

/**
 * Display processing message (now just logs to console)
 * @param {string} message
 */
export function displayProcessingMessage(message) {
  console.log('Action:', message);
}

/**
 * Display dice roll results (deprecated - results now shown in detail pane)
 * @param {string} rollType
 * @param {Array} dice
 * @param {string} summary
 */
export function displayDiceResult(rollType, dice, summary = null) {
  console.log(`${rollType} Roll:`, dice, summary);
}

/**
 * Handle hero action result from server
 * @param {HeroActionResult} result
 */
export function handleHeroActionResult(result) {
  // Log result to console for debugging
  console.log('Action Result:', result.action, result.success ? 'SUCCESS' : 'FAILED', result.message);

  // Handle movement dice rolls
  if (result.movementRolls && result.movementRolls.length > 0) {
    let totalMovement = 0;
    const diceValues = result.movementRolls.map(roll => {
      totalMovement += roll.result;
      return roll.result;
    });

    // Display dice in the compact dice results area
    const diceDisplay = document.getElementById('diceResultsDisplay');
    if (diceDisplay) {
      diceDisplay.textContent = `[${diceValues.join(', ')}] = ${totalMovement}`;
    }

    // Set movement dice roll for turn tracking
    setMovementDiceRoll(totalMovement);

    // Start movement planning with the rolled movement points
    setTimeout(() => {
      try {
        startMovementPlanning();
        showMovementPlanningControls();
      } catch (error) {
        console.error('Error in movement planning:', error);
      }
    }, 100);
  }

  // Handle monster updates after attack
  if (result.action === 'attack' && gameState.getSelectedMonsterId()) {
    setTimeout(() => {
      updateMonsterDetailsUI();
    }, 100);

    if (result.damage > 0 && result.message && result.message.includes('killed')) {
      clearMonsterSelection();
    }
  }

  // Display action result in detail pane
  if (gameState.detailPaneController && result.action !== 'roll_movement') {
    gameState.detailPaneController.showActionResult(result);
  }

  // Reset button states
  updateActionButtons();
}

/**
 * Update action button states
 */
export function updateActionButtons() {
  const rollAttackBtn = document.getElementById('rollAttack');
  const rollDefenseBtn = document.getElementById('rollDefense');
  const executeBtn = document.getElementById('executeAction');

  if (rollAttackBtn) {
    rollAttackBtn.disabled =
      getCurrentActionMode() !== ACTION_MODES.ATTACK || !isMonsterSelected();
  }

  if (rollDefenseBtn) {
    rollDefenseBtn.disabled = true; // Always disabled as it's automatic
  }

  if (executeBtn) {
    executeBtn.disabled = !canExecuteCurrentAction();
  }
}

/**
 * Disable action buttons during processing
 */
function disableActionButtons() {
  const attackButton = document.getElementById('rollAttack');
  const defenseButton = document.getElementById('rollDefense');

  if (attackButton) {
    attackButton.disabled = true;
  }
  if (defenseButton) {
    defenseButton.disabled = true;
  }
}

/**
 * Initialize action system UI
 */
export function initializeActionUI() {
  // Action menu buttons
  document
    .getElementById('actionMove')
    ?.addEventListener('click', () => setActionMode(ACTION_MODES.MOVE));
  document
    .getElementById('actionAttack')
    ?.addEventListener('click', () => setActionMode(ACTION_MODES.ATTACK));
  document
    .getElementById('actionSpell')
    ?.addEventListener('click', () => setActionMode(ACTION_MODES.SPELL));
  document
    .getElementById('actionSearchTreasure')
    ?.addEventListener('click', () =>
      setActionMode(ACTION_MODES.SEARCH_TREASURE),
    );
  document
    .getElementById('actionSearchTraps')
    ?.addEventListener('click', () => setActionMode(ACTION_MODES.SEARCH_TRAPS));
  document
    .getElementById('actionSearchsearch_hidden_doors')
    ?.addEventListener('click', () =>
      setActionMode(ACTION_MODES.SEARCH_HIDDEN_DOORS),
    );

  // Dice rolling buttons
  document
    .getElementById('rollMovement')
    ?.addEventListener('click', rollMovementDice);
  document
    .getElementById('rollAttack')
    ?.addEventListener('click', rollAttackDice);
  document
    .getElementById('rollDefense')
    ?.addEventListener('click', rollDefenseDice);

  // Movement planning buttons
  document
    .getElementById('resetMovement')
    ?.addEventListener('click', resetMovementPlan);
  document
    .getElementById('executeMovement')
    ?.addEventListener('click', executeMovementPlan);
  document
    .getElementById('cancelMovement')
    ?.addEventListener('click', endMovementPlanning);

  // Execute action button
  document
    .getElementById('executeAction')
    ?.addEventListener('click', executeCurrentAction);

  // Pass turn button
  document
    .getElementById('passTurn')
    ?.addEventListener('click', passTurn);

  // Debug button for GM turn passing
  document
    .getElementById('passGMTurn')
    ?.addEventListener('click', passGMTurn);

  // Debug button for testing detail pane with card
  document
    .getElementById('testDetailPane')
    ?.addEventListener('click', testDetailPane);

  // Initialize default state
  setActionMode(ACTION_MODES.MOVE);
}

/**
 * Debug function to pass GM turn - skips all monster actions and returns to hero turn
 */
function passGMTurn() {
  // Send debug command to server
  const intent = {
    type: 'PassGMTurn',
    payload: {
      debug: true
    }
  };

  const envelope = {
    type: intent.type,
    payload: intent.payload
  };

  gameState.sendMessage(envelope);
}

/**
 * Debug function to test detail pane with a sample card
 */
function testDetailPane() {
  if (!gameState.detailPaneController) {
    console.error('DetailPaneController not available');
    return;
  }

  const testCard = {
    name: 'Longsword',
    description: 'This long blade gives you the attack strength of 3 combat dice. Because of its length, the longsword enables you to attack diagonally. May not be used by the wizard.',
    type: 'weapon',
    cardImage: '/assets/cards/equipment/longsword.jpg',
    stats: {
      attackDice: 3,
      attackDiagonal: true,
      cost: '350 gold coins'
    }
  };

  gameState.detailPaneController.showItem(testCard);
}

/**
 * Pass the current player's turn
 */
function passTurn() {
  const socket = gameState.getSocket();
  if (!socket) {
    console.error('WebSocket not connected');
    return;
  }

  const instantActionRequest = {
    playerID: 'player-1', // TODO: Get actual player ID
    entityID: 'hero-1',   // TODO: Get actual entity ID
    action: 'pass_turn',
    parameters: {}
  };

  // Send the instant action request
  socket.send(JSON.stringify({
    type: 'InstantActionRequest',
    payload: instantActionRequest
  }));

  // Clear any current movement planning
  if (window.endMovementPlanning) {
    endMovementPlanning();
  }
}

/**
 * Show movement planning controls and start updating UI
 */
export function showMovementPlanningControls() {
  const controlsDiv = document.getElementById('movementPlanningControls');
  if (controlsDiv) {
    controlsDiv.style.display = 'block';
    startMovementPlanningUI();
  }
}

/**
 * Hide movement planning controls
 */
function hideMovementPlanningControls() {
  const controlsDiv = document.getElementById('movementPlanningControls');
  if (controlsDiv) {
    controlsDiv.style.display = 'none';
  }
}

/**
 * Start updating movement planning UI
 */
function startMovementPlanningUI() {
  // Update UI every 100ms while planning is active
  const updateUI = () => {
    const movementState = getMovementState();
    if (!movementState.isPlanning) {
      hideMovementPlanningControls();
      return;
    }

    updateMovementStatusUI(movementState);
    setTimeout(updateUI, 100);
  };
  updateUI();
}

/**
 * Update movement status display
 */
export function updateMovementStatusUI(movementState) {
  const statusDiv = document.getElementById('movementStatus');
  const executeButton = document.getElementById('executeMovement');

  if (statusDiv) {
    // Use turn state for accurate movement tracking when available
    const usedMovement = movementState.isPlanning ? movementState.usedMovement : (turnMovementState.diceRolled ? turnMovementState.movementUsedThisTurn : 0);
    const maxMovement = movementState.isPlanning ? movementState.maxMovement : (turnMovementState.diceRolled ? turnMovementState.maxMovementForTurn : 0);
    const remainingMovement = movementState.isPlanning ? movementState.availableMovement : (turnMovementState.diceRolled ? getRemainingMovement() : 0);

    statusDiv.innerHTML = `
      <div>Movement: ${usedMovement}/${maxMovement}</div>
      <div>Remaining: ${remainingMovement}</div>
      <div>Path Steps: ${movementState.pathLength}</div>
    `;
  }

  if (executeButton) {
    executeButton.disabled = !movementState.canExecute;
  }
}
