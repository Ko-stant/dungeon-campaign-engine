/**
 * Movement Planning System
 * Handles tactical movement with visual feedback, path planning, and drag-to-move
 */

import { gameState } from './gameState.js';
import { getGridMetrics, getTileRect } from './rendering.js';
import { updateMovementStatusUI } from './actionSystem.js';

/**
 * Movement planning state
 */
const movementState = {
  isPlanning: false,
  isDragging: false,
  isExecuting: false,
  startPosition: null,
  currentPosition: null,
  plannedPath: [],
  executionQueue: [], // Remaining steps to execute
  maxMovement: 0,
  usedMovement: 0,
  availableMovement: 0,
  dragStart: null,
  dragCurrent: null,
  validTiles: new Set(),
  pathPreview: [],
};

/**
 * Global movement tracking for the turn (tracks all movement regardless of how it was triggered)
 */
export const turnMovementState = {
  diceRolled: false,           // Whether movement dice have been rolled this turn
  maxMovementForTurn: 0,       // Total movement rolled for this turn
  movementUsedThisTurn: 0,     // Total movement consumed this turn (arrow keys + planned)
  initialHeroPosition: null,   // Hero position at start of turn for tracking
  movementHistory: [],         // Complete history of all movement this turn for replay/visualization
  currentSegment: null,        // Current movement segment being planned/executed
};

/**
 * Set movement dice roll for the turn
 * @param {number} maxMovement - Maximum movement points rolled
 */
export function setMovementDiceRoll(maxMovement) {
  turnMovementState.diceRolled = true;
  turnMovementState.maxMovementForTurn = maxMovement;
  turnMovementState.movementUsedThisTurn = 0;
  turnMovementState.initialHeroPosition = getHeroPosition();
  turnMovementState.movementHistory = [];
  turnMovementState.currentSegment = null;
  // console.log(`MOVEMENT-TRACKING: Dice rolled - ${maxMovement} movement available`);
}

/**
 * Reset turn movement state (called at end of turn)
 */
export function resetTurnMovement() {
  turnMovementState.diceRolled = false;
  turnMovementState.maxMovementForTurn = 0;
  turnMovementState.movementUsedThisTurn = 0;
  turnMovementState.initialHeroPosition = null;
  turnMovementState.movementHistory = [];
  turnMovementState.currentSegment = null;
  // console.log('MOVEMENT-TRACKING: Turn movement reset');
}

/**
 * Start tracking a new movement segment
 * @param {string} type - Type of movement: 'manual', 'planned'
 * @param {Object} startPosition - Starting position of this segment
 */
export function startMovementSegment(type, startPosition) {
  turnMovementState.currentSegment = {
    type: type,
    startPosition: { ...startPosition },
    path: [],
    startTime: Date.now(),
    executed: false
  };
}

/**
 * Add a movement step to the current segment
 * @param {Object} position - Position moved to
 */
export function addMovementStep(position) {
  if (turnMovementState.currentSegment) {
    turnMovementState.currentSegment.path.push({ ...position });
  }
}

/**
 * Complete the current movement segment and add to history
 */
export function completeMovementSegment() {
  if (turnMovementState.currentSegment) {
    turnMovementState.currentSegment.endTime = Date.now();
    turnMovementState.movementHistory.push({ ...turnMovementState.currentSegment });
    turnMovementState.currentSegment = null;
  }
}

/**
 * Mark a segment as executed (for planned movement)
 * @param {number} segmentIndex - Index of segment in history
 */
export function markSegmentExecuted(segmentIndex) {
  if (turnMovementState.movementHistory[segmentIndex]) {
    turnMovementState.movementHistory[segmentIndex].executed = true;
    turnMovementState.movementHistory[segmentIndex].executedTime = Date.now();
  }
}

/**
 * Get complete movement history for this turn
 * @returns {Array} Array of movement segments
 */
export function getMovementHistory() {
  return [...turnMovementState.movementHistory];
}

/**
 * Get current movement segment being planned/executed
 * @returns {Object|null} Current segment or null
 */
export function getCurrentMovementSegment() {
  return turnMovementState.currentSegment;
}

/**
 * Get the complete movement path for this turn (executed + planned)
 * @returns {Array} Array of positions showing full movement path
 */
export function getCompleteTurnMovementPath() {
  const completePath = [];

  // Add executed movement from history
  for (const segment of turnMovementState.movementHistory) {
    if (segment.executed) {
      completePath.push(...segment.path);
    }
  }

  // Add currently planned movement
  if (movementState.isPlanning && movementState.plannedPath.length > 0) {
    completePath.push(...movementState.plannedPath);
  }

  return completePath;
}

/**
 * Get movement visualization data for rendering
 * @returns {Object} Visualization data with segments and states
 */
export function getMovementVisualizationData() {
  return {
    history: turnMovementState.movementHistory,
    currentPlanned: movementState.isPlanning ? movementState.plannedPath : [],
    totalUsed: turnMovementState.movementUsedThisTurn,
    totalAvailable: turnMovementState.maxMovementForTurn,
    initialPosition: turnMovementState.initialHeroPosition,
    currentPosition: getHeroPosition()
  };
}

/**
 * Replay a specific movement segment
 * @param {number} segmentIndex - Index of segment to replay
 * @param {Function} stepCallback - Called for each step in the replay
 * @param {number} delay - Delay between steps in milliseconds
 */
export function replayMovementSegment(segmentIndex, stepCallback, delay = 500) {
  const segment = turnMovementState.movementHistory[segmentIndex];
  if (!segment) {
    console.error('Invalid segment index for replay:', segmentIndex);
    return;
  }

  let stepIndex = 0;
  const replayStep = () => {
    if (stepIndex < segment.path.length) {
      stepCallback(segment.path[stepIndex], stepIndex, segment);
      stepIndex++;
      setTimeout(replayStep, delay);
    }
  };

  // Start from segment start position
  stepCallback(segment.startPosition, -1, segment);
  setTimeout(replayStep, delay);
}

/**
 * Replay all movement for this turn
 * @param {Function} stepCallback - Called for each step
 * @param {number} delay - Delay between steps in milliseconds
 */
