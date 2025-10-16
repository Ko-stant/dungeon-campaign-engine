/**
 *   Manages quest setup phase UI and interactions
 */
class QuestSetupController {
  constructor(gameState) {
    this.gameState = gameState;

    // DOM elements - Status Panel
    this.statusPanel = document.getElementById('quest-setup-status-panel');
    this.setupPlayersList = document.getElementById('setup-players-list');
    this.setupProgressBar = document.getElementById('setup-progress-bar');
    this.setupProgressCurrent = document.getElementById('setup-progress-current');
    this.setupProgressTotal = document.getElementById('setup-progress-total');
    this.openPositionBtn = document.getElementById('open-position-selection-btn');

    // DOM elements - Position Modal
    this.modal = document.getElementById('starting-position-modal');
    this.characterIcon = document.getElementById('starting-character-icon');
    this.characterName = document.getElementById('starting-character-name');
    this.characterMovement = document.getElementById('starting-character-movement');
    this.availablePositionsList = document.getElementById('available-positions-list');
    this.selectedPosDisplay = document.getElementById('selected-position-display');
    this.selectedPosX = document.getElementById('selected-pos-x');
    this.selectedPosY = document.getElementById('selected-pos-y');
    this.otherPlayersList = document.getElementById('other-players-list');
    this.playersReadyCount = document.getElementById('players-ready-count');
    this.totalPlayersCount = document.getElementById('total-players-count');
    this.changePositionBtn = document.getElementById('change-position-btn');
    this.confirmPositionBtn = document.getElementById('confirm-position-btn');

    // Quest info elements
    this.questName = document.getElementById('quest-name');
    this.questDescription = document.getElementById('quest-description');

    // State
    this.selectedPosition = null;
    this.availablePositions = [];
    this.myPlayerID = null;
    this.isModalOpen = false;
    this.animationFrameId = null;

    // Initialize event listeners
    this.initializeEventListeners();
  }

  initializeEventListeners() {
    if (this.openPositionBtn) {
      this.openPositionBtn.addEventListener('click', () => this.openModal());
    }

    if (this.changePositionBtn) {
      this.changePositionBtn.addEventListener('click', () => this.clearSelection());
    }

    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.addEventListener('click', () => this.confirmPosition());
    }

