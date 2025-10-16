/**
 * Hero Turn Controls Module
 * Handles hero-side turn management including election, active turn UI,
 * and starting position selection
 */

export class HeroTurnControlsController {
  constructor(gameState) {
    this.gameState = gameState;
    this.selectedStartingPosition = null;

    this.initializeElements();
    this.attachEventListeners();
  }

  initializeElements() {
    // Turn Election Panel Elements
    this.electionPanel = document.getElementById('hero-election-panel');
    this.electionStatusText = document.getElementById('election-status-text');
    this.electionCycleNumber = document.getElementById('election-cycle-number');
    this.electionHeroesRemaining = document.getElementById('election-heroes-remaining');
    this.electedPlayerInfo = document.getElementById('elected-player-info');
    this.electedPlayerName = document.getElementById('elected-player-name');
    this.noElectionInfo = document.getElementById('no-election-info');
    this.heroesActedSection = document.getElementById('heroes-acted-section');
    this.heroesActedList = document.getElementById('heroes-acted-list');
    this.electSelfBtn = document.getElementById('elect-self-btn');
    this.cancelElectionBtn = document.getElementById('cancel-election-btn');
    this.waitingForElection = document.getElementById('waiting-for-election');

    // Active Turn Panel Elements
    this.activePanel = document.getElementById('hero-active-panel');
    this.activeCharacterName = document.getElementById('active-character-name');
    this.activeMovementRemaining = document.getElementById('active-movement-remaining');
    this.maxMovement = document.getElementById('max-movement');
    this.completeHeroTurnBtn = document.getElementById('complete-hero-turn-btn');

    // Quest Setup Elements
    this.questSetupPanel = document.getElementById('quest-setup-status-panel');
    this.questName = document.getElementById('quest-name');
    this.questDescription = document.getElementById('quest-description');
    this.setupPlayersList = document.getElementById('setup-players-list');
    this.setupProgressCurrent = document.getElementById('setup-progress-current');
    this.setupProgressTotal = document.getElementById('setup-progress-total');
    this.setupProgressBar = document.getElementById('setup-progress-bar');
    this.openPositionSelectionBtn = document.getElementById('open-position-selection-btn');

    // Starting Position Modal Elements
    this.startingPositionModal = document.getElementById('starting-position-modal');
    this.startingCharacterIcon = document.getElementById('starting-character-icon');
    this.startingCharacterName = document.getElementById('starting-character-name');
    this.startingCharacterMovement = document.getElementById('starting-character-movement');
    this.availablePositionsList = document.getElementById('available-positions-list');
    this.selectedPositionDisplay = document.getElementById('selected-position-display');
    this.selectedPosX = document.getElementById('selected-pos-x');
    this.selectedPosY = document.getElementById('selected-pos-y');
    this.otherPlayersList = document.getElementById('other-players-list');
    this.playersReadyCount = document.getElementById('players-ready-count');
    this.totalPlayersCount = document.getElementById('total-players-count');
    this.changePositionBtn = document.getElementById('change-position-btn');
    this.confirmPositionBtn = document.getElementById('confirm-position-btn');

    // Header Turn Phase Indicator
    this.turnPhaseIndicator = document.getElementById('turn-phase-indicator');
    this.phaseStatusDot = document.getElementById('phase-status-dot');
    this.phaseStatusText = document.getElementById('phase-status-text');

    // Character Info in Header
    this.characterIcon = document.getElementById('character-icon');
    this.characterName = document.getElementById('character-name');
    this.characterClass = document.getElementById('character-class');
    this.characterBody = document.getElementById('character-body');
    this.characterMind = document.getElementById('character-mind');
    this.characterGold = document.getElementById('character-gold');

    // Turn Info in Header
    this.turnNumber = document.getElementById('turn-number');
    this.activePlayer = document.getElementById('active-player');
  }