export function replayTurnMovement(stepCallback, delay = 500) {
  let segmentIndex = 0;

  const replayNextSegment = () => {
    if (segmentIndex < turnMovementState.movementHistory.length) {
      replayMovementSegment(segmentIndex, stepCallback, delay);
      segmentIndex++;
      // Wait for current segment to finish before starting next
      const segment = turnMovementState.movementHistory[segmentIndex - 1];
      setTimeout(replayNextSegment, (segment.path.length + 1) * delay);
    }
  };

  replayNextSegment();
}

/**
 * Sync movement history from server data
 * @param {Object} serverData - Movement history data from server
 */
export function syncMovementHistoryFromServer(serverData) {
  turnMovementState.movementHistory = serverData.history || [];
  turnMovementState.currentSegment = serverData.currentSegment || null;
  turnMovementState.initialHeroPosition = serverData.initialPosition || null;
  turnMovementState.maxMovementForTurn = serverData.movementLeft || 0;
  turnMovementState.diceRolled = serverData.movementDiceRolled || false;

  // Convert server timestamp strings back to Date objects for client use
  turnMovementState.movementHistory.forEach(segment => {
    if (segment.startTime) {
      segment.startTime = new Date(segment.startTime);
    }
    if (segment.endTime) {
      segment.endTime = new Date(segment.endTime);
    }
    if (segment.executedTime) {
      segment.executedTime = new Date(segment.executedTime);
    }
  });

  if (turnMovementState.currentSegment) {
    if (turnMovementState.currentSegment.startTime) {
      turnMovementState.currentSegment.startTime = new Date(turnMovementState.currentSegment.startTime);
    }
  }

  console.log('MOVEMENT-HISTORY: Synced from server:', turnMovementState.movementHistory);
}

/**
 * Convert movement history to server format for transmission
 * @returns {Object} Server-compatible movement history data
 */
export function prepareMovementHistoryForServer() {
  return {
    history: turnMovementState.movementHistory.map(segment => ({
      ...segment,
      startTime: segment.startTime ? segment.startTime.toISOString() : null,
      endTime: segment.endTime ? segment.endTime.toISOString() : null,
      executedTime: segment.executedTime ? segment.executedTime.toISOString() : null
    })),
    currentSegment: turnMovementState.currentSegment ? {
      ...turnMovementState.currentSegment,
      startTime: turnMovementState.currentSegment.startTime ? turnMovementState.currentSegment.startTime.toISOString() : null,
      endTime: turnMovementState.currentSegment.endTime ? turnMovementState.currentSegment.endTime.toISOString() : null,
      executedTime: turnMovementState.currentSegment.executedTime ? turnMovementState.currentSegment.executedTime.toISOString() : null
    } : null,
    initialPosition: turnMovementState.initialHeroPosition,
    movementLeft: turnMovementState.maxMovementForTurn,
    movementDiceRolled: turnMovementState.diceRolled
  };
}

/**
 * Check if movement is allowed (dice rolled and movement available)
 * @returns {boolean}
 */
export function isMovementAllowed() {
  if (!turnMovementState.diceRolled) {
    // console.log('MOVEMENT-TRACKING: Movement blocked - no dice rolled');
    return false;
  }
  if (turnMovementState.movementUsedThisTurn >= turnMovementState.maxMovementForTurn) {
    // console.log('MOVEMENT-TRACKING: Movement blocked - no movement remaining');
    return false;
  }
  return true;
}

/**
 * Track a movement step (used by both arrow keys and planned movement)
 */
export function trackMovementStep() {
  if (!turnMovementState.diceRolled) {
    console.error('MOVEMENT-TRACKING: Attempting to track movement without dice roll');
    return false;
  }

  if (turnMovementState.movementUsedThisTurn >= turnMovementState.maxMovementForTurn) {
    console.error('MOVEMENT-TRACKING: Attempting to move beyond available movement');
    return false;
  }

  turnMovementState.movementUsedThisTurn++;
  // console.log(`MOVEMENT-TRACKING: Movement used: ${turnMovementState.movementUsedThisTurn}/${turnMovementState.maxMovementForTurn}`);
  return true;
}

/**
 * Get remaining movement for this turn
 * @returns {number}
 */
export function getRemainingMovement() {
  if (!turnMovementState.diceRolled) return 0;
  return Math.max(0, turnMovementState.maxMovementForTurn - turnMovementState.movementUsedThisTurn);
}

/**
 * Initialize movement planning system
 */
export function initMovementPlanning() {
  const canvas = gameState.canvas;
  if (!canvas) return;

  // Add mouse event listeners for drag-to-move
  canvas.addEventListener('mousedown', handleMouseDown);
  canvas.addEventListener('mousemove', handleMouseMove);
  canvas.addEventListener('mouseup', handleMouseUp);
  canvas.addEventListener('mouseleave', handleMouseLeave);

  // Movement planning system initialized
}

/**
 * Start movement planning mode
 * Uses remaining movement for this turn from turnMovementState
 */
export function startMovementPlanning() {
  const heroPos = getHeroPosition();
  if (!heroPos) {
    console.error('Cannot start movement planning: hero position not found');
    return;
  }

  // Complete any active manual movement segment before starting planning
  if (turnMovementState.currentSegment && turnMovementState.currentSegment.type === 'manual') {
    completeMovementSegment();
  }

  // Use remaining movement for this turn, not the passed maxMovement
  const remainingMovement = getRemainingMovement();
  if (remainingMovement <= 0) {
    // console.log('MOVEMENT-PLANNING: No movement remaining for this turn');
    return;
  }

  movementState.isPlanning = true;
  movementState.isExecuting = false;
  movementState.startPosition = { ...heroPos };
  movementState.currentPosition = { ...heroPos };
  // Initialize movement state based on current turn state
  movementState.maxMovement = turnMovementState.maxMovementForTurn;
  movementState.usedMovement = turnMovementState.movementUsedThisTurn;
  movementState.availableMovement = remainingMovement;
  movementState.plannedPath = [];
  movementState.pathPreview = [];

  // Calculate valid movement tiles based on remaining movement
  calculateValidMovementTiles(heroPos, remainingMovement);

  // console.log(`MOVEMENT-PLANNING: Started with ${remainingMovement} remaining movement`);
  gameState.requestRedraw();
}

/**
 * End movement planning mode
 */
