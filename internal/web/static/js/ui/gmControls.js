/**
 * GM Controls Module
 * Handles all Game Master UI interactions including monster control,
 * turn phase management, and event logging
 */

export class GMControlsController {
  constructor(gameState) {
    this.gameState = gameState;
    this.selectedMonsterID = null;
    this.selectedMonsterData = null;
    this.eventLog = [];
    this.movementMode = false;
    this.attackMode = false;
    this.availableMovementTiles = [];
    this.availableAttackTargets = [];

    this.initializeElements();
    this.attachEventListeners();
  }

  initializeElements() {
    // Monster Control Elements
    this.spawnMonsterBtn = document.getElementById('spawn-monster-btn');
    this.monsterList = document.getElementById('monster-list');
    this.selectedMonsterPanel = document.getElementById('selected-monster-panel');
    this.selectedMonsterInfo = {
      type: document.getElementById('selected-monster-type'),
      position: document.getElementById('selected-monster-position'),
      body: document.getElementById('selected-monster-body'),
      movement: document.getElementById('selected-monster-movement'),
      hasMoved: document.getElementById('selected-monster-has-moved'),
      actionTaken: document.getElementById('selected-monster-action-taken'),
    };
    this.selectedMonsterTurnState = document.getElementById('selected-monster-turn-state');
    this.selectedMonsterActions = document.getElementById('selected-monster-actions');
    this.deselectMonsterBtn = document.getElementById('deselect-monster-btn');
    this.monsterMoveBtn = document.getElementById('monster-move-btn');
    this.monsterAttackBtn = document.getElementById('monster-attack-btn');

    // Monster Spawn Modal Elements
    this.monsterSpawnModal = document.getElementById('monster-spawn-modal');
    this.spawnMonsterType = document.getElementById('spawn-monster-type');
    this.spawnPositionX = document.getElementById('spawn-position-x');
    this.spawnPositionY = document.getElementById('spawn-position-y');
    this.spawnAsWandering = document.getElementById('spawn-as-wandering');
    this.confirmSpawnBtn = document.getElementById('confirm-spawn-btn');
    this.cancelSpawnBtn = document.getElementById('cancel-spawn-btn');
    this.closeSpawnModalBtn = document.getElementById('close-spawn-modal-btn');

    // Turn Phase Elements
    this.gmPhaseIndicator = document.getElementById('gm-phase-indicator');
    this.completeGmTurnBtn = document.getElementById('complete-gm-turn-btn');
    this.confirmElectionBtn = document.getElementById('confirm-election-btn');

    // Event Log Elements
    this.gmEventLog = document.getElementById('gm-event-log');
    this.clearEventLogBtn = document.getElementById('clear-event-log-btn');

    // Detail Pane Tabs
    this.gmDetailTabs = document.querySelectorAll('.gm-detail-tab');
    this.gmDetailContents = document.querySelectorAll('.gm-detail-content');
    this.monsterStatsContent = document.getElementById('monster-stats-content');
  }

  attachEventListeners() {
    // Monster spawn modal
    if (this.spawnMonsterBtn) {
      this.spawnMonsterBtn.addEventListener('click', () => this.openSpawnModal());
    }
    if (this.confirmSpawnBtn) {
      this.confirmSpawnBtn.addEventListener('click', () => this.confirmMonsterSpawn());
    }
    if (this.cancelSpawnBtn) {
      this.cancelSpawnBtn.addEventListener('click', () => this.closeSpawnModal());
    }
    if (this.closeSpawnModalBtn) {
      this.closeSpawnModalBtn.addEventListener('click', () => this.closeSpawnModal());
    }

    // Monster selection
    if (this.deselectMonsterBtn) {
      this.deselectMonsterBtn.addEventListener('click', () => this.deselectMonster());
    }
    if (this.monsterMoveBtn) {
      this.monsterMoveBtn.addEventListener('click', () => {
        if (this.movementMode) {
          this.cancelMovement();
        } else {
          this.initiateMonsterMove();
        }
      });
    }
    if (this.monsterAttackBtn) {
      this.monsterAttackBtn.addEventListener('click', () => {
        if (this.attackMode) {
          this.cancelAttack();
        } else {
          this.initiateMonsterAttack();
        }
      });
    }

    // Turn phase controls
    if (this.completeGmTurnBtn) {
      this.completeGmTurnBtn.addEventListener('click', () => this.completeGMTurn());
    }
    if (this.confirmElectionBtn) {
      this.confirmElectionBtn.addEventListener('click', () => this.confirmElection());
    }

    // Event log
    if (this.clearEventLogBtn) {
      this.clearEventLogBtn.addEventListener('click', () => this.clearEventLog());
    }

    // Detail pane tabs
    this.gmDetailTabs.forEach(tab => {
      tab.addEventListener('click', (e) => this.switchDetailTab(e.target.dataset.tab));
    });

    // Modal backdrop click to close
    if (this.monsterSpawnModal) {
      this.monsterSpawnModal.addEventListener('click', (e) => {
        if (e.target === this.monsterSpawnModal) {
          this.closeSpawnModal();
        }
      });
    }

    // Enable spawn button when all fields are filled
    if (this.spawnMonsterType && this.spawnPositionX && this.spawnPositionY) {
      [this.spawnMonsterType, this.spawnPositionX, this.spawnPositionY].forEach(el => {
        el.addEventListener('input', () => this.validateSpawnForm());
      });
    }
  }

