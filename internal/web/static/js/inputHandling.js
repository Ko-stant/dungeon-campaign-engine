/**
 * Input handling - keyboard and mouse events
 */

import { keyToStep } from './geometry.js';
import { gameState } from './gameState.js';
import { selectDoorInDirection, toggleSelectedDoor } from './doorSystem.js';
import { selectMonsterInDirection, updateMonsterDetailsUI } from './monsterSystem.js';
import {
  getCurrentActionMode,
  toggleActionsMenu,
  selectActionByNumber,
  executeCurrentAction,
  updateActionButtons,
  rollMovementDice,
  rollAttackDice,
} from './actionSystem.js';
import { ACTION_MODES } from './types.js';
import { isMovementAllowed } from './movementPlanning.js';

/**
 * Select appropriate target based on current action mode
 * @param {number} dx
 * @param {number} dy
 */
export function selectTargetInDirection(dx, dy) {
  const mode = getCurrentActionMode();

  if (mode === ACTION_MODES.ATTACK || mode === ACTION_MODES.SPELL) {
    selectMonsterInDirection(dx, dy);
    updateMonsterDetailsUI();
    updateActionButtons();
  } else {
    selectDoorInDirection(dx, dy);
  }
}

/**
 * Send movement request to server
 * @param {number} dx
 * @param {number} dy
 * @returns {boolean} True if movement was sent
 */
export function requestMovement(dx, dy) {
  // Check if movement is allowed before sending
  if (!isMovementAllowed()) {
    // console.log('INPUT: Arrow key movement blocked - no movement available');
    return false;
  }

  if (gameState.isSocketReady()) {
    const msg = {
      type: 'MovementRequest',
      payload: {
        playerID: 'player-1',
        entityID: 'hero-1',
        action: 'move_before', // or 'move_after' depending on when in turn
        parameters: {
          dx: dx,
          dy: dy
        }
      }
    };

    // Note: Movement tracking is handled server-side with turn manager
    return gameState.sendMessage(msg);
  }
  return false;
}

/**
 * Handle keyboard input
 * @param {KeyboardEvent} event
 * @returns {boolean} True if event was handled
 */
export function handleKeyboardInput(event) {
  // Ignore keys when modifier keys are pressed (Ctrl, Alt, Meta/Cmd, Shift)
  if (event.ctrlKey || event.altKey || event.metaKey || event.shiftKey) {
    return false;
  }

  const step = keyToStep(event);

  if (step) {
    // Handle directional input
    selectTargetInDirection(step.dx, step.dy);

    if (getCurrentActionMode() === ACTION_MODES.MOVE) {
      requestMovement(step.dx, step.dy);
    }

    event.preventDefault();
    return true;
  }

  // Handle other key inputs
  switch (event.key.toLowerCase()) {
    case 'e':
      if (toggleSelectedDoor()) {
        event.preventDefault();
        return true;
      }
      break;

    case 'f':
      toggleActionsMenu();
      event.preventDefault();
      return true;

    case 'enter': {
      // Use specific action function for attack mode
      const mode = getCurrentActionMode();
      let executed = false;

      if (mode === ACTION_MODES.ATTACK) {
        executed = rollAttackDice();
      } else {
        executed = executeCurrentAction();
      }

      if (executed) {
        event.preventDefault();
        return true;
      }
      break;
    }

    case 'r':
      // Roll movement dice
      rollMovementDice();

      event.preventDefault();
      return true;

    case '1':
    case '2':
    case '3':
    case '4':
    case '5':
    case '6':
      selectActionByNumber(parseInt(event.key));
      event.preventDefault();
      return true;

    default:
      return false;
  }

  return false;
}

/**
 * Initialize input handling
 */
export function initializeInputHandling() {
  window.addEventListener('keydown', handleKeyboardInput);

  // Handle window resize
  window.addEventListener('resize', async () => {
    if (gameState.canvas && gameState.canvasContext) {
      const { resizeCanvas } = await import('./rendering.js');
      resizeCanvas(gameState.canvas, gameState.canvasContext);

      // Trigger redraw
      if (window.drawBoard) {
        window.drawBoard();
      }
    }
  });
}

/**
 * Clean up input handling
 */
export function cleanupInputHandling() {
  window.removeEventListener('keydown', handleKeyboardInput);
}

/**
 * Get input state for debugging
 * @returns {Object}
 */
export function getInputState() {
  return {
    currentActionMode: getCurrentActionMode(),
    selectedDoorId: gameState.getSelectedDoorId(),
    selectedMonsterId: gameState.getSelectedMonsterId(),
    socketReady: gameState.isSocketReady(),
  };
}

/**
 * Simulate key press (useful for testing)
 * @param {string} key
 * @param {Object} options
 */
export function simulateKeyPress(key, options = {}) {
  const event = new KeyboardEvent('keydown', {
    key: key,
    code: options.code || key,
    shiftKey: options.shift || false,
    ctrlKey: options.ctrl || false,
    altKey: options.alt || false,
    metaKey: options.meta || false,
    bubbles: true,
    cancelable: true,
  });

  return handleKeyboardInput(event);
}

/**
 * Check if a key combination is valid for current context
 * @param {string} key
 * @returns {boolean}
 */
export function isValidKeyForContext(key) {
  const mode = getCurrentActionMode();

  switch (key.toLowerCase()) {
    case 'e':
      return gameState.getSelectedDoorId() !== null;
    case 'enter':
      return mode === ACTION_MODES.ATTACK ? gameState.getSelectedMonsterId() !== null : true;
    case 'arrowup':
    case 'arrowdown':
    case 'arrowleft':
    case 'arrowright':
    case 'w':
    case 'a':
    case 's':
    case 'd':
      return true;
    case '1':
    case '2':
    case '3':
    case '4':
    case '5':
    case '6':
    case 'f':
      return true;
    default:
      return false;
  }
}

/**
 * Get help text for current context
 * @returns {string[]}
 */
export function getContextualHelp() {
  const mode = getCurrentActionMode();
  const help = [
    'Arrow keys or WASD: Move/Select targets',
    'F: Toggle actions menu',
    '1-6: Quick action selection',
  ];

  if (gameState.getSelectedDoorId()) {
    help.push('E: Toggle selected door');
  }

  if (mode === ACTION_MODES.ATTACK && gameState.getSelectedMonsterId()) {
    help.push('Enter: Execute attack');
  } else if (mode !== ACTION_MODES.MOVE) {
    help.push('Enter: Execute action');
  }

  return help;
}