export function endMovementPlanning() {
  movementState.isPlanning = false;
  movementState.isDragging = false;
  movementState.isExecuting = false;
  movementState.plannedPath = [];
  movementState.pathPreview = [];
  movementState.executionQueue = [];
  movementState.validTiles.clear();

  // If there's still movement available, recalculate flood effect from actual hero position
  if (turnMovementState.diceRolled) {
    const remainingMovement = getRemainingMovement();
    if (remainingMovement > 0) {
      const heroPos = gameState.getEntityPosition('hero-1') || movementState.startPosition;
      calculateValidMovementTiles(heroPos, remainingMovement);
    }
  }

  gameState.requestRedraw();
}

/**
 * Reset planned movement to starting position
 */
export function resetMovementPlan() {
  if (!movementState.isPlanning) return;

  movementState.currentPosition = { ...movementState.startPosition };
  movementState.plannedPath = [];
  movementState.pathPreview = [];
  movementState.usedMovement = 0;
  movementState.availableMovement = movementState.maxMovement;

  // Recalculate valid tiles from starting position
  calculateValidMovementTiles(movementState.startPosition, movementState.maxMovement);

  gameState.requestRedraw();
}

/**
 * Execute the planned movement
 */
export function executeMovementPlan() {
  // console.log('MOVEMENT-EXEC: executeMovementPlan() called');
  // console.log('MOVEMENT-EXEC: isPlanning:', movementState.isPlanning);
  // console.log('MOVEMENT-EXEC: plannedPath.length:', movementState.plannedPath.length);

  if (!movementState.isPlanning || movementState.plannedPath.length === 0) {
    // console.log('MOVEMENT-EXEC: Cannot execute - not planning or no path');
    return false;
  }

  // Store the planned path as execution queue
  movementState.executionQueue = [...movementState.plannedPath];
  // console.log('MOVEMENT-EXEC: Stored execution queue:', movementState.executionQueue);
  // console.log('MOVEMENT-EXEC: Execution queue details:');
  // movementState.executionQueue.forEach((pos, i) => {
  //   console.log(`  Step ${i}: (${pos.x}, ${pos.y})`);
  // });

  // Mark all planned segments as executed in history
  for (let i = 0; i < turnMovementState.movementHistory.length; i++) {
    const segment = turnMovementState.movementHistory[i];
    if (segment.type === 'planned' && !segment.executed) {
      markSegmentExecuted(i);
    }
  }

  // Clear the planned path since we're executing it
  // but keep planning mode active to handle partial movement
  movementState.plannedPath = [];
  movementState.pathPreview = [];

  // Mark that we're executing movement
  movementState.isExecuting = true;

  // Send first movement step
  // console.log('MOVEMENT-EXEC: About to send first movement step');
  sendNextMovementStep();

  // Don't end planning mode yet - let handleEntityMovement manage that
  // based on whether there's remaining movement
  gameState.requestRedraw();
  return true;
}

/**
 * Get current movement state for UI display
 */
export function getMovementState() {
  return {
    isPlanning: movementState.isPlanning,
    isExecuting: movementState.isExecuting,
    usedMovement: movementState.usedMovement,
    availableMovement: movementState.availableMovement,
    maxMovement: movementState.maxMovement,
    pathLength: movementState.plannedPath.length,
    executionQueueLength: movementState.executionQueue.length,
    canExecute: movementState.plannedPath.length > 0 && !movementState.isExecuting,
  };
}

/**
 * Draw movement planning visuals
 * @param {CanvasRenderingContext2D} ctx
 */
export function drawMovementPlanning(ctx) {
  if (!movementState.isPlanning) {
    return;
  }

  const metrics = getGridMetrics();

  // Draw valid movement range
  drawValidMovementRange(ctx, metrics);

  // Draw planned path
  drawPlannedPath(ctx, metrics);

  // Draw path preview (during drag)
  drawPathPreview(ctx, metrics);

  // Draw movement indicators
  drawMovementIndicators(ctx, metrics);
}

/**
 * Handle mouse down for drag-to-move
 */
function handleMouseDown(event) {
  // console.log(`DRAG-DEBUG: MouseDown - isPlanning: ${movementState.isPlanning}, isDragging: ${movementState.isDragging}, isExecuting: ${movementState.isExecuting}`);

  if (!movementState.isPlanning) {
    // console.log('DRAG-DEBUG: Not planning, ignoring mousedown');
    return;
  }

  const rect = gameState.canvas.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  const tile = screenToTile(x, y);
  if (!tile) {
    return;
  }

  // For drag operations, use current planning position if already planning, otherwise actual hero position
  const actualHeroPos = gameState.getEntityPosition('hero-1') || movementState.startPosition;

  // For subsequent drags during planning, use the current planning position
  // For fresh planning, use actual hero position
  const heroPos = movementState.isPlanning && movementState.currentPosition
    ? movementState.currentPosition
    : actualHeroPos;

  // console.log(`DRAG-START-DEBUG: Planning: ${movementState.isPlanning}, CurrentPos: (${movementState.currentPosition?.x || 'none'}, ${movementState.currentPosition?.y || 'none'}), Actual: (${actualHeroPos.x}, ${actualHeroPos.y}), Using: (${heroPos.x}, ${heroPos.y})`);

  if (tile.x === heroPos.x && tile.y === heroPos.y) {
    // console.log('DRAG-DEBUG: Clicked on hero position, starting drag');
    movementState.isDragging = true;
    movementState.dragStart = { ...tile };
    movementState.dragCurrent = { ...tile };
    movementState.pathPreview = [];

    event.preventDefault();
  } else {
    // Allow clicking on any valid movement tile to start dragging from current position
    const key = `${tile.x},${tile.y}`;
    // console.log(`DRAG-DEBUG: Checking if tile (${tile.x}, ${tile.y}) is valid for movement`);
    if (movementState.validTiles.has(key)) {
      // console.log('DRAG-DEBUG: Valid tile clicked, starting drag');
      movementState.isDragging = true;
      movementState.dragStart = { ...heroPos };
      movementState.dragCurrent = { ...tile };

      // Calculate immediate path preview
      const path = calculatePath(heroPos, tile);
      if (path && path.length <= movementState.availableMovement) {
        movementState.pathPreview = path;
      } else {
        movementState.pathPreview = [];
      }

      event.preventDefault();
    }
  }
}