  /**
   * Monster Spawn Modal Functions
   */
  openSpawnModal() {
    if (this.monsterSpawnModal) {
      this.monsterSpawnModal.classList.remove('hidden');
      this.monsterSpawnModal.classList.add('flex');
      this.resetSpawnForm();
    }
  }

  closeSpawnModal() {
    if (this.monsterSpawnModal) {
      this.monsterSpawnModal.classList.add('hidden');
      this.monsterSpawnModal.classList.remove('flex');
      this.resetSpawnForm();
    }
  }

  resetSpawnForm() {
    if (this.spawnMonsterType) this.spawnMonsterType.value = '';
    if (this.spawnPositionX) this.spawnPositionX.value = '';
    if (this.spawnPositionY) this.spawnPositionY.value = '';
    if (this.spawnAsWandering) this.spawnAsWandering.checked = false;
    this.validateSpawnForm();
  }

  validateSpawnForm() {
    const isValid =
      this.spawnMonsterType?.value &&
      this.spawnPositionX?.value !== '' &&
      this.spawnPositionY?.value !== '';

    if (this.confirmSpawnBtn) {
      this.confirmSpawnBtn.disabled = !isValid;
    }
  }

  confirmMonsterSpawn() {
    const monsterType = this.spawnMonsterType?.value;
    const x = parseInt(this.spawnPositionX?.value, 10);
    const y = parseInt(this.spawnPositionY?.value, 10);
    const isWandering = this.spawnAsWandering?.checked || false;

    if (!monsterType || isNaN(x) || isNaN(y)) {
      console.error('Invalid spawn parameters');
      return;
    }

    // Send spawn intent via WebSocket
    this.sendIntent({
      kind: 'SpawnMonster',
      monsterType: monsterType,
      position: { x, y },
      isWandering: isWandering,
    });

    this.logEvent(`Spawned ${monsterType} at (${x}, ${y})${isWandering ? ' (wandering)' : ''}`);
    this.closeSpawnModal();
  }

  /**
   * Monster Selection Functions
   */
  selectMonster(monsterID) {
    this.selectedMonsterID = monsterID;
    const monster = this.gameState.monsters.get(monsterID);

    if (!monster) {
      console.error('Monster not found:', monsterID);
      return;
    }

    this.selectedMonsterData = monster;
    this.updateSelectedMonsterPanel();
    this.showSelectedMonsterPanel();
  }

  deselectMonster() {
    this.selectedMonsterID = null;
    this.selectedMonsterData = null;
    this.hideSelectedMonsterPanel();
  }

