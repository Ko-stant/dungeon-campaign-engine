/**
 * Player Stats Panel UI Controller
 * Displays hero stats, equipment, spells, and abilities
 */

export class PlayerStatsPanelController {
  constructor(gameState) {
    this.gameState = gameState;
    this.contentElement = document.getElementById('playerStatsContent');
    this.currentHeroId = null;
  }

  /**
   * Update stats panel from snapshot
   * @param {Object} snapshot
   */
  updateFromSnapshot(snapshot) {
    console.log('PLAYER-STATS: updateFromSnapshot called');
    console.log('PLAYER-STATS: snapshot:', snapshot);
    console.log('PLAYER-STATS: contentElement exists:', !!this.contentElement);

    if (!snapshot || !this.contentElement) return;

    // Find the first hero entity (for now, we'll support single player)
    const hero = this.findPlayerHero(snapshot);
    console.log('PLAYER-STATS: findPlayerHero returned:', hero);

    if (!hero) {
      console.log('PLAYER-STATS: No hero found, showing message');
      this.showNoHeroMessage();
      return;
    }

    this.currentHeroId = hero.id;
    console.log('PLAYER-STATS: Rendering hero stats for:', hero.id);
    this.renderHeroStats(hero, snapshot);
  }

  /**
   * Find the player's hero entity
   * @param {Object} snapshot
   * @returns {Object|null}
   */
  findPlayerHero(snapshot) {
    if (!snapshot.entities || !Array.isArray(snapshot.entities)) {
      return null;
    }

    // If we have a viewer entity ID, find that specific hero
    if (snapshot.viewerEntityId) {
      const viewerHero = snapshot.entities.find(e => e.id === snapshot.viewerEntityId);
      if (viewerHero) {
        return viewerHero;
      }
    }

    // Fallback: find first entity with kind="hero" (for backwards compatibility)
    return snapshot.entities.find(e => e.kind === 'hero') || null;
  }

  /**
   * Show message when no hero is found
   */
  showNoHeroMessage() {
    this.contentElement.innerHTML = `
      <div class="text-sm opacity-70 text-center py-8">
        No hero selected
      </div>
    `;
  }

  /**
   * Render hero stats
   * @param {Object} hero
   * @param {Object} snapshot
   */
  renderHeroStats(hero, snapshot) {
    const heroClass = this.getHeroClass(hero);
    const heroIcon = this.getHeroIcon(heroClass);
    const turnState = snapshot.heroTurnStates?.[hero.id];

    this.contentElement.innerHTML = `
      <div class="space-y-4">
        <!-- Hero Identity -->
        <div class="flex items-center gap-3 p-3 bg-surface-2 rounded-md border border-border/60">
          <div class="w-12 h-12 rounded-md bg-blue-600/20 border border-blue-600/40 flex items-center justify-center text-2xl">
            ${heroIcon}
          </div>
          <div>
            <div class="font-semibold">${heroClass}</div>
            <div class="text-xs opacity-70">Turn ${snapshot.turn || 1}</div>
          </div>
        </div>

        ${this.renderBodyPoints(hero)}
        ${this.renderMindPoints(hero)}
        ${this.renderMovementStatus(turnState)}
        ${this.renderEquipment(hero)}
        ${this.renderSpells(heroClass)}
        ${this.renderAbilities(heroClass)}
      </div>
    `;
  }

  /**
   * Get hero class from tags
   * @param {Object} hero
   * @returns {string}
   */
  getHeroClass(hero) {
    if (!hero.tags || !Array.isArray(hero.tags)) {
      return 'Hero';
    }

    const classNames = ['barbarian', 'wizard', 'dwarf', 'elf', 'berserker', 'knight', 'explorer', 'druid', 'monk'];
    const foundClass = hero.tags.find(tag => classNames.includes(tag.toLowerCase()));

    if (foundClass) {
      return foundClass.charAt(0).toUpperCase() + foundClass.slice(1);
    }

    return 'Hero';
  }

  /**
   * Get hero icon emoji
   * @param {string} heroClass
   * @returns {string}
   */
  getHeroIcon(heroClass) {
    const icons = {
      'Wizard': '🧙',
      'Barbarian': '⚔️',
      'Dwarf': '🪓',
      'Elf': '🏹',
      'Berserker': '🗡️',
      'Knight': '🛡️',
      'Explorer': '🧭',
      'Druid': '🌿',
      'Monk': '👊'
    };
    return icons[heroClass] || '🧑';
  }

  /**
   * Render body points with visual bar
   * @param {Object} hero
   * @returns {string}
   */
  renderBodyPoints(hero) {
    const current = hero.hp?.current ?? 0;
    const max = hero.hp?.max ?? 0;
    const percentage = max > 0 ? (current / max) * 100 : 0;

    return `
      <div class="space-y-1">
        <div class="flex items-center justify-between text-sm">
          <span class="opacity-70">Body Points</span>
          <span class="font-mono font-semibold text-red-400">${current} / ${max}</span>
        </div>
        <div class="h-2 bg-surface-2 rounded-full overflow-hidden">
          <div class="h-full bg-red-500 transition-all duration-300" style="width: ${percentage}%"></div>
        </div>
      </div>
    `;
  }