/**
 * Handle mouse move for path preview
 */
function handleMouseMove(event) {
  if (!movementState.isPlanning || !movementState.isDragging) return;

  const rect = gameState.canvas.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  const tile = screenToTile(x, y);
  if (!tile) return;

  // Update drag current position
  movementState.dragCurrent = { ...tile };

  // Calculate path preview from drag start position
  const startPos = movementState.dragStart || movementState.currentPosition;
  const path = calculatePath(startPos, tile);

  if (path && path.length <= movementState.availableMovement) {
    movementState.pathPreview = path;
  } else {
    movementState.pathPreview = [];
  }

  gameState.requestRedraw();
}

/**
 * Handle mouse up to confirm movement
 */
function handleMouseUp() {
  if (!movementState.isPlanning || !movementState.isDragging) return;

  movementState.isDragging = false;

  // Apply the path preview if valid
  if (movementState.pathPreview.length > 0) {
    applyPathToPlanning(movementState.pathPreview);
  }

  movementState.pathPreview = [];
  gameState.requestRedraw();
}

/**
 * Handle mouse leaving canvas
 */
function handleMouseLeave() {
  if (movementState.isDragging) {
    movementState.isDragging = false;
    movementState.pathPreview = [];
    gameState.requestRedraw();
  }
}

/**
 * Calculate valid movement tiles within range and line of sight
 */
function calculateValidMovementTiles(startPos, maxMovement) {
  // console.log(`FLOOD-FILL: Starting from (${startPos.x},${startPos.y}) with ${maxMovement} movement`);

  movementState.validTiles.clear();

  // Flood fill for movement range, respecting walls and doors
  const visited = new Set();
  const queue = [{ pos: startPos, distance: 0 }];

  while (queue.length > 0) {
    const { pos, distance } = queue.shift();
    const key = `${pos.x},${pos.y}`;

    if (visited.has(key) || distance > maxMovement) continue;
    visited.add(key);

    // Check if current tile is valid for movement (not blocked by monsters/furniture)
    const validMovement = isValidMovementTile(pos);

    // Add to valid movement tiles if this tile is valid
    if (validMovement) {
      movementState.validTiles.add(key);
      // console.log(`FLOOD-FILL: Added valid tile (${pos.x},${pos.y}) to movement range`);
    }

    // Only explore from tiles that are valid (not blocked by monsters/furniture)
    // This prevents flood fill from "passing through" furniture
    if (validMovement) {
      // Add adjacent tiles for continued exploration
      const adjacent = [
        { x: pos.x + 1, y: pos.y },
        { x: pos.x - 1, y: pos.y },
        { x: pos.x, y: pos.y + 1 },
        { x: pos.x, y: pos.y - 1 },
      ];

      for (const nextPos of adjacent) {
        const nextKey = `${nextPos.x},${nextPos.y}`;

        // Skip if already processed
        if (visited.has(nextKey)) {
          continue;
        }

        // Check if movement from current position to next position is blocked by walls/doors
        if (isMovementBlocked(pos, nextPos)) {
          // console.log(`FLOOD-FILL: Blocked movement from (${pos.x},${pos.y}) to (${nextPos.x},${nextPos.y})`);
          continue;
        }

        // Check if next tile is within bounds and not blocked by furniture/monsters
        // before adding to queue - this prevents exploring from blocked tiles
        if (isValidMovementTile(nextPos)) {
          // console.log(`FLOOD-FILL: Adding (${nextPos.x},${nextPos.y}) to queue, distance ${distance + 1}`);
          queue.push({ pos: nextPos, distance: distance + 1 });
        } else {
          // console.log(`FLOOD-FILL: Invalid tile (${nextPos.x},${nextPos.y}) - not adding to queue`);
        }
      }
    }
  }
}

/**
 * Check if a tile is within line of sight (in revealed regions)
 */

/**
 * Calculate path between two points
 */
function calculatePath(start, end) {
  // Simple Manhattan distance pathfinding
  // console.log(`PATH-CALC: Calculating path from (${start.x}, ${start.y}) to (${end.x}, ${end.y})`);
  const path = [];
  const current = { ...start };

  while (current.x !== end.x || current.y !== end.y) {
    // Move towards target (Manhattan style)
    if (current.x < end.x) {
      current.x++;
    } else if (current.x > end.x) {
      current.x--;
    } else if (current.y < end.y) {
      current.y++;
    } else if (current.y > end.y) {
      current.y--;
    }

    if (!isValidMovementTile(current)) {
      return null; // Invalid path
    }

    path.push({ ...current });

    // Prevent infinite loops
    if (path.length > 20) break;
  }

  // console.log(`PATH-CALC: Generated path:`, path);
  return path;
}

/**
 * Apply calculated path to movement planning
 */
function applyPathToPlanning(path) {
  // For multi-segment planning, append to existing path instead of replacing
  const newTotalPath = [...movementState.plannedPath, ...path];
  const totalMovementAfterPath = turnMovementState.movementUsedThisTurn + newTotalPath.length;

  if (totalMovementAfterPath > movementState.maxMovement) {
    console.log(`PLANNING: Path rejected - would use ${totalMovementAfterPath}/${movementState.maxMovement} movement`);
    return;
  }

  // Track this planned segment in movement history
  if (path.length > 0) {
    const startPos = movementState.currentPosition || movementState.startPosition;
    startMovementSegment('planned', startPos);

    // Add all path steps to the current segment
    for (const step of path) {
      addMovementStep(step);
    }

    completeMovementSegment();
  }

  movementState.plannedPath = newTotalPath;
  movementState.currentPosition = path[path.length - 1] || movementState.startPosition;

  // Update planning movement tracking
  // Note: Don't update turnMovementState here - that happens during execution
  movementState.usedMovement = totalMovementAfterPath;
  movementState.availableMovement = movementState.maxMovement - movementState.usedMovement;

  // Recalculate valid tiles based on new position and remaining movement
  console.log(`FLOOD-DEBUG: Recalculating from position (${movementState.currentPosition.x}, ${movementState.currentPosition.y}) with ${movementState.availableMovement} movement`);
  recalculateValidMovementTiles();

  // Path applied successfully
}

/**
 * Recalculate valid movement tiles from current position with remaining movement
 */
