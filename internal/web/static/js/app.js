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
  drawStartingPositionHighlights,
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
import { initMovementPlanning, drawMovementPlanning } from './movementPlanning.js';
import { initializeCanvasClickHandling } from './canvasClickHandler.js';
import { ActionsPanelController } from './ui/actionsPanel.js';
import { EntityModalController } from './ui/entityModal.js';
import { TurnCounterController } from './ui/turnCounter.js';
import { PlayerStatsPanelController } from './ui/playerStatsPanel.js';
import { DetailPaneController } from './ui/detailPane.js';
import { initializeGMControls } from './ui/gmControls.js';
import { initializeHeroTurnControls } from './ui/heroTurnControls.js';
import { QuestSetupController } from './ui/questSetupControls.js';

/**
 * Main drawing function that renders the entire game board
 */
export function drawBoard() {
  // Draw in layers from back to front
  drawBackground();           // Background tiles based on region visibility
  drawGrid();                 // Subtle per-cell grid for counting
  drawRegionBorders();        // Bold room/corridor outlines
  drawFurniture();            // Furniture on top of regions
  drawMonsters();             // Monsters on top of furniture
  drawDoors();                // Door overlays on top of regions
  drawBlockingWalls();        // Blocking walls on top of regions
  drawMovementPlanning(gameState.canvasContext); // Movement planning overlays
  drawStartingPositionHighlights(); // Quest setup position highlights
  drawEntities();             // Heroes and other entities on top
}

/**
 * Initialize the application
 */
export function initializeApp() {

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
  gameState.setRedrawFunction(() => requestAnimationFrame(drawBoard));

  // Initialize subsystems
  initializeActionUI();
  initializeInputHandling();
  initMovementPlanning();

  // Initialize new UI controllers
  const actionsPanelController = new ActionsPanelController();
  const entityModalController = new EntityModalController(gameState);
  const turnCounterController = new TurnCounterController(gameState);
  const playerStatsPanelController = new PlayerStatsPanelController(gameState);
  const detailPaneController = new DetailPaneController();

  // Initialize GM and Hero turn controls (will detect which page we're on)
  const gmControlsController = initializeGMControls(gameState);
  const heroTurnControlsController = initializeHeroTurnControls(gameState);

  // Initialize Quest Setup Controller
  const questSetupController = new QuestSetupController(gameState);

  // Make UI controllers available globally
  gameState.actionsPanelController = actionsPanelController;
  gameState.entityModalController = entityModalController;
  gameState.turnCounterController = turnCounterController;
  gameState.playerStatsPanelController = playerStatsPanelController;
  gameState.detailPaneController = detailPaneController;
  gameState.gmControlsController = gmControlsController;
  gameState.heroTurnControlsController = heroTurnControlsController;
  gameState.questSetupController = questSetupController;

  // Initialize UI from snapshot
  turnCounterController.updateFromSnapshot(gameState.snapshot);
  turnCounterController.updatePatchCount(0);
  playerStatsPanelController.updateFromSnapshot(gameState.snapshot);
  detailPaneController.clear(); // Show placeholder content
  questSetupController.updateFromSnapshot(gameState.snapshot);

  // Initialize canvas click handling for entity inspection
  initializeCanvasClickHandling();

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

  // Make movement planning available globally for testing
  import('./movementPlanning.js').then(module => {
    window.movementPlanningModule = module;
  });
}

/**
 * Cleanup function for when the app is unloaded
 */
export async function cleanupApp() {

  // Close WebSocket connection
  const { closeWebSocket } = await import('./patchSystem.js');
  closeWebSocket();

  // Remove event listeners
  const { cleanupInputHandling } = await import('./inputHandling.js');
  cleanupInputHandling();

  const { cleanupCanvasClickHandling } = await import('./canvasClickHandler.js');
  cleanupCanvasClickHandling();

  // Clear global references
  delete window.drawBoard;
  delete window.gameState;
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