  updateSelectedMonsterPanel() {
    if (!this.selectedMonsterData) return;

    const monster = this.selectedMonsterData;

    // Update monster info
    if (this.selectedMonsterInfo.type) {
      this.selectedMonsterInfo.type.textContent = monster.type || 'Unknown';
    }
    if (this.selectedMonsterInfo.position) {
      this.selectedMonsterInfo.position.textContent =
        `(${monster.tile?.x ?? '?'}, ${monster.tile?.y ?? '?'})`;
    }
    if (this.selectedMonsterInfo.body) {
      this.selectedMonsterInfo.body.textContent =
        `${monster.body ?? 0} / ${monster.maxBody ?? 0}`;
    }

    // Update turn state if in GM phase
    const currentPhase = this.gameState.snapshot?.turnPhase;
    if (currentPhase === 'gm_phase') {
      this.showMonsterTurnState();
      if (this.selectedMonsterInfo.movement) {
        this.selectedMonsterInfo.movement.textContent = monster.movement ?? '0';
      }
      if (this.selectedMonsterInfo.hasMoved) {
        this.selectedMonsterInfo.hasMoved.textContent = monster.hasMoved ? 'Yes' : 'No';
      }
      if (this.selectedMonsterInfo.actionTaken) {
        this.selectedMonsterInfo.actionTaken.textContent = monster.actionTaken ? 'Yes' : 'No';
      }

      // Enable/disable action buttons based on monster state
      if (this.monsterMoveBtn) {
        this.monsterMoveBtn.disabled = monster.hasMoved || false;
      }
      if (this.monsterAttackBtn) {
        this.monsterAttackBtn.disabled = monster.actionTaken || false;
      }
    } else {
      this.hideMonsterTurnState();
    }
  }

  showSelectedMonsterPanel() {
    if (this.selectedMonsterPanel) {
      this.selectedMonsterPanel.classList.remove('hidden');
    }
  }

  hideSelectedMonsterPanel() {
    if (this.selectedMonsterPanel) {
      this.selectedMonsterPanel.classList.add('hidden');
    }
  }

  showMonsterTurnState() {
    if (this.selectedMonsterTurnState) {
      this.selectedMonsterTurnState.classList.remove('hidden');
    }
    if (this.selectedMonsterActions) {
      this.selectedMonsterActions.classList.remove('hidden');
    }
  }

  hideMonsterTurnState() {
    if (this.selectedMonsterTurnState) {
      this.selectedMonsterTurnState.classList.add('hidden');
    }
    if (this.selectedMonsterActions) {
      this.selectedMonsterActions.classList.add('hidden');
    }
  }

  /**
   * Monster List Functions
   */
  updateMonsterList() {
    if (!this.monsterList) return;

    const monsters = Array.from(this.gameState.monsters.values());

    if (monsters.length === 0) {
      this.monsterList.innerHTML = `
        <div class="text-sm text-slate-400 text-center py-4">
          No monsters on the board
        </div>
      `;
      return;
    }

    this.monsterList.innerHTML = monsters.map(monster => `
      <div
        class="monster-item p-2 bg-slate-800/50 rounded border border-border/40 hover:border-amber-500/50 cursor-pointer transition-colors ${this.selectedMonsterID === monster.id ? 'border-amber-500 bg-amber-900/20' : ''}"
        data-monster-id="${monster.id}"
      >
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm font-medium text-slate-200">${monster.type}</span>
          <span class="text-xs text-slate-400">(${monster.tile?.x ?? '?'}, ${monster.tile?.y ?? '?'})</span>
        </div>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span>Body: ${monster.body}/${monster.maxBody}</span>
          <span>Atk: ${monster.attackDice ?? 0}d</span>
          <span>Def: ${monster.defenseDice ?? 0}d</span>
        </div>
      </div>
    `).join('');

    // Attach click handlers to monster items
    this.monsterList.querySelectorAll('.monster-item').forEach(item => {
      item.addEventListener('click', (e) => {
        const monsterID = e.currentTarget.dataset.monsterId;
        this.selectMonster(monsterID);
      });
    });
  }

  /**
   * Monster Action Functions
   */
  initiateMonsterMove() {
    if (!this.selectedMonsterID) {
      console.error('No monster selected');
      return;
    }

    if (!this.selectedMonsterData) {
      console.error('No monster data available');
      return;
    }

    // Enter movement mode
    this.movementMode = true;
    this.attackMode = false;

    // Calculate available movement tiles
    this.availableMovementTiles = this.calculateMovementTiles(
      this.selectedMonsterData.tile,
      this.selectedMonsterData.movement || 0
    );

    // Highlight available tiles on the board
    this.highlightMovementTiles();

    // Update UI
    if (this.monsterMoveBtn) {
      this.monsterMoveBtn.textContent = 'Cancel Move';
      this.monsterMoveBtn.className = 'px-3 py-1 bg-red-600 hover:bg-red-700 text-white rounded transition-colors text-sm';
    }

    this.logEvent(`Movement mode active for ${this.selectedMonsterData.type} (${this.selectedMonsterData.movement} tiles)`);
  }

