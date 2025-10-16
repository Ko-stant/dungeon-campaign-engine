/**
 * Entity Inspection Modal Controller
 * Handles displaying hero/monster stats when clicked
 */

export class EntityModalController {
  constructor(gameState) {
    this.gameState = gameState;
    this.modal = document.getElementById('entityModal');
    this.modalTitle = document.getElementById('entityModalTitle');
    this.modalContent = document.getElementById('entityModalContent');

    this.init();
  }

  init() {
    // Close modal on Escape key
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && this.isOpen()) {
        this.close();
      }
    });

    // Make close function globally available for onclick handler
    window.closeEntityModal = () => this.close();
  }

  isOpen() {
    return this.modal && this.modal.style.display !== 'none';
  }

  open() {
    if (this.modal) {
      this.modal.style.display = 'flex';
    }
  }

  close() {
    if (this.modal) {
      this.modal.style.display = 'none';
    }
  }

  /**
   * Show hero inspection modal
   * @param {Object} hero - Hero entity data
   */
  showHero(hero) {
    if (!this.modal) return;

    this.modalTitle.textContent = 'Hero Details';

    // Extract class from tags (e.g., ["barbarian"] -> "Barbarian")
    const heroClass = hero.tags && hero.tags.length > 0
      ? hero.tags[0].charAt(0).toUpperCase() + hero.tags[0].slice(1)
      : 'Unknown Class';

    // Build hero stats HTML
    this.modalContent.innerHTML = `
      <div class="space-y-4">
        <!-- Hero Identity -->
        <div class="flex items-center gap-3 p-3 bg-surface-2 rounded-md">
          <div class="w-16 h-16 rounded-md bg-blue-600/20 border border-blue-600/40 flex items-center justify-center text-3xl">
            ${this.getHeroIcon(heroClass)}
          </div>
          <div>
            <div class="text-xl font-semibold">${hero.id || 'Hero'}</div>
            <div class="text-sm opacity-70">${heroClass}</div>
          </div>
        </div>

        <!-- Stats Grid -->
        <div class="grid grid-cols-2 gap-3">
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Body Points</div>
            <div class="text-lg font-bold text-red-400">${hero.hp?.current || 0} / ${hero.hp?.max || 0}</div>
          </div>
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Mind Points</div>
            <div class="text-lg font-bold text-blue-400">${hero.mindPoints?.current || 0} / ${hero.mindPoints?.max || 0}</div>
          </div>
        </div>

        <!-- Equipment -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold opacity-80">Equipment</h3>
          <div class="space-y-1 text-sm">
            ${this.renderEquipment(hero.equipment)}
          </div>
        </div>

        ${this.renderHeroSpecial(hero)}
      </div>
    `;

    this.open();
  }

  /**
   * Show monster inspection modal
   * @param {Object} monster - Monster entity data
   */
  showMonster(monster) {
    if (!this.modal) return;

    this.modalTitle.textContent = 'Monster Details';

    // Get monster turn state to find fixed movement
    const snapshot = this.gameState.snapshot;
    const monsterTurnState = snapshot?.monsterTurnStates?.[monster.id];
    const fixedMovement = monsterTurnState?.fixedMovement;

    // Build monster stats HTML
    this.modalContent.innerHTML = `
      <div class="space-y-4">
        <!-- Monster Identity -->
        <div class="flex items-center gap-3 p-3 bg-red-900/20 rounded-md border border-red-600/40">
          <div class="w-16 h-16 rounded-md bg-red-600/20 border border-red-600/40 flex items-center justify-center text-3xl">
            ${this.getMonsterIcon(monster.type)}
          </div>
          <div>
            <div class="text-xl font-semibold">${monster.type || 'Monster'}</div>
            <div class="text-sm opacity-70">Level ${monster.level || 1}</div>
          </div>
        </div>

        <!-- Stats Grid -->
        <div class="grid grid-cols-2 gap-3">
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Body Points</div>
            <div class="text-lg font-bold text-red-400">${monster.body || 0} / ${monster.maxBody || 0}</div>
          </div>
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Mind Points</div>
            <div class="text-lg font-bold text-blue-400">${monster.mind || 0}</div>
          </div>
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Movement</div>
            <div class="text-lg font-bold">${fixedMovement ? fixedMovement + ' squares' : 'Unknown'}</div>
          </div>
          <div class="p-3 bg-surface-2 rounded-md">
            <div class="text-xs opacity-70 mb-1">Attack Dice</div>
            <div class="text-lg font-bold text-orange-400">${monster.attackDice || 0}d6</div>
          </div>
          <div class="p-3 bg-surface-2 rounded-md col-span-2">
            <div class="text-xs opacity-70 mb-1">Defense Dice</div>
            <div class="text-lg font-bold text-yellow-400">${monster.defenseDice || 0}d6</div>
          </div>
        </div>

        ${this.renderMonsterAbilities(monster)}
      </div>
    `;

    this.open();
  }

  getHeroIcon(heroClass) {
    const icons = {
      'Wizard': '🧙',
      'Barbarian': '⚔️',
      'Dwarf': '🪓',
      'Elf': '🏹'
    };
    return icons[heroClass] || '🧑';
  }

  getMonsterIcon(monsterType) {
    const icons = {
      'Goblin': '👺',
      'Orc': '👹',
      'Skeleton': '💀',
      'Zombie': '🧟',
      'Mummy': '🧟‍♂️',
      'Chaos Warrior': '⚔️',
      'Fimir': '🐊',
      'Gargoyle': '🦅'
    };
    return icons[monsterType] || '👾';
  }

  renderEquipment(equipment) {
    if (!equipment) {
      return '<div class="text-sm opacity-70">No equipment</div>';
    }

    let html = '';
    if (equipment.weapon) {
      html += `<div class="flex items-center justify-between p-2 bg-surface-2 rounded">
        <span class="opacity-70">Weapon:</span>
        <span class="font-medium">${equipment.weapon.name} (${equipment.weapon.attackDice}d)</span>
      </div>`;
    }
    if (equipment.armor) {
      html += `<div class="flex items-center justify-between p-2 bg-surface-2 rounded">
        <span class="opacity-70">Armor:</span>
        <span class="font-medium">${equipment.armor.name} (${equipment.armor.defenseDice}d)</span>
      </div>`;
    }
    if (equipment.shield) {
      html += `<div class="flex items-center justify-between p-2 bg-surface-2 rounded">
        <span class="opacity-70">Shield:</span>
        <span class="font-medium">${equipment.shield.name}</span>
      </div>`;
    }
    return html || '<div class="text-sm opacity-70">No equipment</div>';
  }

  renderHeroSpecial(hero) {
    // TODO: Render spells for Wizard/Elf, abilities for other classes
    return '';
  }

  renderMonsterAbilities(monster) {
    if (!monster.abilities || monster.abilities.length === 0) {
      return '';
    }

    return `
      <div class="space-y-2">
        <h3 class="text-sm font-semibold opacity-80">Special Abilities</h3>
        <div class="text-sm opacity-70">
          ${monster.abilities.map(ability => `<div>• ${ability}</div>`).join('')}
        </div>
      </div>
    `;
  }
}