function recalculateValidMovementTiles() {
  if (!movementState.isPlanning) return;

  // Calculate from current position with available movement
  calculateValidMovementTiles(movementState.currentPosition, movementState.availableMovement);
}

/**
 * Refresh movement range (called when doors open/close or game state changes)
 */
export function refreshMovementRange() {
  if (!movementState.isPlanning) {
    return;
  }

  // Recalculate flood effect from current position with available movement
  const startPos = movementState.currentPosition || gameState.getEntityPosition('hero-1') || movementState.startPosition;
  const remainingMovement = getRemainingMovement();

  if (remainingMovement > 0) {
    calculateValidMovementTiles(startPos, remainingMovement);
    gameState.requestRedraw();
  }
}

/**
 * Draw valid movement range overlay
 */
function drawValidMovementRange(ctx, metrics) {
  ctx.save();
  ctx.fillStyle = 'rgba(0, 150, 255, 0.2)';

  for (const tileKey of movementState.validTiles) {
    const [x, y] = tileKey.split(',').map(Number);
    const rect = getTileRect(x, y, metrics);
    ctx.fillRect(rect.x, rect.y, rect.w, rect.h);
  }

  ctx.restore();
}

/**
 * Draw planned movement path
 */
function drawPlannedPath(ctx, metrics) {
  if (movementState.plannedPath.length === 0) return;

  ctx.save();
  ctx.strokeStyle = 'rgba(0, 255, 0, 0.8)';
  ctx.lineWidth = 4;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  // Draw path line
  ctx.beginPath();
  const startRect = getTileRect(movementState.startPosition.x, movementState.startPosition.y, metrics);
  ctx.moveTo(startRect.x + startRect.w/2, startRect.y + startRect.h/2);

  for (const step of movementState.plannedPath) {
    const rect = getTileRect(step.x, step.y, metrics);
    ctx.lineTo(rect.x + rect.w/2, rect.y + rect.h/2);
  }
  ctx.stroke();

  // Draw step indicators
  ctx.fillStyle = 'rgba(0, 255, 0, 0.6)';
  for (let i = 0; i < movementState.plannedPath.length; i++) {
    const step = movementState.plannedPath[i];
    const rect = getTileRect(step.x, step.y, metrics);

    // Draw step number
    ctx.save();
    ctx.fillStyle = 'white';
    ctx.font = '12px sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(
      (i + 1).toString(),
      rect.x + rect.w/2,
      rect.y + rect.h/2
    );
    ctx.restore();
  }

  ctx.restore();
}

/**
 * Draw path preview during drag
 */
function drawPathPreview(ctx, metrics) {
  if (movementState.pathPreview.length === 0) return;

  ctx.save();
  ctx.strokeStyle = 'rgba(255, 255, 0, 0.6)';
  ctx.lineWidth = 3;
  ctx.setLineDash([5, 5]);

  ctx.beginPath();
  const startRect = getTileRect(movementState.currentPosition.x, movementState.currentPosition.y, metrics);
  ctx.moveTo(startRect.x + startRect.w/2, startRect.y + startRect.h/2);

  for (const step of movementState.pathPreview) {
    const rect = getTileRect(step.x, step.y, metrics);
    ctx.lineTo(rect.x + rect.w/2, rect.y + rect.h/2);
  }
  ctx.stroke();

  ctx.restore();
}

/**
 * Draw movement indicators and text
 */
function drawMovementIndicators() {
  // This could show movement remaining, etc.
  // For now, we'll handle this in the UI panel
}

/**
 * Helper functions
 */
function getHeroPosition() {
  return gameState.entityPositions.get('hero-1') || null;
}

function screenToTile(screenX, screenY) {
  const metrics = getGridMetrics();
  const x = Math.floor((screenX - metrics.ox) / metrics.tile);
  const y = Math.floor((screenY - metrics.oy) / metrics.tile);

  // Validate bounds
  const snapshot = gameState.snapshot;
  if (x < 0 || y < 0 || x >= (snapshot?.mapWidth ?? 26) || y >= (snapshot?.mapHeight ?? 19)) {
    return null;
  }

  return { x, y };
}

function isValidMovementTile(pos) {
  // Basic bounds validation
  const snapshot = gameState.snapshot;
  if (pos.x < 0 || pos.y < 0 || pos.x >= (snapshot?.mapWidth ?? 26) || pos.y >= (snapshot?.mapHeight ?? 19)) {
    return false;
  }

  // Check if tile is in a revealed region (player can only move to areas they can see)
  const tileRegionId = getTileRegionId(pos.x, pos.y);
  if (tileRegionId !== null && !gameState.revealedRegions.has(tileRegionId)) {
    // console.log(`FLOOD-FILL: Tile (${pos.x},${pos.y}) blocked - region ${tileRegionId} not revealed`);
    return false;
  }

  // Note: Wall and door blocking is now checked during pathfinding using edge-based collision detection
  // This function only checks tile-specific blocking (monsters, furniture)

  // Check for monsters
  if (isMonsterBlocking(pos)) {
    return false;
  }

  // Check for furniture that blocks movement
  if (isFurnitureBlocking(pos)) {
    return false;
  }

  return true;
}

/**
 * Check if movement from a position to an adjacent position is blocked by walls or doors
 */
function isMovementBlocked(fromPos, toPos) {
  const dx = toPos.x - fromPos.x;
  const dy = toPos.y - fromPos.y;

  // Only check adjacent movement (one step)
  if (Math.abs(dx) + Math.abs(dy) !== 1) {
    return false;
  }

  // Calculate which edge this movement crosses
  const edge = getEdgeForMovement(fromPos.x, fromPos.y, dx, dy);

  // Check if edge is blocked by a room wall (region boundary)
  const roomWallBlocked = isEdgeBlockedByRoomWall(fromPos, toPos, edge);
  if (roomWallBlocked) {
    // console.log(`COLLISION: Movement blocked from (${fromPos.x},${fromPos.y}) to (${toPos.x},${toPos.y}) by room wall`);
    return true;
  }

  // Check if edge is blocked by a blocking wall
  const wallBlocked = isEdgeBlockedByWall(edge);
  if (wallBlocked) {
    console.log(`COLLISION: Movement blocked from (${fromPos.x},${fromPos.y}) to (${toPos.x},${toPos.y}) by blocking wall`);
    return true;
  }

  // Check if edge is blocked by a closed door
  const doorBlocked = isEdgeBlockedByClosedDoor(edge);
  if (doorBlocked) {
    // console.log(`COLLISION: Movement blocked from (${fromPos.x},${fromPos.y}) to (${toPos.x},${toPos.y}) by closed door`);
    return true;
  }

  return false;
}