  cancelMovement() {
    this.movementMode = false;
    this.availableMovementTiles = [];
    this.clearMovementHighlights();

    // Restore button - remove the onclick handler and rely on the original event listener
    if (this.monsterMoveBtn) {
      this.monsterMoveBtn.textContent = 'Move';
      this.monsterMoveBtn.className = 'px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors text-sm disabled:opacity-50 disabled:cursor-not-allowed';
    }

    this.logEvent('Movement mode cancelled');
  }

  calculateMovementTiles(startTile, movementRange) {
    if (!startTile || movementRange <= 0) {
      return [];
    }

    const tiles = [];
    const startX = startTile.x;
    const startY = startTile.y;

    // Simple grid-based movement - all tiles within movement range
    // This is a basic implementation; could be enhanced with pathfinding
    for (let dx = -movementRange; dx <= movementRange; dx++) {
      for (let dy = -movementRange; dy <= movementRange; dy++) {
        const distance = Math.abs(dx) + Math.abs(dy); // Manhattan distance
        if (distance > 0 && distance <= movementRange) {
          const x = startX + dx;
          const y = startY + dy;

          // Basic bounds checking
          const snapshot = this.gameState.snapshot;
          if (snapshot && x >= 0 && y >= 0 && x < snapshot.mapWidth && y < snapshot.mapHeight) {
            // Check if tile is not occupied by another entity
            if (!this.isTileOccupied(x, y)) {
              tiles.push({ x, y });
            }
          }
        }
      }
    }

    return tiles;
  }

  isTileOccupied(x, y) {
    // Check if any hero is on this tile
    const snapshot = this.gameState.snapshot;
    if (snapshot && snapshot.entities) {
      for (const entity of snapshot.entities) {
        if (entity.tile && entity.tile.x === x && entity.tile.y === y) {
          return true;
        }
      }
    }

    // Check if any monster is on this tile (except the selected one)
    const monsters = Array.from(this.gameState.monsters.values());
    for (const monster of monsters) {
      if (monster.id !== this.selectedMonsterID && monster.tile && monster.tile.x === x && monster.tile.y === y) {
        return true;
      }
    }

    return false;
  }

  highlightMovementTiles() {
    this.clearMovementHighlights();

    // Highlight monster's current position
    if (this.selectedMonsterData && this.selectedMonsterData.tile) {
      const currentTile = document.querySelector(
        `[data-x="${this.selectedMonsterData.tile.x}"][data-y="${this.selectedMonsterData.tile.y}"]`
      );
      if (currentTile) {
        currentTile.classList.add('monster-selected-tile');
      }
    }

    // Highlight available movement tiles
    this.availableMovementTiles.forEach(tile => {
      const tileElement = document.querySelector(`[data-x="${tile.x}"][data-y="${tile.y}"]`);
      if (tileElement) {
        tileElement.classList.add('monster-movement-available');
      }
    });
  }

  clearMovementHighlights() {
    document.querySelectorAll('.monster-movement-available').forEach(tile => {
      tile.classList.remove('monster-movement-available');
    });
    document.querySelectorAll('.monster-selected-tile').forEach(tile => {
      tile.classList.remove('monster-selected-tile');
    });
  }

  handleMovementTileClick(x, y) {
    if (!this.movementMode) return false;

    // Check if clicked tile is available for movement
    const isAvailable = this.availableMovementTiles.some(tile => tile.x === x && tile.y === y);
    if (!isAvailable) return false;

    // Send move intent to server
    this.sendIntent({
      kind: 'MoveMonster',
      monsterID: this.selectedMonsterID,
      targetX: x,
      targetY: y,
    });

    this.logEvent(`Moving ${this.selectedMonsterData?.type} to (${x}, ${y})`);

    // Exit movement mode
    this.cancelMovement();

    return true;
  }