    // Close modal on click outside
    if (this.modal) {
      this.modal.addEventListener('click', (e) => {
        if (e.target === this.modal) {
          this.closeModal();
        }
      });
    }
  }

  // Called when snapshot is received
  updateFromSnapshot(snapshot) {
    this.myPlayerID = snapshot.viewerPlayerId;

    // Check if we're in quest setup phase AND viewer is a hero (not GM)
    // GM should never see quest setup UI elements
    if (snapshot.turnPhase === 'quest_setup' && snapshot.viewerRole !== 'gm') {
      this.show();
      this.updateQuestSetupState(snapshot);
    } else {
      this.hide();
    }
  }

  updateQuestSetupState(snapshot) {
    // Update quest info
    if (this.questName && snapshot.questName) {
      this.questName.textContent = snapshot.questName;
    }
    if (this.questDescription && snapshot.questDescription) {
      this.questDescription.textContent = snapshot.questDescription;
    }

    // Store available positions
    this.availablePositions = snapshot.startingPositions || [];

    // Update players ready status
    const playersReady = snapshot.playersReady || {};
    const playerStartPositions = snapshot.playerStartPositions || {};

    // Get list of hero player IDs (exclude GM)
    const heroEntities = (snapshot.entities || []).filter(e => e.kind === 'hero');
    const heroPlayerIDs = heroEntities
      .map(entity => {
        const heroState = snapshot.heroTurnStates ? snapshot.heroTurnStates[entity.id] : null;
        return heroState ? heroState.playerId : null;
      })
      .filter(pid => pid !== null);

    // Count ready players (only heroes, not GM)
    const readyCount = heroPlayerIDs.filter(pid => playersReady[pid]).length;
    const totalPlayers = heroPlayerIDs.length;

    // Update progress
    if (this.setupProgressCurrent) {
      this.setupProgressCurrent.textContent = readyCount;
    }
    if (this.setupProgressTotal) {
      this.setupProgressTotal.textContent = totalPlayers;
    }
    if (this.setupProgressBar) {
      const progressPercent = totalPlayers > 0 ? (readyCount / totalPlayers) * 100 : 0;
      this.setupProgressBar.style.width = `${progressPercent}%`;
    }
    if (this.playersReadyCount) {
      this.playersReadyCount.textContent = readyCount;
    }
    if (this.totalPlayersCount) {
      this.totalPlayersCount.textContent = totalPlayers;
    }

    // Update players list in status panel
    this.updatePlayersStatusList(snapshot, playersReady, playerStartPositions);

    // If modal is open, update other players list
    if (this.isModalOpen) {
      this.updateOtherPlayersList(snapshot, playersReady, playerStartPositions);
    }

    // Check if current player has selected a position
    const myPosition = playerStartPositions[this.myPlayerID];
    console.log('QUEST-SETUP: updateQuestSetupState called');
    console.log('QUEST-SETUP: myPlayerID =', this.myPlayerID);
    console.log('QUEST-SETUP: playerStartPositions =', playerStartPositions);
    console.log('QUEST-SETUP: myPosition =', myPosition);

    if (myPosition && this.isModalOpen) {
      this.selectPosition(myPosition.x, myPosition.y, false); // Don't send to server
    }

    // Update button states
    const hasSelectedPosition = !!myPosition;
    const isReady = playersReady[this.myPlayerID] || false;

    console.log('QUEST-SETUP: hasSelectedPosition =', hasSelectedPosition);
    console.log('QUEST-SETUP: isReady =', isReady);
    console.log('QUEST-SETUP: Button will be disabled =', !hasSelectedPosition);

    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.disabled = !hasSelectedPosition;
      if (isReady) {
        this.confirmPositionBtn.textContent = 'Ready!';
        this.confirmPositionBtn.className = 'px-4 py-2 bg-green-600 text-white rounded-lg cursor-default';
      } else {
        this.confirmPositionBtn.textContent = 'Confirm Position';
        this.confirmPositionBtn.className = 'px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
      }
    }

    if (this.changePositionBtn) {
      this.changePositionBtn.disabled = !hasSelectedPosition || isReady;
    }
  }

  updatePlayersStatusList(snapshot, playersReady, playerStartPositions) {
    if (!this.setupPlayersList) return;

    // Get all hero entities from snapshot
    const heroEntities = (snapshot.entities || []).filter(e => e.kind === 'hero');

    if (heroEntities.length === 0) {
      this.setupPlayersList.innerHTML = '<div class="text-xs text-slate-400 text-center py-2">No heroes yet</div>';
      return;
    }

    this.setupPlayersList.innerHTML = '';

    heroEntities.forEach(entity => {
      // Find player ID for this entity (from heroTurnStates)
      const heroState = snapshot.heroTurnStates ? snapshot.heroTurnStates[entity.id] : null;
      const playerID = heroState ? heroState.playerId : null;

      if (!playerID) return;

      const hasPosition = !!playerStartPositions[playerID];
      const isReady = playersReady[playerID] || false;
      const characterClass = entity.tags && entity.tags[0] ? entity.tags[0] : 'hero';
      const displayName = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);

      let statusIcon = '⏳';
      let statusText = 'Choosing position...';
      let statusColor = 'text-slate-400';

      if (isReady) {
        statusIcon = '✓';
        statusText = 'Ready';
        statusColor = 'text-green-400';
      } else if (hasPosition) {
        statusIcon = '📍';
        statusText = 'Position selected';
        statusColor = 'text-amber-400';
      }

      const playerDiv = document.createElement('div');
      playerDiv.className = 'flex items-center justify-between p-2 bg-slate-800/50 rounded border border-border/40';
      playerDiv.innerHTML = `
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-500 rounded flex items-center justify-center text-sm">
            ${characterClass.charAt(0).toUpperCase()}
          </div>
          <span class="text-sm font-semibold text-slate-200">${displayName}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs ${statusColor}">${statusIcon} ${statusText}</span>
        </div>
      `;

      this.setupPlayersList.appendChild(playerDiv);
    });
  }

  updateOtherPlayersList(snapshot, playersReady, playerStartPositions) {
    if (!this.otherPlayersList) return;

    const heroEntities = (snapshot.entities || []).filter(e => e.kind === 'hero');

    // Filter out current player
    const otherHeroes = heroEntities.filter(entity => {
      const heroState = snapshot.heroTurnStates ? snapshot.heroTurnStates[entity.id] : null;
      const playerID = heroState ? heroState.playerId : null;
      return playerID && playerID !== this.myPlayerID;
    });

    if (otherHeroes.length === 0) {
      this.otherPlayersList.innerHTML = '<div class="text-xs text-slate-400 text-center py-2">No other heroes</div>';
      return;
    }

    this.otherPlayersList.innerHTML = '';

    otherHeroes.forEach(entity => {
      const heroState = snapshot.heroTurnStates ? snapshot.heroTurnStates[entity.id] : null;
      const playerID = heroState ? heroState.playerId : null;

      if (!playerID) return;

      const position = playerStartPositions[playerID];
      const isReady = playersReady[playerID] || false;
      const characterClass = entity.tags && entity.tags[0] ? entity.tags[0] : 'hero';
      const displayName = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);

      let statusText = 'Choosing...';
      let statusColor = 'text-slate-400';

      if (isReady) {
        statusText = `Ready at (${position.x}, ${position.y})`;
        statusColor = 'text-green-400';
      } else if (position) {
        statusText = `Selected (${position.x}, ${position.y})`;
        statusColor = 'text-amber-400';
      }

      const playerDiv = document.createElement('div');
      playerDiv.className = 'flex items-center justify-between p-2 bg-slate-800/30 rounded text-xs';
      playerDiv.innerHTML = `
        <span class="text-slate-200">${displayName}</span>
        <span class="${statusColor}">${statusText}</span>
      `;

      this.otherPlayersList.appendChild(playerDiv);
    });
  }

  show() {
    if (this.statusPanel) {
      this.statusPanel.classList.remove('hidden');
    }
  }

  hide() {
    if (this.statusPanel) {
      this.statusPanel.classList.add('hidden');
    }
    this.closeModal();
  }

  openModal() {
    if (this.modal) {
      this.isModalOpen = true;
      this.modal.classList.remove('hidden');

      // Update character info
      const snapshot = this.gameState.getSnapshot();
      if (snapshot && snapshot.viewerEntityId) {
        const entity = (snapshot.entities || []).find(e => e.id === snapshot.viewerEntityId);
        if (entity) {
          const characterClass = entity.tags && entity.tags[0] ? entity.tags[0] : 'hero';
          const displayName = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);

          if (this.characterName) {
            this.characterName.textContent = displayName;
          }

          // Get movement from hero turn state
          const heroState = snapshot.heroTurnStates ? snapshot.heroTurnStates[entity.id] : null;
          if (heroState && this.characterMovement) {
            this.characterMovement.textContent = heroState.movementTotal || '0';
          }
        }
      }

      // Start animation loop for pulsing highlights
      this.startAnimationLoop();
    }
  }

  closeModal() {
    if (this.modal) {
      this.isModalOpen = false;
      this.modal.classList.add('hidden');

      // Stop animation loop
      this.stopAnimationLoop();

      // Trigger one final redraw to clear highlights
      this.gameState.requestRedraw();
    }
  }


  // Called when player clicks on a tile
  handleTileClick(x, y) {
    if (!this.isModalOpen) return false;

    // Check if this position is available
    const isAvailable = this.availablePositions.some(pos => pos.x === x && pos.y === y);
    if (!isAvailable) return false;

    // Select this position
    this.selectPosition(x, y, true); // Send to server
    return true;
  }

  selectPosition(x, y, sendToServer = true) {
    this.selectedPosition = { x, y };

    // Update UI
    if (this.selectedPosDisplay) {
      this.selectedPosDisplay.classList.remove('hidden');
    }
    if (this.selectedPosX) {
      this.selectedPosX.textContent = x;
    }
    if (this.selectedPosY) {
      this.selectedPosY.textContent = y;
    }
    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.disabled = false;
    }
    if (this.changePositionBtn) {
      this.changePositionBtn.disabled = false;
    }

    // Send to server if requested
    if (sendToServer) {
      this.gameState.sendIntent('RequestSelectStartingPosition', { x, y });
      console.log(`Selected starting position: (${x}, ${y})`);
    }

    // Trigger redraw to show selection
    this.gameState.requestRedraw();
  }

  clearSelection() {
    this.selectedPosition = null;

    if (this.selectedPosDisplay) {
      this.selectedPosDisplay.classList.add('hidden');
    }
    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.disabled = true;
    }
    if (this.changePositionBtn) {
      this.changePositionBtn.disabled = true;
    }

    // Trigger redraw to clear selection highlight
    this.gameState.requestRedraw();
  }

  confirmPosition() {
    if (!this.selectedPosition) return;

    // Send ready status to server
    this.gameState.sendIntent('RequestQuestSetupToggleReady', { isReady: true });
    console.log('Confirmed starting position, marked as ready');

    // Close modal
    this.closeModal();
  }

  startAnimationLoop() {
    if (this.animationFrameId) {
      return; // Already running
    }

    const animate = () => {
      if (!this.isModalOpen) {
        return; // Stop if modal closed
      }

      // Request redraw
      this.gameState.requestRedraw();

      // Schedule next frame
      this.animationFrameId = requestAnimationFrame(animate);
    };

    // Start the loop
    this.animationFrameId = requestAnimationFrame(animate);
  }

  stopAnimationLoop() {
    if (this.animationFrameId) {
      cancelAnimationFrame(this.animationFrameId);
      this.animationFrameId = null;
    }
  }
}

// Export for ES6 module usage
export { QuestSetupController };