/**
 * Get the edge address for a movement step (matches server-side edgeForStep function)
 */
function getEdgeForMovement(x, y, dx, dy) {
  if (dx === 1 && dy === 0) {
    // Moving right: cross the right edge of current tile = left edge of tile (x+1,y)
    return { x: x + 1, y: y, orientation: 'vertical' };
  }
  if (dx === -1 && dy === 0) {
    // Moving left: cross the left edge of current tile = left edge of tile (x,y)
    return { x: x, y: y, orientation: 'vertical' };
  }
  if (dx === 0 && dy === 1) {
    // Moving down: cross the bottom edge of current tile = top edge of tile (x,y+1)
    return { x: x, y: y + 1, orientation: 'horizontal' };
  }
  if (dx === 0 && dy === -1) {
    // Moving up: cross the top edge of current tile = top edge of tile (x,y)
    return { x: x, y: y, orientation: 'horizontal' };
  }
  return null;
}

/**
 * Check if an edge is blocked by a wall
 */
function isEdgeBlockedByWall(edge) {
  if (!edge) {
    return false;
  }

  // Use current game state blocking walls, not snapshot (which might be outdated)
  if (!gameState.blockingWalls || gameState.blockingWalls.length === 0) {
    console.log('WALL-CHECK: No blocking walls in game state');
    return false;
  }

  // console.log(`WALL-CHECK: Checking edge (${edge.x},${edge.y},${edge.orientation}) against ${gameState.blockingWalls.length} blocking walls`);
  for (const wall of gameState.blockingWalls) {
    // console.log(`WALL-CHECK: Wall at (${wall.x},${wall.y},${wall.orientation})`);
    if (doesWallBlockEdge(wall, edge)) {
      // console.log(`WALL-CHECK: Wall blocks edge!`);
      return true;
    }
  }

  // console.log('WALL-CHECK: No walls block this edge');
  return false;
}

/**
 * Check if an edge is blocked by a closed door
 */
function isEdgeBlockedByClosedDoor(edge) {
  if (!edge) return false;

  // Use current game state thresholds, not snapshot (which is cached)
  if (!gameState.thresholds) return false;

  // Check all thresholds (doors) to see if any closed door is at this edge
  for (const threshold of gameState.thresholds) {
    if (threshold.x === edge.x &&
        threshold.y === edge.y &&
        threshold.orientation === edge.orientation &&
        threshold.state === 'closed') {
      return true;
    }
  }

  return false;
}

/**
 * Check if an edge is blocked by a room wall (region boundary without a door)
 */
function isEdgeBlockedByRoomWall(fromPos, toPos, edge) {
  if (!edge) return false;

  const snapshot = gameState.snapshot;
  if (!snapshot?.tileRegionIds) return false;

  // Get region IDs for both positions
  const fromRegionId = getTileRegionId(fromPos.x, fromPos.y);
  const toRegionId = getTileRegionId(toPos.x, toPos.y);

  // If both tiles are in the same region, no room wall blocks the movement
  if (fromRegionId === toRegionId) {
    return false;
  }

  // If tiles are in different regions, there's a room wall between them
  // Check if there's an open door at this edge that allows passage
  const hasOpenDoor = hasOpenDoorAtEdge(edge);
  if (hasOpenDoor) {
    // console.log(`ROOM-WALL: Open door allows passage from region ${fromRegionId} to ${toRegionId} at (${fromPos.x},${fromPos.y}) → (${toPos.x},${toPos.y})`);
    return false; // Open door allows passage through room wall
  }

  // Check if there's a closed door at this edge
  const hasClosedDoor = isEdgeBlockedByClosedDoor(edge);
  if (hasClosedDoor) {
    // console.log(`ROOM-WALL: Closed door blocks passage from region ${fromRegionId} to ${toRegionId} at (${fromPos.x},${fromPos.y}) → (${toPos.x},${toPos.y})`);
    return true; // Closed door blocks movement
  }

  // Different regions with no door at all = blocked by solid room wall
  // console.log(`ROOM-WALL: Solid wall blocks passage from region ${fromRegionId} to ${toRegionId} at (${fromPos.x},${fromPos.y}) → (${toPos.x},${toPos.y})`);
  return true;
}

/**
 * Get the region ID for a tile
 */
function getTileRegionId(x, y) {
  const snapshot = gameState.snapshot;
  if (!snapshot?.tileRegionIds) return null;

  const mapWidth = snapshot.mapWidth || 26;
  const index = y * mapWidth + x;

  if (index >= 0 && index < snapshot.tileRegionIds.length) {
    return snapshot.tileRegionIds[index];
  }

  return null;
}

/**
 * Check if there's an open door at the specified edge
 */
function hasOpenDoorAtEdge(edge) {
  if (!edge) return false;

  // Use current game state thresholds, not snapshot (which is cached)
  if (!gameState.thresholds) return false;

  // Check all thresholds (doors) to see if any open door is at this edge
  for (const threshold of gameState.thresholds) {
    if (threshold.x === edge.x &&
        threshold.y === edge.y &&
        threshold.orientation === edge.orientation &&
        threshold.state === 'open') {
      return true;
    }
  }

  return false;
}

/**
 * Check if a blocking wall blocks a specific edge
 */