  initiateMonsterAttack() {
    if (!this.selectedMonsterID) {
      console.error('No monster selected');
      return;
    }

    if (!this.selectedMonsterData) {
      console.error('No monster data available');
      return;
    }

    // Enter attack mode
    this.attackMode = true;
    this.movementMode = false;

    // Find available attack targets (heroes within range 1)
    this.availableAttackTargets = this.calculateAttackTargets(
      this.selectedMonsterData.tile
    );

    // Highlight available targets on the board
    this.highlightAttackTargets();

    // Update UI
    if (this.monsterAttackBtn) {
      this.monsterAttackBtn.textContent = 'Cancel Attack';
      this.monsterAttackBtn.className = 'px-3 py-1 bg-red-600 hover:bg-red-700 text-white rounded transition-colors text-sm';
    }

    this.logEvent(`Attack mode active for ${this.selectedMonsterData.type} - select target`);
  }

  cancelAttack() {
    this.attackMode = false;
    this.availableAttackTargets = [];
    this.clearAttackHighlights();

    // Restore button
    if (this.monsterAttackBtn) {
      this.monsterAttackBtn.textContent = 'Attack';
      this.monsterAttackBtn.className = 'px-3 py-1 bg-red-600 hover:bg-red-700 text-white rounded transition-colors text-sm disabled:opacity-50 disabled:cursor-not-allowed';
    }

    this.logEvent('Attack mode cancelled');
  }

  calculateAttackTargets(monsterTile) {
    if (!monsterTile) {
      return [];
    }

    const targets = [];
    const snapshot = this.gameState.snapshot;

    if (!snapshot || !snapshot.entities) {
      return [];
    }

    // Find all heroes within attack range (adjacent tiles, range 1)
    for (const entity of snapshot.entities) {
      if (entity.kind === 'hero' && entity.tile) {
        const dx = Math.abs(entity.tile.x - monsterTile.x);
        const dy = Math.abs(entity.tile.y - monsterTile.y);
        const distance = dx + dy; // Manhattan distance

        // Monsters can attack heroes that are adjacent (distance 1)
        if (distance === 1) {
          targets.push({
            entityID: entity.id,
            x: entity.tile.x,
            y: entity.tile.y,
          });
        }
      }
    }

    return targets;
  }

  highlightAttackTargets() {
    this.clearAttackHighlights();

    // Highlight monster's current position
    if (this.selectedMonsterData && this.selectedMonsterData.tile) {
      const currentTile = document.querySelector(
        `[data-x="${this.selectedMonsterData.tile.x}"][data-y="${this.selectedMonsterData.tile.y}"]`
      );
      if (currentTile) {
        currentTile.classList.add('monster-selected-tile');
      }
    }

    // Highlight available attack targets
    this.availableAttackTargets.forEach(target => {
      const tileElement = document.querySelector(`[data-x="${target.x}"][data-y="${target.y}"]`);
      if (tileElement) {
        tileElement.classList.add('monster-attack-target');
      }
    });
  }

  clearAttackHighlights() {
    document.querySelectorAll('.monster-attack-target').forEach(tile => {
      tile.classList.remove('monster-attack-target');
    });
    // Also clear selected tile highlight
    document.querySelectorAll('.monster-selected-tile').forEach(tile => {
      tile.classList.remove('monster-selected-tile');
    });
  }

  handleAttackTargetClick(x, y) {
    if (!this.attackMode) return false;

    // Check if clicked tile is an available target
    const target = this.availableAttackTargets.find(t => t.x === x && t.y === y);
    if (!target) return false;

    // Send attack intent to server
    this.sendIntent({
      kind: 'MonsterAttack',
      monsterID: this.selectedMonsterID,
      targetEntityID: target.entityID,
    });

    this.logEvent(`${this.selectedMonsterData?.type} attacking hero at (${x}, ${y})`);

    // Exit attack mode
    this.cancelAttack();

    return true;
  }