  attachEventListeners() {
    // Turn Election
    if (this.electSelfBtn) {
      this.electSelfBtn.addEventListener('click', () => this.electSelf());
    }
    if (this.cancelElectionBtn) {
      this.cancelElectionBtn.addEventListener('click', () => this.cancelElection());
    }

    // Active Turn
    if (this.completeHeroTurnBtn) {
      this.completeHeroTurnBtn.addEventListener('click', () => this.completeHeroTurn());
    }

    // Quest Setup
    if (this.openPositionSelectionBtn) {
      this.openPositionSelectionBtn.addEventListener('click', () => this.openPositionSelectionModal());
    }

    // Starting Position Selection
    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.addEventListener('click', () => this.confirmStartingPosition());
    }
    if (this.changePositionBtn) {
      this.changePositionBtn.addEventListener('click', () => this.clearSelectedPosition());
    }

    // Modal backdrop click to close
    if (this.startingPositionModal) {
      this.startingPositionModal.addEventListener('click', (e) => {
        if (e.target === this.startingPositionModal) {
          // Don't allow closing during quest setup
          // Users must select a position
        }
      });
    }
  }

  /**
   * Turn Phase Management
   */
  updateTurnPhase(snapshot) {
    if (!snapshot) return;

    const phase = snapshot.turnPhase;
    const cycleNumber = snapshot.cycleNumber || 0;

    // Hide all panels first
    this.hideAllPanels();

    // Update header phase indicator
    this.updateHeaderPhaseIndicator(phase, snapshot);

    // Show appropriate panel based on phase
    switch (phase) {
      case 'quest_setup':
        this.showQuestSetupPanel(snapshot);
        break;
      case 'hero_election':
        this.showElectionPanel(snapshot);
        break;
      case 'hero_active':
        this.showActivePanel(snapshot);
        break;
      case 'gm_phase':
        this.showGMPhaseIndicator(cycleNumber);
        break;
      default:
        console.warn('Unknown turn phase:', phase);
    }
  }

  hideAllPanels() {
    [
      this.electionPanel,
      this.activePanel,
      this.questSetupPanel,
    ].forEach(panel => {
      if (panel) {
        panel.classList.add('hidden');
      }
    });
  }

  updateHeaderPhaseIndicator(phase, snapshot) {
    if (!this.turnPhaseIndicator) return;

    let dotClass = '';
    let text = '';
    let borderClass = '';

    switch (phase) {
      case 'quest_setup':
        dotClass = 'bg-purple-500 animate-pulse';
        text = 'Quest Setup';
        borderClass = 'border-purple-500 bg-purple-900/20';
        break;
      case 'hero_election':
        dotClass = 'bg-blue-500 animate-pulse';
        text = 'Waiting for Hero';
        borderClass = 'border-blue-500 bg-blue-900/20';
        break;
      case 'hero_active':
        const isMyTurn = snapshot.activeHeroPlayerID === snapshot.viewerPlayerID;
        if (isMyTurn) {
          dotClass = 'bg-green-500 animate-pulse';
          text = 'Your Turn';
          borderClass = 'border-green-500 bg-green-900/20';
        } else {
          dotClass = 'bg-yellow-500';
          text = 'Hero Turn';
          borderClass = 'border-yellow-500 bg-yellow-900/20';
        }
        break;
      case 'gm_phase':
        dotClass = 'bg-red-500 animate-pulse';
        text = 'GM Phase';
        borderClass = 'border-red-500 bg-red-900/20';
        break;
      default:
        this.turnPhaseIndicator.classList.add('hidden');
        return;
    }

    this.turnPhaseIndicator.classList.remove('hidden');
    this.turnPhaseIndicator.className = `flex items-center gap-2 px-3 py-1.5 rounded-lg border ${borderClass}`;

    if (this.phaseStatusDot) {
      this.phaseStatusDot.className = `w-2 h-2 rounded-full ${dotClass}`;
    }
    if (this.phaseStatusText) {
      this.phaseStatusText.textContent = text;
    }
  }

  /**
   * Quest Setup Phase
   */
  showQuestSetupPanel(snapshot) {
    if (!this.questSetupPanel) return;

    this.questSetupPanel.classList.remove('hidden');

    // Update quest info (would come from snapshot)
    // For now, using placeholder data
    if (this.questName) {
      this.questName.textContent = snapshot.questName || 'The Trial of Champions';
    }
    if (this.questDescription) {
      this.questDescription.textContent = snapshot.questDescription || 'A dangerous quest awaits the heroes...';
    }

    // Update players list and progress
    this.updateSetupProgress(snapshot);
  }

  updateSetupProgress(snapshot) {
    // This would be updated based on who has selected positions
    // For now, placeholder implementation
    const totalPlayers = snapshot.totalPlayers || 4;
    const readyPlayers = snapshot.playersReady?.length || 0;

    if (this.setupProgressCurrent) {
      this.setupProgressCurrent.textContent = readyPlayers;
    }
    if (this.setupProgressTotal) {
      this.setupProgressTotal.textContent = totalPlayers;
    }
    if (this.setupProgressBar) {
      const percentage = (readyPlayers / totalPlayers) * 100;
      this.setupProgressBar.style.width = `${percentage}%`;
    }
  }

  openPositionSelectionModal() {
    if (!this.startingPositionModal) return;

    this.startingPositionModal.classList.remove('hidden');
    this.startingPositionModal.classList.add('flex');

    // Update character info
    this.updateCharacterInfoInModal();

    // Request available positions from server
    this.requestAvailableStartingPositions();
  }

  updateCharacterInfoInModal() {
    // Get player's character info from game state
    const player = this.gameState.snapshot?.players?.find(p => p.id === this.gameState.playerID);

    if (player && this.startingCharacterName) {
      this.startingCharacterName.textContent = player.name || 'Hero';
    }
    if (player && this.startingCharacterMovement) {
      this.startingCharacterMovement.textContent = player.movement || '0';
    }
  }

  selectStartingPosition(x, y) {
    this.selectedStartingPosition = { x, y };

    // Update UI
    if (this.selectedPositionDisplay) {
      this.selectedPositionDisplay.classList.remove('hidden');
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
  }

  clearSelectedPosition() {
    this.selectedStartingPosition = null;

    if (this.selectedPositionDisplay) {
      this.selectedPositionDisplay.classList.add('hidden');
    }
    if (this.confirmPositionBtn) {
      this.confirmPositionBtn.disabled = true;
    }
    if (this.changePositionBtn) {
      this.changePositionBtn.disabled = true;
    }
  }

  confirmStartingPosition() {
    if (!this.selectedStartingPosition) {
      console.error('No position selected');
      return;
    }

    this.sendIntent({
      kind: 'SelectStartingPosition',
      position: this.selectedStartingPosition,
    });

    // Close modal (server will confirm via patch)
    if (this.startingPositionModal) {
      this.startingPositionModal.classList.add('hidden');
      this.startingPositionModal.classList.remove('flex');
    }
  }

  requestAvailableStartingPositions() {
    this.sendIntent({
      kind: 'RequestStartingPositions',
    });
  }

  /**
   * Hero Election Phase
   */
  showElectionPanel(snapshot) {
    if (!this.electionPanel) return;

    // Count total hero players (exclude GM)
    const totalHeroPlayers = snapshot.players?.filter(p => p.role !== 'gm').length || 0;

    // If only one hero player, skip election and hide panel
    if (totalHeroPlayers <= 1) {
      this.electionPanel.classList.add('hidden');
      return;
    }

    this.electionPanel.classList.remove('hidden');

    const cycleNumber = snapshot.cycleNumber || 1;
    const electedPlayerID = snapshot.electedPlayerID;
    const heroesActed = snapshot.heroesActedIDs || [];
    const totalHeroes = totalHeroPlayers;
    const heroesRemaining = totalHeroes - heroesActed.length;

    // Update cycle info
    if (this.electionCycleNumber) {
      this.electionCycleNumber.textContent = cycleNumber;
    }
    if (this.electionHeroesRemaining) {
      this.electionHeroesRemaining.textContent = heroesRemaining;
    }

    // Update elected player info
    if (electedPlayerID) {
      this.showElectedPlayer(electedPlayerID, snapshot);
    } else {
      this.showNoElection();
    }

    // Update heroes acted list
    if (heroesActed.length > 0) {
      this.showHeroesActed(heroesActed, snapshot);
    } else {
      if (this.heroesActedSection) {
        this.heroesActedSection.classList.add('hidden');
      }
    }

    // Update button states
    const viewerPlayerID = snapshot.viewerPlayerID || this.gameState.playerID;
    const viewerEntityID = snapshot.viewerEntityID;
    const isMyElection = electedPlayerID === viewerPlayerID;
    const hasActed = heroesActed.includes(viewerPlayerID);

    // Check if viewer has rolled for movement or taken action
    const viewerTurnState = snapshot.heroTurnStates?.[viewerEntityID];
    const hasRolledOrActed = viewerTurnState && (viewerTurnState.movementDiceRolled || viewerTurnState.actionTaken);

    // Only show "I'll Go Next!" if haven't acted and not currently elected
    if (this.electSelfBtn) {
      this.electSelfBtn.disabled = hasActed || isMyElection;
      this.electSelfBtn.classList.toggle('hidden', isMyElection || hasActed);
    }

    // Only show cancel button if elected AND haven't rolled/acted yet
    if (this.cancelElectionBtn) {
      this.cancelElectionBtn.classList.toggle('hidden', !isMyElection);
      this.cancelElectionBtn.disabled = hasRolledOrActed;

      // Update button text to indicate why cancel is disabled
      if (hasRolledOrActed) {
        this.cancelElectionBtn.textContent = 'Cannot Cancel (Actions Taken)';
        this.cancelElectionBtn.classList.add('opacity-50', 'cursor-not-allowed');
      } else {
        this.cancelElectionBtn.textContent = 'Cancel My Election';
        this.cancelElectionBtn.classList.remove('opacity-50', 'cursor-not-allowed');
      }
    }

    if (this.waitingForElection) {
      this.waitingForElection.classList.toggle('hidden', !electedPlayerID || isMyElection);
    }
  }

  showElectedPlayer(playerID, snapshot) {
    const player = snapshot.players?.find(p => p.id === playerID);
    const playerName = player?.name || 'Unknown Hero';

    if (this.electedPlayerInfo) {
      this.electedPlayerInfo.classList.remove('hidden');
    }
    if (this.electedPlayerName) {
      this.electedPlayerName.textContent = playerName;
    }
    if (this.noElectionInfo) {
      this.noElectionInfo.classList.add('hidden');
    }
  }

  showNoElection() {
    if (this.electedPlayerInfo) {
      this.electedPlayerInfo.classList.add('hidden');
    }
    if (this.noElectionInfo) {
      this.noElectionInfo.classList.remove('hidden');
    }
  }

  showHeroesActed(heroesActedIDs, snapshot) {
    if (!this.heroesActedList) return;

    if (this.heroesActedSection) {
      this.heroesActedSection.classList.remove('hidden');
    }

    const heroesHTML = heroesActedIDs.map(playerID => {
      const player = snapshot.players?.find(p => p.id === playerID);
      const playerName = player?.name || 'Unknown Hero';

      return `
        <div class="flex items-center gap-2 text-xs text-slate-400">
          <svg class="w-3 h-3 text-green-500" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
          </svg>
          <span>${playerName}</span>
        </div>
      `;
    }).join('');

    this.heroesActedList.innerHTML = heroesHTML;
  }

  electSelf() {
    this.sendIntent({
      kind: 'ElectSelfAsNextPlayer',
    });
  }

  cancelElection() {
    this.sendIntent({
      kind: 'CancelElection',
    });
  }

  /**
   * Hero Active Phase
   */
  showActivePanel(snapshot) {
    const viewerPlayerID = snapshot.viewerPlayerID || this.gameState.playerID;
    const isMyTurn = snapshot.activeHeroPlayerID === viewerPlayerID;

    if (!isMyTurn) {
      // Not my turn, just show header indicator
      return;
    }

    if (!this.activePanel) return;

    this.activePanel.classList.remove('hidden');

    // Get character info from snapshot
    const viewerEntityID = snapshot.viewerEntityID;
    const heroEntity = snapshot.entities?.find(e => e.id === viewerEntityID);
    const heroTurnState = snapshot.heroTurnStates?.[viewerEntityID];

    if (heroEntity && this.activeCharacterName) {
      const characterClass = heroEntity.tags?.[0] || 'hero';
      const classDisplay = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);
      this.activeCharacterName.textContent = classDisplay;
    }

    // Update movement from turn state
    if (heroTurnState) {
      if (this.activeMovementRemaining) {
        this.activeMovementRemaining.textContent = `${heroTurnState.movementRemaining}/${heroTurnState.movementTotal}`;
      }
      if (this.maxMovement) {
        this.maxMovement.textContent = heroTurnState.movementTotal || '0';
      }
    }
  }

  completeHeroTurn() {
    this.sendIntent({
      kind: 'CompleteHeroTurn',
    });
  }

  showGMPhaseIndicator(cycleNumber) {
    // Just update the header indicator, no panel needed
    // Already handled in updateHeaderPhaseIndicator
  }

  /**
   * Update Character Info in Header
   */
  updateCharacterInfo(snapshot) {
    if (!snapshot) return;

    // Update turn info in header
    if (this.turnNumber && typeof snapshot.turn === 'number') {
      this.turnNumber.textContent = String(snapshot.turn);
    }

    // Get viewer info from snapshot
    const viewerPlayerID = snapshot.viewerPlayerId;
    const viewerRole = snapshot.viewerRole;
    const viewerEntityID = snapshot.viewerEntityId;

    // If GM, show GM label
    if (viewerRole === 'gm') {
      if (this.characterIcon) {
        this.characterIcon.textContent = '🎲';
      }
      if (this.characterName) {
        this.characterName.textContent = 'Game Master';
        this.characterName.className = 'text-sm font-semibold text-purple-400';
      }
      if (this.characterClass) {
        this.characterClass.textContent = '';
      }

      // Update active player for GM
      this.updateActivePlayer(snapshot);
      return;
    }

    // For heroes, get character data
    if (!viewerEntityID) return;

    // Find this hero's entity
    const heroEntity = snapshot.entities?.find(e => e.id === viewerEntityID);

    // Find hero turn state for this player
    const heroTurnState = snapshot.heroTurnStates?.[viewerEntityID];

    if (heroEntity) {
      // Get character class from tags
      const characterClass = heroEntity.tags?.[0] || 'hero';
      const classDisplay = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);

      if (this.characterIcon) {
        // Set icon based on class
        const classIcons = {
          'barbarian': '⚔️',
          'dwarf': '🪓',
          'elf': '🏹',
          'wizard': '🔮',
          'berserker': '🗡️',
          'knight': '🛡️',
          'explorer': '🧭',
          'druid': '🌿',
          'monk': '👊'
        };
        this.characterIcon.textContent = classIcons[characterClass] || '⚔️';
      }

      if (this.characterName) {
        this.characterName.textContent = classDisplay;
        this.characterName.className = 'text-sm font-semibold text-amber-400';
      }

      if (this.characterClass) {
        this.characterClass.textContent = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);
      }

      if (this.characterBody && heroEntity.hp) {
        this.characterBody.textContent = `${heroEntity.hp.current || 0}/${heroEntity.hp.max || 0}`;
      }

      if (this.characterMind && heroEntity.mindPoints) {
        this.characterMind.textContent = `${heroEntity.mindPoints.current || 0}/${heroEntity.mindPoints.max || 0}`;
      } else if (this.characterMind) {
        this.characterMind.textContent = '0/0';
      }
    }

    // Gold would come from inventory data (not yet in snapshot)
    if (this.characterGold) {
      this.characterGold.textContent = '0'; // TODO: Add gold to snapshot
    }

    // Update active player
    this.updateActivePlayer(snapshot);
  }

  /**
   * Update active player display
   */
  updateActivePlayer(snapshot) {
    if (!this.activePlayer) return;

    const phase = snapshot.turnPhase;

    // Determine active player based on phase
    if (phase === 'quest_setup') {
      this.activePlayer.textContent = 'Quest Setup';
    } else if (phase === 'hero_election') {
      if (snapshot.electedPlayerID) {
        // Find the hero for the elected player
        const heroTurnState = Object.values(snapshot.heroTurnStates || {}).find(
          state => state.playerId === snapshot.electedPlayerID
        );

        if (heroTurnState) {
          const heroEntity = snapshot.entities?.find(e => e.id === heroTurnState.heroId);
          if (heroEntity) {
            const characterClass = heroEntity.tags?.[0] || 'hero';
            const className = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);
            this.activePlayer.textContent = `${className} (Elected)`;
          } else {
            this.activePlayer.textContent = 'Hero (Elected)';
          }
        } else {
          this.activePlayer.textContent = 'Hero (Elected)';
        }
      } else {
        this.activePlayer.textContent = 'Electing Hero';
      }
    } else if (phase === 'hero_active') {
      const activeHeroPlayerID = snapshot.activeHeroPlayerID;
      if (activeHeroPlayerID) {
        // Find the hero entity for the active player
        const heroTurnState = Object.values(snapshot.heroTurnStates || {}).find(
          state => state.playerId === activeHeroPlayerID
        );

        if (heroTurnState) {
          // Get hero entity to find class
          const heroEntity = snapshot.entities?.find(e => e.id === heroTurnState.heroId);
          if (heroEntity) {
            const characterClass = heroEntity.tags?.[0] || 'hero';
            const className = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);
            this.activePlayer.textContent = className;
          } else {
            this.activePlayer.textContent = 'Hero';
          }
        } else {
          this.activePlayer.textContent = 'Hero Turn';
        }
      } else {
        this.activePlayer.textContent = 'Hero Turn';
      }
    } else if (phase === 'gm_phase') {
      this.activePlayer.textContent = 'Game Master';
    } else {
      // Fallback: try to determine active player from hero turn states
      console.log('Unknown phase:', phase, '- attempting fallback logic');

      // Check if there's an active hero in hero turn states
      if (snapshot.heroTurnStates && Object.keys(snapshot.heroTurnStates).length > 0) {
        // Find the first hero turn state (for single hero) or check actionTaken
        const activeHeroState = Object.values(snapshot.heroTurnStates).find(
          state => !state.actionTaken
        ) || Object.values(snapshot.heroTurnStates)[0];

        if (activeHeroState) {
          const heroEntity = snapshot.entities?.find(e => e.id === activeHeroState.heroId);
          if (heroEntity) {
            const characterClass = heroEntity.tags?.[0] || 'hero';
            const className = characterClass.charAt(0).toUpperCase() + characterClass.slice(1);
            this.activePlayer.textContent = className;
            return;
          }
        }
      }

      // Final fallback
      this.activePlayer.textContent = 'Unknown';
    }
  }

  /**
   * Update from snapshot
   */
  updateFromSnapshot(snapshot) {
    if (!snapshot) return;

    this.updateTurnPhase(snapshot);
    this.updateCharacterInfo(snapshot);
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
    }
  }
}

/**
 * Initialize hero turn controls
 */
export function initializeHeroTurnControls(gameState) {
  console.log('Initializing hero turn controls');
  const controller = new HeroTurnControlsController(gameState);

  // Update from initial snapshot
  if (gameState.snapshot) {
    controller.updateFromSnapshot(gameState.snapshot);
  }

  return controller;
}