function doesWallBlockEdge(wall, edge) {
  // Handle multi-tile walls
  const size = wall.size || 1;

  for (let i = 0; i < size; i++) {
    let wallX = wall.x;
    let wallY = wall.y;

    // Calculate position of this wall segment
    if (wall.orientation === 'horizontal') {
      wallX += i;
    } else {
      wallY += i;
    }

    // Check if this wall segment blocks the edge
    if (wall.orientation === 'horizontal') {
      // Horizontal wall blocks vertical movement (north-south movement crosses horizontal edges)
      if (edge.orientation === 'horizontal' &&
          edge.x === wallX && edge.y === wallY) {
        console.log(`WALL-BLOCK: Horizontal wall at (${wallX},${wallY}) blocks horizontal edge at (${edge.x},${edge.y})`);
        return true;
      }
    } else {
      // Vertical wall blocks horizontal movement (east-west movement crosses vertical edges)
      if (edge.orientation === 'vertical' &&
          edge.x === wallX && edge.y === wallY) {
        console.log(`WALL-BLOCK: Vertical wall at (${wallX},${wallY}) blocks vertical edge at (${edge.x},${edge.y})`);
        return true;
      }
    }
  }

  return false;
}

/**
 * Check if a monster blocks movement to this position
 */
function isMonsterBlocking(pos) {
  // Check if any monster is at this position
  for (const [entityId, entityPos] of gameState.entityPositions.entries()) {
    if (entityId.startsWith('monster-') && entityPos.x === pos.x && entityPos.y === pos.y) {
      return true;
    }
  }
  return false;
}

/**
 * Check if furniture blocks movement to this position
 */
function isFurnitureBlocking(pos) {
  const snapshot = gameState.snapshot;
  if (!snapshot || !snapshot.furniture) return false;

  // Check all furniture to see if any blocks this tile
  for (const furniture of snapshot.furniture) {
    if (furniture.blocksMovement && doesFurnitureBlockTile(furniture, pos)) {
      // console.log(`FURNITURE-BLOCK: Tile (${pos.x},${pos.y}) blocked by ${furniture.type} at (${furniture.tile.x},${furniture.tile.y})`);
      return true;
    }
  }
  return false;
}

/**
 * Helper function to check if furniture blocks a specific tile
 */
function doesFurnitureBlockTile(furniture, pos) {
  if (!furniture.tile || !furniture.gridSize) {
    return false;
  }

  // Get effective grid size considering rotation
  let effectiveWidth = furniture.gridSize.width;
  let effectiveHeight = furniture.gridSize.height;

  // For 90/270 degree rotations, swap width and height if swapAspectOnRotate is true
  if (furniture.swapAspectOnRotate && (furniture.rotation === 90 || furniture.rotation === 270)) {
    effectiveWidth = furniture.gridSize.height;
    effectiveHeight = furniture.gridSize.width;
  }

  // Check if position is within the furniture's effective grid area
  const startX = furniture.tile.x;
  const startY = furniture.tile.y;
  const endX = startX + effectiveWidth - 1;
  const endY = startY + effectiveHeight - 1;

  const isBlocked = pos.x >= startX && pos.x <= endX && pos.y >= startY && pos.y <= endY;

  return isBlocked;
}

/**
 * Send the next movement step in the execution queue
 * Only sends one step and waits for server confirmation
 */