  /**
   * Turn Phase Functions
   */
  updateTurnPhase(snapshot) {
    if (!snapshot) return;

    const phase = snapshot.turnPhase;
    const cycle = snapshot.cycleNumber;

    // Update phase indicator
    if (this.gmPhaseIndicator) {
      this.gmPhaseIndicator.dataset.phase = phase || 'quest_setup';
      this.gmPhaseIndicator.dataset.cycle = cycle || '0';

      // Update display based on phase
      const phaseDisplay = this.gmPhaseIndicator.querySelector('[data-phase-display]');
      if (phaseDisplay) {
        switch (phase) {
          case 'quest_setup':
            phaseDisplay.textContent = 'Quest Setup';
            phaseDisplay.className = 'text-sm font-medium text-purple-400';
            break;
          case 'hero_election':
            phaseDisplay.textContent = 'Hero Election';
            phaseDisplay.className = 'text-sm font-medium text-blue-400';
            break;
          case 'hero_active':
            phaseDisplay.textContent = 'Hero Turn';
            phaseDisplay.className = 'text-sm font-medium text-green-400';
            break;
          case 'gm_phase':
            phaseDisplay.textContent = 'GM Phase';
            phaseDisplay.className = 'text-sm font-medium text-amber-400';
            break;
          default:
            phaseDisplay.textContent = 'Unknown Phase';
            phaseDisplay.className = 'text-sm font-medium text-slate-400';
        }
      }
    }

    // Show/hide Complete GM Turn button
    if (this.completeGmTurnBtn) {
      if (phase === 'gm_phase') {
        this.completeGmTurnBtn.style.display = 'block';
      } else {
        this.completeGmTurnBtn.style.display = 'none';
      }
    }

    // Update elected player section during hero_election phase
    const electedPlayerSection = document.getElementById('elected-player-section');
    const electedPlayerNameEl = document.getElementById('elected-player-name');

    if (phase === 'hero_election' && snapshot.electedPlayerID) {
      // Show elected player section
      if (electedPlayerSection) {
        electedPlayerSection.classList.remove('hidden');
      }

      // Find elected player name
      const electedPlayer = snapshot.players?.find(p => p.id === snapshot.electedPlayerID);
      if (electedPlayerNameEl && electedPlayer) {
        electedPlayerNameEl.textContent = electedPlayer.name || 'Unknown Hero';
      }
    } else {
      // Hide elected player section
      if (electedPlayerSection) {
        electedPlayerSection.classList.add('hidden');
      }
    }

    // Update monster panel based on phase
    if (this.selectedMonsterID) {
      this.updateSelectedMonsterPanel();
    }
  }

  completeGMTurn() {
    this.sendIntent({
      kind: 'CompleteGMTurn',
    });
    this.logEvent('GM turn completed');
  }

  confirmElection() {
    this.sendIntent({
      kind: 'ConfirmElectionAndStartTurn',
    });
    this.logEvent('Election confirmed, starting hero turn');
  }

  /**
   * Event Log Functions
   */
  logEvent(message, type = 'info') {
    const timestamp = new Date().toLocaleTimeString();
    const event = {
      message,
      type,
      timestamp,
    };

    this.eventLog.push(event);
    this.addEventToLog(event);
  }

  addEventToLog(event) {
    if (!this.gmEventLog) return;

    // Remove placeholder if present
    const placeholder = this.gmEventLog.querySelector('.text-center');
    if (placeholder) {
      placeholder.remove();
    }

    const eventElement = document.createElement('div');
    eventElement.className = `event-log-item px-2 py-1 rounded text-xs ${this.getEventClass(event.type)}`;
    eventElement.innerHTML = `
      <span class="opacity-60">[${event.timestamp}]</span>
      <span class="ml-2">${event.message}</span>
    `;

    this.gmEventLog.appendChild(eventElement);

    // Auto-scroll to bottom
    this.gmEventLog.scrollTop = this.gmEventLog.scrollHeight;

    // Limit log size to last 100 entries
    const items = this.gmEventLog.querySelectorAll('.event-log-item');
    if (items.length > 100) {
      items[0].remove();
    }
  }

  getEventClass(type) {
    switch (type) {
      case 'error':
        return 'text-red-400';
      case 'warning':
        return 'text-yellow-400';
      case 'success':
        return 'text-green-400';
      default:
        return 'text-slate-300';
    }
  }

  clearEventLog() {
    this.eventLog = [];
    if (this.gmEventLog) {
      this.gmEventLog.innerHTML = `
        <div class="text-slate-500 text-center py-4">
          No events yet
        </div>
      `;
    }
  }

  /**
   * Detail Pane Tab Functions
   */
  switchDetailTab(tabName) {
    // Update tab buttons
    this.gmDetailTabs.forEach(tab => {
      if (tab.dataset.tab === tabName) {
        tab.classList.add('border-amber-500', 'text-amber-400');
        tab.classList.remove('border-transparent', 'text-slate-400');
      } else {
        tab.classList.remove('border-amber-500', 'text-amber-400');
        tab.classList.add('border-transparent', 'text-slate-400');
      }
    });

    // Update content visibility
    this.gmDetailContents.forEach(content => {
      const contentId = content.id.replace('gm-tab-', '');
      if (contentId === tabName) {
        content.classList.remove('hidden');
      } else {
        content.classList.add('hidden');
      }
    });
  }

