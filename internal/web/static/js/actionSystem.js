/**
 * Action system - action modes, dice rolling, combat execution
 */

import { ACTION_MODES, ACTION_NAMES } from './types.js';
import { gameState } from './gameState.js';
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
  console.log('Setting action mode to:', mode);
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

  console.log('Execute action called:', {
    canExecute: canExecuteCurrentAction(),
    socketReady: gameState.isSocketReady(),
    currentActionMode: mode,
    selectedMonsterId: gameState.getSelectedMonsterId(),
  });

  if (!canExecuteCurrentAction() || !gameState.isSocketReady()) {
    return false;
  }

  const msg = createActionMessage(mode);
  if (msg) {
    gameState.sendMessage(msg);
    console.log('Sent action:', msg);
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
          entityId: 'hero-1',
          action: 'search_treasure',
          parameters: {},
        },
      };

    case ACTION_MODES.SEARCH_TRAPS:
      return {
        type: 'HeroAction',
        payload: {
          entityId: 'hero-1',
          action: 'search_traps',
          parameters: {},
        },
      };

    case ACTION_MODES.SEARCH_HIDDEN_DOORS:
      return {
        type: 'HeroAction',
        payload: {
          entityId: 'hero-1',
          action: 'search_hidden_doors',
          parameters: {},
        },
      };
  }

  return null;
}

/**
 * Roll movement dice (local, no server needed)
 */
export function rollMovementDice() {
  const roll1 = Math.floor(Math.random() * 6) + 1;
  const roll2 = Math.floor(Math.random() * 6) + 1;
  const total = roll1 + roll2;

  displayDiceResult(
    'Movement',
    [
      { die: 'movement', result: roll1 },
      { die: 'movement', result: roll2 },
    ],
    `Total: ${total} squares`,
  );
}

/**
 * Execute attack action (server-side dice rolling)
 */
