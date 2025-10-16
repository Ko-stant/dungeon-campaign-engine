/**
 * Canvas click handler - detects entity clicks and opens inspection modal
 */

import { gameState } from './gameState.js';
import { getGridMetrics } from './rendering.js';

/**
 * Convert canvas pixel coordinates to tile coordinates
 * @param {number} canvasX - X coordinate in canvas pixels
 * @param {number} canvasY - Y coordinate in canvas pixels
 * @returns {Object|null} - {x, y} tile coordinates or null if outside grid
 */
function canvasToTile(canvasX, canvasY) {
  const m = getGridMetrics();

  // Calculate tile coordinates
  const tileX = Math.floor((canvasX - m.ox) / m.tile);
  const tileY = Math.floor((canvasY - m.oy) / m.tile);

  // Check if within grid bounds
  if (tileX < 0 || tileX >= m.cols || tileY < 0 || tileY >= m.rows) {
    return null;
  }

  return { x: tileX, y: tileY };
}

/**
 * Find hero at the given tile coordinates
 * @param {number} x - Tile x coordinate
 * @param {number} y - Tile y coordinate
 * @returns {Object|null} - Hero entity or null
 */
function findHeroAtTile(x, y) {
  const entities = gameState.snapshot?.entities || [];

  for (const entity of entities) {
    if (entity.kind === 'hero' && entity.tile && entity.tile.x === x && entity.tile.y === y) {
      return entity;
    }
  }

  return null;
}

/**
 * Find monster at the given tile coordinates
 * @param {number} x - Tile x coordinate
 * @param {number} y - Tile y coordinate
 * @returns {Object|null} - Monster or null
 */
function findMonsterAtTile(x, y) {
  const monsters = gameState.monsters || [];

  for (const monster of monsters) {
    if (monster.tile && monster.tile.x === x && monster.tile.y === y && monster.isVisible && monster.isAlive) {
      return monster;
    }
  }

  return null;
}

/**
 * Handle canvas click event
 * @param {MouseEvent} event
 */
function handleCanvasClick(event) {
  const canvas = gameState.canvas;
  if (!canvas) return;

  // Get canvas bounding rect for accurate coordinates
  const rect = canvas.getBoundingClientRect();
  const canvasX = event.clientX - rect.left;
  const canvasY = event.clientY - rect.top;

  // Convert to tile coordinates
  const tile = canvasToTile(canvasX, canvasY);
  if (!tile) return;

  // Check if we're in quest setup phase - allow selecting starting positions
  if (gameState.questSetupController) {
    const handled = gameState.questSetupController.handleTileClick(tile.x, tile.y);
    if (handled) {
      return; // Quest setup handled the click
    }
  }

  // Check if GM is in movement mode
  if (gameState.gmControlsController && gameState.gmControlsController.movementMode) {
    const handled = gameState.gmControlsController.handleMovementTileClick(tile.x, tile.y);
    if (handled) {
      return; // Movement handled the click
    }
  }

  // Check if GM is in attack mode
  if (gameState.gmControlsController && gameState.gmControlsController.attackMode) {
    const handled = gameState.gmControlsController.handleAttackTargetClick(tile.x, tile.y);
    if (handled) {
      return; // Attack handled the click
    }
  }

  // Check for entities at this tile (monsters first for better targeting)
  const monster = findMonsterAtTile(tile.x, tile.y);
  if (monster) {
    // If GM, select monster for control instead of showing modal
    if (gameState.gmControlsController && gameState.snapshot?.viewerRole === 'gm') {
      gameState.gmControlsController.selectMonster(monster.id);
      return;
    }

    // Otherwise, show monster details in modal (for heroes)
    if (gameState.entityModalController) {
      gameState.entityModalController.showMonster(monster);
    }
    return;
  }

  const hero = findHeroAtTile(tile.x, tile.y);
  if (hero) {
    // Show hero details in modal
    if (gameState.entityModalController) {
      gameState.entityModalController.showHero(hero);
    }
    return;
  }

  // No entity at this tile - could add furniture/door inspection later
}

/**
 * Initialize canvas click handling
 */
export function initializeCanvasClickHandling() {
  const canvas = gameState.canvas;
  if (!canvas) {
    console.error('Canvas not found for click handling');
    return;
  }

  canvas.addEventListener('click', handleCanvasClick);

  // Add pointer cursor when hovering over entities
  canvas.addEventListener('mousemove', handleCanvasHover);

  // Set default cursor
  canvas.style.cursor = 'default';
}

/**
 * Handle canvas hover to show pointer cursor over entities
 * @param {MouseEvent} event
 */
function handleCanvasHover(event) {
  const canvas = gameState.canvas;
  if (!canvas) return;

  const rect = canvas.getBoundingClientRect();
  const canvasX = event.clientX - rect.left;
  const canvasY = event.clientY - rect.top;

  const tile = canvasToTile(canvasX, canvasY);
  if (!tile) {
    canvas.style.cursor = 'default';
    return;
  }

  // Check if there's an entity at this tile
  const monster = findMonsterAtTile(tile.x, tile.y);
  const hero = findHeroAtTile(tile.x, tile.y);

  // Check if we're in quest setup and hovering over available position
  const isQuestSetup = gameState.snapshot?.turnPhase === 'quest_setup';
  const isAvailablePosition = isQuestSetup && gameState.questSetupController?.availablePositions?.some(
    pos => pos.x === tile.x && pos.y === tile.y
  );

  // Check if we're in GM movement mode and hovering over available movement tile
  const isMovementMode = gameState.gmControlsController?.movementMode;
  const isMovementTile = isMovementMode && gameState.gmControlsController?.availableMovementTiles?.some(
    t => t.x === tile.x && t.y === tile.y
  );

  // Check if we're in GM attack mode and hovering over available attack target
  const isAttackMode = gameState.gmControlsController?.attackMode;
  const isAttackTarget = isAttackMode && gameState.gmControlsController?.availableAttackTargets?.some(
    t => t.x === tile.x && t.y === tile.y
  );

  if (isAttackTarget) {
    canvas.style.cursor = 'crosshair';
  } else if (monster || hero || isAvailablePosition || isMovementTile) {
    canvas.style.cursor = 'pointer';
  } else {
    canvas.style.cursor = 'default';
  }
}

/**
 * Cleanup canvas click handling
 */
export function cleanupCanvasClickHandling() {
  const canvas = gameState.canvas;
  if (!canvas) return;

  canvas.removeEventListener('click', handleCanvasClick);
  canvas.removeEventListener('mousemove', handleCanvasHover);
}
