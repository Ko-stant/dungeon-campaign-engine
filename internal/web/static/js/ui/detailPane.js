/**
 * Detail Pane Controller
 * Shows card images and detailed information for actions, items, and spells
 */

export class DetailPaneController {
  constructor() {
    this.paneElement = document.getElementById('detailPane');
    this.titleElement = document.getElementById('detailTitle');
    this.descriptionElement = document.getElementById('detailDescription');
    this.statsElement = document.getElementById('detailStats');
    this.cardImageElement = document.getElementById('detailCardImage');
    this.cardPlaceholderElement = document.getElementById('detailCardPlaceholder');
  }

  /**
   * Clear all content and show placeholder
   */
  clear() {
    if (this.titleElement) this.titleElement.textContent = '';
    if (this.descriptionElement) {
      this.descriptionElement.innerHTML = '<p class="text-sm opacity-50 text-center py-4">Action results and card details will appear here</p>';
    }
    if (this.statsElement) this.statsElement.innerHTML = '';
    this.hideCardImage();
  }

  /**
   * Show card image
   * @param {string} imagePath
   */
  showCardImage(imagePath) {
    if (!this.cardImageElement || !this.cardPlaceholderElement) return;

    this.cardImageElement.src = imagePath;
    this.cardImageElement.style.display = 'block';
    this.cardPlaceholderElement.style.display = 'none';

    // Handle image load errors
    this.cardImageElement.onerror = () => {
      this.hideCardImage();
    };
  }

  /**
   * Hide card image and show placeholder
   */
  hideCardImage() {
    if (!this.cardImageElement || !this.cardPlaceholderElement) return;

    this.cardImageElement.style.display = 'none';
    this.cardImageElement.src = '';
    this.cardPlaceholderElement.style.display = 'block';
  }

  /**
   * Display action result (attack, spell, search, etc.)
   * @param {Object} result - HeroActionResult from server
   */
  showActionResult(result) {
    this.clear();

    // Set title based on action type
    const actionTitles = {
      attack: 'Attack Result',
      cast_spell: 'Spell Cast',
      search_treasure: 'Treasure Search',
      search_traps: 'Trap Search',
      search_hidden_doors: 'Secret Door Search',
      move: 'Movement'
    };

    const title = actionTitles[result.action] || result.action;
    if (this.titleElement) {
      this.titleElement.textContent = title;
    }

    // Build description based on result
    let descriptionHTML = '';

    if (result.success) {
      descriptionHTML += `<p class="text-green-400 font-semibold mb-2">✓ ${result.message || 'Success'}</p>`;
    } else {
      descriptionHTML += `<p class="text-red-400 font-semibold mb-2">✗ ${result.message || 'Failed'}</p>`;
    }

    // Add specific details based on action type
    if (result.action === 'attack') {
      if (result.attackRolls && result.attackRolls.length > 0) {
        descriptionHTML += `<p class="mb-1">Attack Dice: ${this.formatDiceRolls(result.attackRolls)}</p>`;
      }

      if (result.defenseRolls && result.defenseRolls.length > 0) {
        descriptionHTML += `<p class="mb-1">Defense Dice: ${this.formatDiceRolls(result.defenseRolls)}</p>`;
      }

      if (result.damage !== undefined) {
        descriptionHTML += `<p class="mt-2 text-orange-400 font-semibold">Damage: ${result.damage}</p>`;
      }
    }

    if (result.action === 'search_treasure' && result.treasureFound) {
      descriptionHTML += `<p class="text-yellow-400">Found treasure: ${result.treasureFound.name || 'Unknown'}</p>`;
      // TODO: Show treasure card image when available
    }

    if (this.descriptionElement) {
      this.descriptionElement.innerHTML = descriptionHTML;
    }

    // Show any additional stats
    this.showActionStats(result);
  }

  /**
   * Format dice rolls for display
   * @param {Array} rolls
   * @returns {string}
   */
  formatDiceRolls(rolls) {
    if (!Array.isArray(rolls)) return '';

    const results = rolls.map(r => {
      if (r.result === 'skull') return '💀';
      if (r.result === 'shield') return '🛡️';
      if (r.result === 'blank') return '⚪';
      return r.result;
    });

    return results.join(' ');
  }

  /**
   * Show action statistics
   * @param {Object} result
   */
  showActionStats(result) {
    if (!this.statsElement) return;

    let statsHTML = '';

    if (result.entityId) {
      statsHTML += `<div>Actor: ${result.entityId}</div>`;
    }

    if (result.targetId) {
      statsHTML += `<div>Target: ${result.targetId}</div>`;
    }

    if (result.timestamp) {
      const date = new Date(result.timestamp);
      statsHTML += `<div>Time: ${date.toLocaleTimeString()}</div>`;
    }

    this.statsElement.innerHTML = statsHTML;
  }

  /**
   * Display item card details
   * @param {Object} item - Item card data
   */
  showItem(item) {
    this.clear();

    if (this.titleElement) {
      this.titleElement.textContent = item.name || 'Item';
    }

    let descriptionHTML = '';

    if (item.description) {
      descriptionHTML += `<p class="mb-2">${item.description}</p>`;
    }

    if (item.type) {
      descriptionHTML += `<p class="text-xs opacity-70">Type: ${item.type}</p>`;
    }

    if (this.descriptionElement) {
      this.descriptionElement.innerHTML = descriptionHTML;
    }

    // Show item stats if available
    if (item.stats && this.statsElement) {
      let statsHTML = '';
      for (const [key, value] of Object.entries(item.stats)) {
        const label = key.replace(/([A-Z])/g, ' $1').trim();
        const capitalizedLabel = label.charAt(0).toUpperCase() + label.slice(1);
        statsHTML += `<div>${capitalizedLabel}: ${value}</div>`;
      }
      this.statsElement.innerHTML = statsHTML;
    }

    // Show item card image if available
    if (item.cardImage) {
      this.showCardImage(item.cardImage);
    }
  }

  /**
   * Display spell card details
   * @param {Object} spell - Spell card data
   */
  showSpell(spell) {
    this.clear();

    if (this.titleElement) {
      this.titleElement.textContent = spell.name || 'Spell';
    }

    let descriptionHTML = '';

    if (spell.description) {
      descriptionHTML += `<p class="mb-2">${spell.description}</p>`;
    }

    if (spell.element) {
      descriptionHTML += `<p class="text-xs opacity-70">Element: ${spell.element}</p>`;
    }

    if (this.descriptionElement) {
      this.descriptionElement.innerHTML = descriptionHTML;
    }

    // Show spell card image if available
    if (spell.cardImage) {
      this.showCardImage(spell.cardImage);
    }
  }

  /**
   * Display treasure card details
   * @param {Object} treasure - Treasure card data
   */
  showTreasure(treasure) {
    this.clear();

    if (this.titleElement) {
      this.titleElement.textContent = treasure.name || 'Treasure';
    }

    let descriptionHTML = '';

    if (treasure.description) {
      descriptionHTML += `<p class="mb-2">${treasure.description}</p>`;
    }

    if (treasure.value) {
      descriptionHTML += `<p class="text-yellow-400">Gold: ${treasure.value}</p>`;
    }

    if (this.descriptionElement) {
      this.descriptionElement.innerHTML = descriptionHTML;
    }

    // Show treasure card image if available
    if (treasure.cardImage) {
      this.showCardImage(treasure.cardImage);
    }
  }
}
