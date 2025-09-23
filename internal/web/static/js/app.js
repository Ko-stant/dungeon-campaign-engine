/**
 * Main application module - coordinates all other modules
 */

import { gameState } from './gameState.js';
import {
  resizeCanvas,
  drawBackground,
  drawGrid,
  drawRegionBorders,
  drawEntities,
} from './rendering.js';
import {
  drawDoors,
  drawBlockingWalls,
  drawFurniture,
  drawMonsters,
  setDrawMainFunction,
} from './entityRendering.js';
import { initializeActionUI } from './actionSystem.js';
import { initializeInputHandling } from './inputHandling.js';
import { openWebSocket, setRedrawFunction } from './patchSystem.js';

/**
 * Main drawing function that renders the entire game board
 */
export function drawBoard() {
  console.log('DEBUG: drawBoard() called, entityPositions size:', gameState.entityPositions.size,
    'contents:', Array.from(gameState.entityPositions.entries()));

  // Draw in layers from back to front
  drawBackground();           // Background tiles based on region visibility
  drawGrid();                 // Subtle per-cell grid for counting
  drawRegionBorders();        // Bold room/corridor outlines
  drawFurniture();            // Furniture on top of regions
  drawMonsters();             // Monsters on top of furniture
  drawDoors();                // Door overlays on top of regions
  drawBlockingWalls();        // Blocking walls on top of regions
  drawEntities();             // Heroes and other entities on top
}

/**
 * Initialize the application
 */
export function initializeApp() {
  console.log('Initializing Dungeon Campaign Engine app...');

  // Get snapshot data from the page
  const snapshotElement = document.getElementById('snapshot');
  if (!snapshotElement) {
    console.error('No snapshot data found');
    return;
  }

  try {
    const snapshot = JSON.parse(snapshotElement.textContent);
    gameState.initializeFromSnapshot(snapshot);
  } catch (error) {
    console.error('Failed to parse snapshot data:', error);
    return;
  }

  // Get DOM element references
  const canvas = document.getElementById('board');
  const canvasContext = canvas?.getContext('2d');
  const patchCountElement = document.getElementById('patchCount');
  const toggleDoorButton = document.getElementById('toggleDoor');

  if (!canvas || !canvasContext) {
    console.error('Canvas element not found');
    return;
  }

  // Set DOM elements in game state
  gameState.setDOMElements({
    canvas,
    canvasContext,
    patchCountElement,
    toggleDoorButton,
  });

  // Set up canvas
  resizeCanvas(canvas, canvasContext);

  // Set up cross-module dependencies
  setDrawMainFunction(drawBoard);
  setRedrawFunction(() => requestAnimationFrame(drawBoard));

  // Initialize subsystems
  initializeActionUI();
  initializeInputHandling();

  // Set up window resize handler
  window.addEventListener('resize', () => {
    resizeCanvas(canvas, canvasContext);
    drawBoard();
  });

  // Initial draw
  drawBoard();

  // Open WebSocket connection
  openWebSocket();

  // Make drawBoard available globally for debugging
  window.drawBoard = drawBoard;
  window.gameState = gameState;

  console.log('App initialized successfully');
}

/**
 * Cleanup function for when the app is unloaded
 */
export async function cleanupApp() {
  console.log('Cleaning up Dungeon Campaign Engine app...');

  // Close WebSocket connection
  const { closeWebSocket } = await import('./patchSystem.js');
  closeWebSocket();

  // Remove event listeners
  const { cleanupInputHandling } = await import('./inputHandling.js');
  cleanupInputHandling();

  // Clear global references
  delete window.drawBoard;
  delete window.gameState;

  console.log('App cleanup complete');
}

// Auto-initialize when DOM is loaded
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initializeApp);
} else {
  // DOM is already ready
  initializeApp();
}

// Cleanup on page unload
window.addEventListener('beforeunload', cleanupApp);

// Export for manual initialization if needed
export { initializeApp as init, cleanupApp as cleanup };