export function rollAttackDice() {
  if (getCurrentActionMode() !== ACTION_MODES.ATTACK || !isMonsterSelected()) {
    console.log('Attack blocked:', {
      currentActionMode: getCurrentActionMode(),
      selectedMonsterId: gameState.getSelectedMonsterId(),
    });
    return false;
  }

  console.log('Executing attack against:', gameState.getSelectedMonsterId());

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
 * Display processing message
 * @param {string} message
 */
export function displayProcessingMessage(message) {
  const resultDiv = document.getElementById('diceResults');
  const contentDiv = document.getElementById('diceResultsContent');

  if (!resultDiv || !contentDiv) {
    return;
  }

  contentDiv.innerHTML = `
    <div class="mb-2">
      <strong>${message}</strong>
    </div>
    <div class="text-xs opacity-60">Waiting for server response...</div>
  `;
  resultDiv.style.display = 'block';
}

/**
 * Display dice roll results
 * @param {string} rollType
 * @param {Array} dice
 * @param {string} summary
 */
export function displayDiceResult(rollType, dice, summary = null) {
  const resultDiv = document.getElementById('diceResults');
  const contentDiv = document.getElementById('diceResultsContent');

  if (!resultDiv || !contentDiv) {
    return;
  }

  let resultHTML = `<div class="mb-2"><strong>${rollType} Roll:</strong></div>`;

  dice.forEach((die, index) => {
    let dieDisplay = `Die ${index + 1}: ${die.result}`;
    if (die.combatResult) {
      dieDisplay += ` (${die.combatResult})`;
    }
    resultHTML += `<div class="text-xs opacity-80">${dieDisplay}</div>`;
  });

  // Add summary if provided
  if (summary) {
    resultHTML += `<div class="mt-2 text-sm font-semibold">${summary}</div>`;
  }

  // Calculate combat summary
  if (rollType === 'Attack') {
    const skulls = dice.filter((d) => d.combatResult === 'skull').length;
    resultHTML += `<div class="mt-2 text-sm font-semibold">Skulls: ${skulls}</div>`;
  } else if (rollType === 'Defense') {
    const blackShields = dice.filter(
      (d) => d.combatResult === 'black_shield',
    ).length;
    const whiteShields = dice.filter(
      (d) => d.combatResult === 'white_shield',
    ).length;
    const totalShields = blackShields + whiteShields;
    resultHTML += `<div class="mt-2 text-sm font-semibold">Shields: ${totalShields} (${blackShields} black, ${whiteShields} white)</div>`;
  }

  contentDiv.innerHTML = resultHTML;
  resultDiv.style.display = 'block';
}

/**
 * Handle hero action result from server
 * @param {HeroActionResult} result
 */
export function handleHeroActionResult(result) {
  console.log('Processing hero action result:', result);

  const resultDiv = document.getElementById('diceResults');
  const contentDiv = document.getElementById('diceResultsContent');

  if (!resultDiv || !contentDiv) {
    return;
  }

  let resultHTML = `<div class="mb-2"><strong>${result.action} Result:</strong></div>`;

  // Show success/failure
  const statusColor = result.success ? 'text-green-300' : 'text-red-300';
  const message = result.message || result.error || 'Unknown error';
  resultHTML += `<div class="text-sm ${statusColor} mb-2">${result.success ? '✅' : '❌'
    } ${message}</div>`;

  // Show separated dice rolls if present (new format)
  if (result.attackRolls && result.attackRolls.length > 0) {
    resultHTML += '<div class="mb-2 text-sm font-semibold text-red-300">⚔️ Hero Attack Dice:</div>';
    result.attackRolls.forEach((roll, index) => {
      let rollDisplay = `Die ${index + 1}: ${roll.result}`;
      if (roll.combatResult) {
        const symbol = roll.combatResult === 'skull' ? '💀' : '🎯';
        rollDisplay += ` (${symbol} ${roll.combatResult})`;
      }
      resultHTML += `<div class="text-xs opacity-80 ml-2">${rollDisplay}</div>`;
    });
  }

  if (result.defenseRolls && result.defenseRolls.length > 0) {
    resultHTML += '<div class="mb-2 text-sm font-semibold text-blue-300">🛡️ Monster Defense Dice:</div>';
    result.defenseRolls.forEach((roll, index) => {
      let rollDisplay = `Die ${index + 1}: ${roll.result}`;
      if (roll.combatResult) {
        const symbol = roll.combatResult === 'black_shield' ? '⚫' :
                      roll.combatResult === 'white_shield' ? '⚪' : '❌';
        rollDisplay += ` (${symbol} ${roll.combatResult})`;
      }
      resultHTML += `<div class="text-xs opacity-80 ml-2">${rollDisplay}</div>`;
    });
  }

  // Show search dice rolls if present
  if (result.searchRolls && result.searchRolls.length > 0) {
    resultHTML += '<div class="mb-2 text-sm font-semibold text-yellow-300">🔍 Search Dice:</div>';
    result.searchRolls.forEach((roll, index) => {
      let rollDisplay = `Die ${index + 1}: ${roll.result}`;
      if (roll.combatResult) {
        rollDisplay += ` (${roll.combatResult})`;
      }
      resultHTML += `<div class="text-xs opacity-80 ml-2">${rollDisplay}</div>`;
    });
  }

  // Show combat summary for attack actions
  if (result.action === 'attack' && result.attackRolls && result.defenseRolls) {
    const skulls = result.attackRolls.filter(roll => roll.combatResult === 'skull').length;
    // For monster defense, only black shields count (HeroQuest rule)
    const blackShields = result.defenseRolls.filter(roll => roll.combatResult === 'black_shield').length;
    // const whiteShields = result.defenseRolls.filter(roll => roll.combatResult === 'white_shield').length;

    resultHTML += `<div class="mt-3 p-2 bg-gray-700 rounded text-center">`;
    resultHTML += `<div class="text-sm font-semibold">Combat Summary</div>`;
    resultHTML += `<div class="text-xs mt-1">💀 ${skulls} Skulls - ⚫ ${blackShields} Black Shields = ⚡ ${Math.max(0, skulls - blackShields)} Damage</div>`;
    resultHTML += `</div>`;
  }

  // Show damage if present
  if (result.damage !== undefined) {
    const damageColor = result.damage > 0 ? 'text-red-300' : 'text-green-300';
    resultHTML += `<div class="mt-2 text-sm font-semibold ${damageColor}">💥 Final Damage: ${result.damage}</div>`;
  }

  // Handle monster death
  if (result.action === 'attack' && gameState.getSelectedMonsterId()) {
    setTimeout(() => {
      updateMonsterDetailsUI();
    }, 100);

    if (result.damage > 0) {
      if (result.message.includes('killed')) {
        resultHTML +=
          '<div class="mt-2 text-sm text-red-300">💀 Monster defeated!</div>';
        clearMonsterSelection();
      }
    }
  }

  contentDiv.innerHTML = resultHTML;
  resultDiv.style.display = 'block';

  // Auto-clear error messages after 3 seconds
  if (!result.success) {
    setTimeout(() => {
      if (resultDiv.style.display === 'block') {
        resultDiv.style.display = 'none';
        contentDiv.innerHTML = '';
      }
    }, 3000);
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

  // Execute action button
  document
    .getElementById('executeAction')
    ?.addEventListener('click', executeCurrentAction);

  // Debug button for GM turn passing
  document
    .getElementById('passGMTurn')
    ?.addEventListener('click', passGMTurn);

  // Initialize default state
  setActionMode(ACTION_MODES.MOVE);
}

/**
 * Debug function to pass GM turn - skips all monster actions and returns to hero turn
 */
function passGMTurn() {
  console.log('DEBUG: Passing GM turn...');

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