  /**
   * Render mind points with visual bar
   * @param {Object} hero
   * @returns {string}
   */
  renderMindPoints(hero) {
    // For now, mind points aren't in EntityLite HP
    // We'll show placeholder until we have mind point data
    const current = 6;
    const max = 6;
    const percentage = 100;

    return `
      <div class="space-y-1">
        <div class="flex items-center justify-between text-sm">
          <span class="opacity-70">Mind Points</span>
          <span class="font-mono font-semibold text-blue-400">${current} / ${max}</span>
        </div>
        <div class="h-2 bg-surface-2 rounded-full overflow-hidden">
          <div class="h-full bg-blue-500 transition-all duration-300" style="width: ${percentage}%"></div>
        </div>
      </div>
    `;
  }

  /**
   * Render movement status
   * @param {Object} turnState
   * @returns {string}
   */
  renderMovementStatus(turnState) {
    if (!turnState) {
      return '';
    }

    const remaining = turnState.movementRemaining || 0;
    const total = turnState.movementTotal || 0;

    if (!turnState.movementDiceRolled) {
      return `
        <div class="p-2 bg-surface-2 rounded text-sm opacity-70 text-center">
          Movement not rolled
        </div>
      `;
    }

    return `
      <div class="space-y-1">
        <div class="flex items-center justify-between text-sm">
          <span class="opacity-70">Movement</span>
          <span class="font-mono font-semibold text-green-400">${remaining} / ${total}</span>
        </div>
      </div>
    `;
  }

  /**
   * Render equipment section
   * @param {Object} hero
   * @returns {string}
   */
  renderEquipment(hero) {
    // Equipment data not yet in EntityLite, show placeholder
    return `
      <div class="space-y-2">
        <h3 class="text-sm font-semibold opacity-80">Equipment</h3>
        <div class="space-y-1 text-sm">
          <div class="flex items-center justify-between p-2 bg-surface-2 rounded">
            <span class="opacity-70">Weapon:</span>
            <span class="font-medium">Dagger (1d)</span>
          </div>
          <div class="flex items-center justify-between p-2 bg-surface-2 rounded">
            <span class="opacity-70">Armor:</span>
            <span class="font-medium opacity-50">None</span>
          </div>
          <div class="flex items-center justify-between p-2 bg-surface-2 rounded">
            <span class="opacity-70">Shield:</span>
            <span class="font-medium opacity-50">None</span>
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Render spells section for Wizard/Elf
   * @param {string} heroClass
   * @returns {string}
   */
  renderSpells(heroClass) {
    if (heroClass !== 'Wizard' && heroClass !== 'Elf') {
      return '';
    }

    return `
      <div class="space-y-2">
        <h3 class="text-sm font-semibold opacity-80">Spells</h3>
        <div class="space-y-1 text-sm">
          <div class="p-2 bg-surface-2 rounded opacity-70">
            <div class="flex items-center justify-between">
              <span>Heal Body</span>
              <span class="text-xs">3/3</span>
            </div>
          </div>
          <div class="p-2 bg-surface-2 rounded opacity-70">
            <div class="flex items-center justify-between">
              <span>Ball of Flame</span>
              <span class="text-xs">3/3</span>
            </div>
          </div>
          <div class="p-2 bg-surface-2 rounded opacity-70">
            <div class="flex items-center justify-between">
              <span>Swift Wind</span>
              <span class="text-xs">3/3</span>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Render abilities section for special classes
   * @param {string} heroClass
   * @returns {string}
   */
  renderAbilities(heroClass) {
    const abilities = this.getClassAbilities(heroClass);
    if (abilities.length === 0) {
      return '';
    }

    const abilitiesList = abilities.map(ability => `
      <div class="p-2 bg-surface-2 rounded opacity-70">
        <div class="font-medium">${ability.name}</div>
        <div class="text-xs opacity-70">${ability.description}</div>
      </div>
    `).join('');

    return `
      <div class="space-y-2">
        <h3 class="text-sm font-semibold opacity-80">Abilities</h3>
        <div class="space-y-1 text-sm">
          ${abilitiesList}
        </div>
      </div>
    `;
  }

  /**
   * Get class-specific abilities
   * @param {string} heroClass
   * @returns {Array}
   */
  getClassAbilities(heroClass) {
    const abilities = {
      'Barbarian': [
        { name: 'Berserker', description: 'Extra attack dice in melee' }
      ],
      'Dwarf': [
        { name: 'Trap Detection', description: 'Detect traps automatically' }
      ],
      'Berserker': [
        { name: 'Rage', description: 'Extra attack power when wounded' }
      ],
      'Monk': [
        { name: 'Martial Arts', description: 'Unarmed combat bonus' }
      ],
      'Explorer': [
        { name: 'Pathfinder', description: 'Navigate difficult terrain' }
      ],
      'Druid': [
        { name: 'Nature Magic', description: 'Commune with nature' }
      ]
    };

    return abilities[heroClass] || [];
  }

  /**
   * Update stats when patches are received
   * @param {Object} patch
   */
  handlePatch(patch) {
    if (!this.currentHeroId) return;

    // Update on relevant patches
    if (patch.type === 'EntityUpdated' ||
        patch.type === 'TurnStateChanged' ||
        patch.type === 'HeroActionResult') {
      this.updateFromSnapshot(this.gameState.snapshot);
    }
  }
}