function sendNextMovementStep() {
  // console.log('MOVEMENT-EXEC: sendNextMovementStep() called');
  // console.log('MOVEMENT-EXEC: isExecuting:', movementState.isExecuting);
  // console.log('MOVEMENT-EXEC: executionQueue.length:', movementState.executionQueue.length);

  if (!movementState.isExecuting || movementState.executionQueue.length === 0) {
    // console.log('MOVEMENT-EXEC: Cannot send step - not executing or queue empty');
    return;
  }

  const socket = gameState.getSocket();
  if (!socket) {
    console.error('MOVEMENT-EXEC: WebSocket not connected');
    movementState.isExecuting = false;
    return;
  }

  // Get the next step from the queue (but don't remove it yet)
  const targetPos = movementState.executionQueue[0];

  // Get current hero position
  const heroPosition = gameState.getEntityPosition('hero-1');
  if (!heroPosition) {
    console.error('MOVEMENT-EXEC: Could not get hero position');
    return;
  }

  // Calculate relative movement (dx, dy)
  const dx = targetPos.x - heroPosition.x;
  const dy = targetPos.y - heroPosition.y;

  // console.log(`MOVEMENT-EXEC: Hero at (${heroPosition.x}, ${heroPosition.y}), moving by (${dx}, ${dy}) to (${targetPos.x}, ${targetPos.y})`);

  // Validate that this is a single-step movement
  if (Math.abs(dx) > 1 || Math.abs(dy) > 1 || (dx !== 0 && dy !== 0)) {
    console.error(`MOVEMENT-EXEC: Invalid movement step - dx=${dx}, dy=${dy}. Must be single adjacent step.`);
    return;
  }

  const moveRequest = {
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

  // console.log('MOVEMENT-EXEC: WebSocket message:', JSON.stringify(moveRequest));
  socket.send(JSON.stringify(moveRequest));
}

/**
 * Continue execution after successful movement
 * Called by handleEntityMovement when movement is confirmed
 */
function continueMovementExecution() {
  if (!movementState.isExecuting || movementState.executionQueue.length === 0) {
    return;
  }

  // Note: Movement tracking is handled in handleEntityMovement for all movement types

  // Remove the completed step from the queue
  movementState.executionQueue.shift();

  // If there are more steps, send the next one
  if (movementState.executionQueue.length > 0) {
    // console.log(`MOVEMENT-EXEC: Continuing with ${movementState.executionQueue.length} steps remaining`);
    sendNextMovementStep();
  } else {
    // console.log(`MOVEMENT-EXEC: All movement steps completed`);
    movementState.isExecuting = false;

    // Continue movement planning with remaining movement if any
    const remainingMovement = getRemainingMovement();
    // console.log(`MOVEMENT-EXEC: Execution completed, remaining movement: ${remainingMovement}`);
    if (remainingMovement > 0) {
      // console.log(`MOVEMENT-EXEC: ${remainingMovement} movement remaining - continuing planning`);
      // Update movement state without resetting position/progress
      // Keep movementState.usedMovement as-is to track total movement used during planning
      movementState.maxMovement = turnMovementState.maxMovementForTurn;
      movementState.usedMovement = turnMovementState.movementUsedThisTurn;
      movementState.availableMovement = remainingMovement;
      movementState.plannedPath = [];
      movementState.pathPreview = [];

      // Update current position to actual hero position after movement
      movementState.currentPosition = gameState.getEntityPosition('hero-1') || movementState.currentPosition;

      // Recalculate valid tiles from updated position
      calculateValidMovementTiles(movementState.currentPosition, remainingMovement);
      gameState.requestRedraw();
    } else {
      // console.log('MOVEMENT-EXEC: No movement remaining - ending planning');
      endMovementPlanning();
    }
  }
}

/**
 * Handle movement interruption (failed movement, traps, etc.)
 * This stops execution and keeps planning mode active for remaining movement
 */
export function interruptMovementExecution() {
  if (!movementState.isExecuting) {
    return;
  }

  // console.log(`MOVEMENT-EXEC: Movement interrupted - ${reason}`);

  // Stop execution but keep planning mode active
  movementState.isExecuting = false;
  movementState.executionQueue = [];

  // Recalculate movement visualization with remaining movement
  if (movementState.availableMovement > 0) {
    calculateValidMovementTiles(movementState.currentPosition, movementState.availableMovement);
  }

  gameState.requestRedraw();
}

/**
 * Handle entity movement to update planning state
 * @param {string} entityId
 * @param {Object} newPosition
 * @param {Object} previousPosition - The position before the move
 */
export function handleEntityMovement(entityId, newPosition, previousPosition = null) {
  // console.log(`MOVEMENT-DEBUG: handleEntityMovement called for ${entityId} to (${newPosition.x}, ${newPosition.y})`);

  if (entityId !== 'hero-1') {
    return;
  }

  // Always track movement in turn state, regardless of planning status
  if (turnMovementState.diceRolled) {
    // Use previous position if provided, otherwise try to get from game state
    const currentPos = previousPosition || gameState.getEntityPosition('hero-1') || { x: 0, y: 0 };
    const moveCost = Math.abs(newPosition.x - currentPos.x) + Math.abs(newPosition.y - currentPos.y);

    // console.log(`MOVEMENT-DEBUG: Previous pos (${currentPos.x}, ${currentPos.y}), new pos (${newPosition.x}, ${newPosition.y}), cost: ${moveCost}`);

    // Track manual movement in history if not planning
    if (!movementState.isPlanning && moveCost > 0) {
      // Start a new manual movement segment if not already tracking one
      if (!turnMovementState.currentSegment || turnMovementState.currentSegment.type !== 'manual') {
        startMovementSegment('manual', currentPos);
      }

      // Add this movement step to the segment
      addMovementStep(newPosition);
    }

    // Track each movement step in the turn state
    for (let i = 0; i < moveCost; i++) {
      if (!trackMovementStep()) {
        console.error('MOVEMENT-TRACKING: Failed to track movement step in turn state');
        break;
      }
    }
  }

  // If planning is active, update planning state
  if (movementState.isPlanning) {
    // Update movement planning position
    movementState.currentPosition = { ...newPosition };

    // Sync movementState with turnMovementState
    if (turnMovementState.diceRolled) {
      movementState.maxMovement = turnMovementState.maxMovementForTurn;
      movementState.usedMovement = turnMovementState.movementUsedThisTurn;
      movementState.availableMovement = getRemainingMovement();
    }
  }

  // Update flood effect for both manual and planned movement
  if (turnMovementState.diceRolled) {
    const remainingMovement = getRemainingMovement();
    if (remainingMovement > 0) {
      // Recalculate valid tiles from current position with remaining movement
      calculateValidMovementTiles(newPosition, remainingMovement);
    } else {
      // No movement left, clear valid tiles
      movementState.validTiles.clear();
    }
    gameState.requestRedraw();

    // Update movement UI to reflect current state
    updateMovementStatusUI(movementState);
  }

  // Handle planning-specific logic
  if (!movementState.isPlanning) {
    return;
  }

  // console.log(`MOVEMENT-EXEC: Entity movement confirmed to (${newPosition.x}, ${newPosition.y})`);

  // Planning state was already updated above

  // If we're executing movement, continue with next step or finish
  if (movementState.isExecuting) {
    // Continue with the next movement step in queue
    continueMovementExecution();

    // If execution is complete and no movement remains, end planning
    if (!movementState.isExecuting && getRemainingMovement() === 0) {
      endMovementPlanning();
      return;
    }
  } else {
    // If we're planning but not executing, this is manual movement
    // Clear any existing planned path since user moved manually
    movementState.plannedPath = [];
    movementState.pathPreview = [];
  }

  // Recalculate valid tiles with remaining movement
  const remainingMovement = getRemainingMovement();
  if (remainingMovement > 0) {
    calculateValidMovementTiles(movementState.currentPosition, remainingMovement);
  } else {
    movementState.validTiles.clear();
    // If no movement remaining and not executing, end planning
    if (!movementState.isExecuting) {
      endMovementPlanning();
      return;
    }
  }

  // Trigger UI update
  gameState.requestRedraw();
}

/**
 * Restore movement state from snapshot data (called on page load)
 * @param {Object} snapshot - The game snapshot containing turn variables
 */
export function restoreMovementStateFromSnapshot(snapshot) {
  if (!snapshot?.variables) {
    return;
  }

  const vars = snapshot.variables;

  // Restore turn movement state
  if (vars['turn.movementRolled'] === true) {
    turnMovementState.diceRolled = true;

    // If we have the actual movement rolls, use them to calculate max movement
    if (Array.isArray(vars['turn.movementRolls']) && vars['turn.movementRolls'].length > 0) {
      turnMovementState.maxMovementForTurn = vars['turn.movementRolls'].reduce((a, b) => a + b, 0);
    } else {
      // Fallback: calculate from remaining movement and usage indicators
      const remaining = vars['turn.movement'] || 0;
      const hasMoved = vars['turn.hasMoved'] || false;
      // Estimate total movement (this is imperfect but better than nothing)
      turnMovementState.maxMovementForTurn = remaining + (hasMoved ? 1 : 0);
    }

    turnMovementState.movementUsedThisTurn = turnMovementState.maxMovementForTurn - (vars['turn.movement'] || 0);
    turnMovementState.initialHeroPosition = getHeroPosition();

    console.log(`MOVEMENT-TRACKING: Restored from snapshot - ${turnMovementState.maxMovementForTurn} total, ${turnMovementState.movementUsedThisTurn} used, ${vars['turn.movement']} remaining`);
  } else {
    // Reset state if no dice rolled
    resetTurnMovement();
  }
}

/**
 * Expose movement state for external access
 */
export { movementState };