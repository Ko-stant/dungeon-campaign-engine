/**
 * Monster system - monster selection and interaction
 */

import { calculateAdjacentTile } from './geometry.js';
import { gameState } from './gameState.js';

/**
 * Find monster ID from entity position and movement direction
 * @param {string} entityId
 * @param {number} dx
 * @param {number} dy
 * @returns {string|null}
 */
export function findAdjacentMonsterIdFromDirection(entityId, dx, dy) {
  const pos = gameState.getEntityPosition(entityId);
  if (!pos) {
    return null;
  }

  const adjacentTile = calculateAdjacentTile(pos, dx, dy);
  if (!adjacentTile) {
    return null;
  }

  const monster = gameState.findMonsterAtPosition(adjacentTile.x, adjacentTile.y);
  return monster ? monster.id : null;
}

/**
 * Select monster in the given direction from hero
 * @param {number} dx
 * @param {number} dy
 */
export function selectMonsterInDirection(dx, dy) {
  const monsterId = findAdjacentMonsterIdFromDirection('hero-1', dx, dy);
  gameState.setSelectedMonsterId(monsterId);
  gameState.requestRedraw();
}

/**
 * Get selected monster information
 * @returns {MonsterLite|null}
 */
export function getSelectedMonster() {
  const selectedMonsterId = gameState.getSelectedMonsterId();
  return selectedMonsterId ? gameState.getMonsterById(selectedMonsterId) : null;
}

/**
 * Check if a monster is selected
 * @returns {boolean}
 */
export function isMonsterSelected() {
  return gameState.getSelectedMonsterId() !== null;
}

/**
 * Clear monster selection
 */
export function clearMonsterSelection() {
  gameState.setSelectedMonsterId(null);
  gameState.requestRedraw();
}

/**
 * Get all visible, alive monsters
 * @returns {MonsterLite[]}
 */
export function getVisibleMonsters() {
  return gameState.monsters.filter(m => m && m.isVisible && m.isAlive && m.tile);
}

/**
 * Get all monsters (including dead/invisible)
 * @returns {MonsterLite[]}
 */
export function getAllMonsters() {
  return gameState.monsters;
}

/**
 * Check if monster is alive
 * @param {string} monsterId
 * @returns {boolean}
 */
export function isMonsterAlive(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  return monster ? monster.isAlive : false;
}

/**
 * Check if monster is visible
 * @param {string} monsterId
 * @returns {boolean}
 */
export function isMonsterVisible(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  return monster ? monster.isVisible : false;
}

/**
 * Get monster health information
 * @param {string} monsterId
 * @returns {{body: number, maxBody: number, mind: number, maxMind: number}|null}
 */
export function getMonsterHealth(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  if (!monster) {
    return null;
  }

  return {
    body: monster.body,
    maxBody: monster.MaxBody,
    mind: monster.mind,
    maxMind: monster.maxMind,
  };
}

/**
 * Get monster combat stats
 * @param {string} monsterId
 * @returns {{attackDice: number, defenseDice: number}|null}
 */
export function getMonsterCombatStats(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  if (!monster) {
    return null;
  }

  return {
    attackDice: monster.attackDice,
    defenseDice: monster.defenseDice,
  };
}

/**
 * Check if monster is damaged
 * @param {string} monsterId
 * @returns {boolean}
 */
export function isMonsterDamaged(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  return monster ? monster.body < monster.MaxBody : false;
}

/**
 * Get monster type display name
 * @param {string} monsterId
 * @returns {string}
 */
export function getMonsterDisplayName(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  if (!monster) {
    return 'Unknown';
  }

  // Convert snake_case to Title Case
  return monster.type
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Get monster position
 * @param {string} monsterId
 * @returns {TileAddress|null}
 */
export function getMonsterPosition(monsterId) {
  const monster = gameState.getMonsterById(monsterId);
  return monster ? monster.tile : null;
}

/**
 * Update monster details UI
 */
export function updateMonsterDetailsUI() {
  const detailsDiv = document.getElementById('monsterDetails');
  const contentDiv = document.getElementById('monsterDetailsContent');

  if (!detailsDiv || !contentDiv) {
    return;
  }

  const selectedMonsterId = gameState.getSelectedMonsterId();
  if (!selectedMonsterId) {
    detailsDiv.style.display = 'none';
    return;
  }

  const monster = getSelectedMonster();
  if (!monster) {
    detailsDiv.style.display = 'none';
    return;
  }

  // Build monster details HTML
  const detailsHTML = `
    <div class="space-y-1">
      <div><span class="opacity-70">Type:</span> ${getMonsterDisplayName(selectedMonsterId)}</div>
      <div><span class="opacity-70">Body:</span> ${monster.body}/${monster.MaxBody}</div>
      <div><span class="opacity-70">Mind:</span> ${monster.mind}/${monster.maxMind}</div>
      <div><span class="opacity-70">Attack Dice:</span> ${monster.attackDice}</div>
      <div><span class="opacity-70">Defense Dice:</span> ${monster.defenseDice}</div>
      <div><span class="opacity-70">Status:</span> ${monster.isAlive ? 'Alive' : 'Dead'}</div>
    </div>
  `;

  contentDiv.innerHTML = detailsHTML;
  detailsDiv.style.display = 'block';
}

/**
 * Find monster at specific coordinates
 * @param {number} x
 * @param {number} y
 * @returns {MonsterLite|null}
 */
export function findMonsterAtTile(x, y) {
  return gameState.findMonsterAtPosition(x, y);
}

/**
 * Get monsters in a specific area
 * @param {number} startX
 * @param {number} startY
 * @param {number} width
 * @param {number} height
 * @returns {MonsterLite[]}
 */
export function getMonstersInArea(startX, startY, width, height) {
  return getVisibleMonsters().filter(monster => {
    const tile = monster.tile;
    return tile.x >= startX && tile.x < startX + width &&
      tile.y >= startY && tile.y < startY + height;
  });
}

/**
 * Get monsters adjacent to a position
 * @param {number} x
 * @param {number} y
 * @returns {MonsterLite[]}
 */
export function getAdjacentMonsters(x, y) {
  const directions = [
    { dx: -1, dy: 0 },  // left
    { dx: 1, dy: 0 },   // right
    { dx: 0, dy: -1 },  // up
    { dx: 0, dy: 1 },    // down
  ];

  const adjacentMonsters = [];
  for (const dir of directions) {
    const monster = findMonsterAtTile(x + dir.dx, y + dir.dy);
    if (monster) {
      adjacentMonsters.push(monster);
    }
  }

  return adjacentMonsters;
}