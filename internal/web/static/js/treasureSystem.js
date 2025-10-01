/**
 * Treasure system - handles treasure display, notifications, and inventory updates
 */

/**
 * Display treasure found notification
 * @param {Object} treasureResult - Treasure result from server
 */
export function displayTreasureNotification(treasureResult) {
  if (!treasureResult) return;

  const notification = createTreasureNotification(treasureResult);
  if (notification) {
    showNotification(notification);
  }
}

/**
 * Create treasure notification HTML
 * @param {Object} treasureResult
 * @returns {string|null}
 */
function createTreasureNotification(treasureResult) {
  let html = '';
  let hasContent = false;

  // Gold found
  if (treasureResult.foundGold && treasureResult.foundGold > 0) {
    html += `
      <div class="treasure-notification gold-notification">
        <div class="text-yellow-300 font-semibold text-lg">💰 ${treasureResult.foundGold} Gold Coins!</div>
      </div>
    `;
    hasContent = true;
  }

  // Items found
  if (treasureResult.foundItems && treasureResult.foundItems.length > 0) {
    html += '<div class="treasure-notification items-notification">';
    html += '<div class="text-yellow-300 font-semibold mb-2">📦 Items Found:</div>';
    treasureResult.foundItems.forEach(item => {
      html += `<div class="text-yellow-200 text-sm ml-4">• ${item.name}</div>`;
    });
    html += '</div>';
    hasContent = true;
  }

  // Empty chest
  if (treasureResult.isEmpty) {
    html += `
      <div class="treasure-notification empty-notification">
        <div class="text-gray-400 font-semibold">Empty</div>
        <div class="text-gray-500 text-sm">${treasureResult.message || 'Nothing found'}</div>
      </div>
    `;
    hasContent = true;
  }

  // Hazard
  if (treasureResult.isHazard) {
    html += `
      <div class="treasure-notification hazard-notification">
        <div class="text-red-300 font-semibold text-lg">⚠️ Hazard!</div>
        <div class="text-red-200 text-sm">${treasureResult.message}</div>
        ${treasureResult.hazardDamage ? `<div class="text-red-300 mt-1">💥 ${treasureResult.hazardDamage} Damage</div>` : ''}
      </div>
    `;
    hasContent = true;
  }

  // Wandering Monster
  if (treasureResult.isMonster) {
    html += `
      <div class="treasure-notification monster-notification">
        <div class="text-red-300 font-semibold text-lg">👹 Wandering Monster!</div>
        <div class="text-red-200 text-sm">${treasureResult.monsterType || 'Monster'} appears!</div>
      </div>
    `;
    hasContent = true;
  }

  return hasContent ? html : null;
}

/**
 * Show notification in UI
 * @param {string} html
 */
function showNotification(html) {
  // Try to use existing notification system or create temporary one
  const container = document.getElementById('treasureNotifications') || createNotificationContainer();

  const notification = document.createElement('div');
  notification.className = 'treasure-notification-wrapper';
  notification.innerHTML = html;

  container.appendChild(notification);

  // Auto-remove after 5 seconds
  setTimeout(() => {
    notification.style.opacity = '0';
    setTimeout(() => {
      notification.remove();
    }, 300);
  }, 5000);
}

/**
 * Create notification container if it doesn't exist
 * @returns {HTMLElement}
 */
function createNotificationContainer() {
  let container = document.getElementById('treasureNotifications');
  if (!container) {
    container = document.createElement('div');
    container.id = 'treasureNotifications';
    container.style.cssText = `
      position: fixed;
      top: 80px;
      right: 20px;
      z-index: 1000;
      max-width: 400px;
    `;
    document.body.appendChild(container);
  }
  return container;
}

/**
 * Display inventory summary
 * @param {Object} inventory
 */
export function displayInventorySummary(inventory) {
  const panel = document.getElementById('inventoryPanel');
  if (!panel) return;

  let html = '<div class="inventory-summary">';

  // Gold
  html += `<div class="inventory-gold mb-3">
    <span class="text-yellow-300 font-semibold">💰 Gold:</span>
    <span class="text-yellow-200">${inventory.gold || 0}</span>
  </div>`;

  // Equipment
  if (inventory.equipment && Object.keys(inventory.equipment).length > 0) {
    html += '<div class="inventory-equipment mb-3">';
    html += '<div class="text-blue-300 font-semibold mb-1">⚔️ Equipped:</div>';
    for (const [slot, item] of Object.entries(inventory.equipment)) {
      html += `<div class="text-sm text-blue-200 ml-2">• ${slot}: ${item.name}</div>`;
    }
    html += '</div>';
  }

  // Carried items
  if (inventory.carried && inventory.carried.length > 0) {
    html += '<div class="inventory-carried mb-3">';
    html += '<div class="text-gray-300 font-semibold mb-1">🎒 Carried:</div>';
    inventory.carried.forEach(item => {
      html += `<div class="text-sm text-gray-200 ml-2">• ${item.name}</div>`;
    });
    html += '</div>';
  }

  html += '</div>';
  panel.innerHTML = html;
}

/**
 * Update inventory display after treasure found
 * @param {Object} result - Action result with treasure info
 */
export function updateInventoryAfterTreasure(result) {
  // This would be called after treasure is added to inventory
  // For now, just log - full inventory sync would come from server patches
  console.log('Inventory updated after treasure:', result);
}
