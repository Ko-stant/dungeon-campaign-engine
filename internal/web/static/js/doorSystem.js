/**
 * Door system - door selection and interaction
 */

import { calculateDoorEdgeFromDirection } from './geometry.js';
import { gameState } from './gameState.js';

/**
 * Find door ID from entity position and movement direction
 * @param {string} entityId
 * @param {number} dx
 * @param {number} dy
 * @returns {string|null}
 */
export function findAdjacentDoorIdFromDirection(entityId, dx, dy) {
  const pos = gameState.getEntityPosition(entityId);
  if (!pos) {
    return null;
  }

  const edge = calculateDoorEdgeFromDirection(pos, dx, dy);
  if (!edge) {
    return null;
  }

  const threshold = gameState.findThresholdByPosition(edge.x, edge.y, edge.orientation);
  return threshold ? threshold.id : null;
}

/**
 * Select door in the given direction from hero
 * @param {number} dx
 * @param {number} dy
 */
export function selectDoorInDirection(dx, dy) {
  const doorId = findAdjacentDoorIdFromDirection('hero-1', dx, dy);
  gameState.setSelectedDoorId(doorId);
}

/**
 * Toggle the currently selected door
 * @returns {boolean} True if door toggle was sent
 */
export function toggleSelectedDoor() {
  const selectedDoorId = gameState.getSelectedDoorId();

  if (selectedDoorId && gameState.isSocketReady()) {
    const msg = {
      type: 'RequestToggleDoor',
      payload: { thresholdId: selectedDoorId },
    };
    return gameState.sendMessage(msg);
  }

  return false;
}

/**
 * Get selected door information
 * @returns {ThresholdLite|null}
 */
export function getSelectedDoor() {
  const selectedDoorId = gameState.getSelectedDoorId();
  return selectedDoorId ? gameState.getThresholdById(selectedDoorId) : null;
}

/**
 * Check if a door is selected
 * @returns {boolean}
 */
export function isDoorSelected() {
  return gameState.getSelectedDoorId() !== null;
}

/**
 * Clear door selection
 */
export function clearDoorSelection() {
  gameState.setSelectedDoorId(null);
}

/**
 * Get all doors in the game
 * @returns {ThresholdLite[]}
 */
export function getAllDoors() {
  return gameState.thresholds;
}

/**
 * Check if door is open
 * @param {string} doorId
 * @returns {boolean}
 */
export function isDoorOpen(doorId) {
  const door = gameState.getThresholdById(doorId);
  return door ? door.state === 'open' : false;
}

/**
 * Check if door is closed
 * @param {string} doorId
 * @returns {boolean}
 */
export function isDoorClosed(doorId) {
  const door = gameState.getThresholdById(doorId);
  return door ? door.state === 'closed' : false;
}