  /**
   * Update from snapshot
   */
  updateFromSnapshot(snapshot) {
    if (!snapshot) return;

    this.updateTurnPhase(snapshot);
    this.updateMonsterList();
    this.updateQuestData(snapshot);
  }

  /**
   * Update quest data in detail pane
   */
  updateQuestData(snapshot) {
    // Update quest notes tab
    const questDetailTitle = document.getElementById('quest-detail-title');
    const questDetailContent = document.getElementById('quest-detail-content');

    if (questDetailTitle && snapshot.questName) {
      questDetailTitle.textContent = snapshot.questName;
    }

    if (questDetailContent) {
      let html = '';

      // Quest description
      if (snapshot.questDescription) {
        html += `<p class="text-slate-300 mb-3">${snapshot.questDescription}</p>`;
      }

      // Quest objectives
      if (snapshot.questObjectives && snapshot.questObjectives.length > 0) {
        html += `
          <div class="mb-3">
            <h4 class="text-sm font-semibold text-amber-400 mb-2">Objectives:</h4>
            <ul class="list-disc list-inside space-y-1 text-slate-300 text-sm">
              ${snapshot.questObjectives.map(obj => `<li>${obj}</li>`).join('')}
            </ul>
          </div>
        `;
      }

      // GM Notes (GM-only)
      if (snapshot.questGMNotes) {
        html += `
          <div class="mt-4 p-3 bg-purple-900/20 border border-purple-500/30 rounded">
            <h4 class="text-sm font-semibold text-purple-400 mb-2">GM Notes:</h4>
            <div class="text-xs text-slate-300 whitespace-pre-wrap">${snapshot.questGMNotes}</div>
          </div>
        `;
      }

      // Player-visible notes
      if (snapshot.questNotes) {
        html += `
          <div class="mt-3 p-3 bg-blue-900/20 border border-blue-500/30 rounded">
            <h4 class="text-sm font-semibold text-blue-400 mb-2">Player Notes:</h4>
            <div class="text-xs text-slate-300 whitespace-pre-wrap">${snapshot.questNotes}</div>
          </div>
        `;
      }

      if (html) {
        questDetailContent.innerHTML = html;
      }
    }

    // Update quest rules tab
    const questRulesContent = document.getElementById('quest-rules-content');
    if (questRulesContent) {
      // For now, show placeholder with custom dice rules
      // Later this could be populated from quest metadata
      questRulesContent.innerHTML = `
        <p class="text-slate-300 mb-3">Standard HeroQuest rules apply with the following modifications:</p>
        <div class="p-3 bg-slate-900/50 rounded border border-slate-700">
          <h4 class="text-amber-400 font-semibold mb-2">Custom Dice Rules (Optional)</h4>
          <ul class="list-disc list-inside space-y-1 text-slate-400 text-xs">
            <li><strong>Double 1s:</strong> Roll one fewer attack die next attack</li>
            <li><strong>Double 2-5:</strong> Can reroll one attack die</li>
            <li><strong>Double 6s:</strong> Can reroll 2 attack dice</li>
          </ul>
        </div>
      `;
    }
  }

  /**
   * Send WebSocket intent
   */
  sendIntent(intent) {
    if (this.gameState.ws && this.gameState.ws.readyState === WebSocket.OPEN) {
      const envelope = {
        seq: this.gameState.clientSeq++,
        intent: intent,
      };
      this.gameState.ws.send(JSON.stringify(envelope));
    } else {
      console.error('WebSocket not connected');
      this.logEvent('WebSocket not connected', 'error');
    }
  }
}

/**
 * Initialize GM controls if on GM page
 */
export function initializeGMControls(gameState) {
  // Check if we're on the GM page
  const isGMPage = document.getElementById('monster-spawn-modal') !== null;

  if (!isGMPage) {
    return null;
  }

  console.log('Initializing GM controls');
  const controller = new GMControlsController(gameState);

  // Update from initial snapshot
  if (gameState.snapshot) {
    controller.updateFromSnapshot(gameState.snapshot);
  }

  return controller;
}
