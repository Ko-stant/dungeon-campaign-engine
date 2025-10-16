/**
 * Turn Counter UI Controller
 * Updates turn number, current player, and connection status
 */

export class TurnCounterController {
  constructor(gameState) {
    this.gameState = gameState;
    this.turnNumberElement = document.getElementById('turnCounter');
    this.currentPlayerElement = document.getElementById('currentPlayer');
    this.patchCountElement = document.getElementById('patchCount');
  }

  /**
   * Update turn counter from snapshot
   * @param {Object} snapshot
   */
  updateFromSnapshot(snapshot) {
    if (!snapshot) return;

    // Update turn number (use cycleNumber from DynamicTurnOrderManager)
    if (this.turnNumberElement && typeof snapshot.cycleNumber === 'number') {
      this.turnNumberElement.textContent = String(snapshot.cycleNumber);
    }

    // Update current player
    this.updateCurrentPlayer(snapshot);
  }

  /**
   * Determine and display current player/phase
   * @param {Object} snapshot
   */
  updateCurrentPlayer(snapshot) {
    if (!this.currentPlayerElement) return;

    console.log('TURN-COUNTER: updateCurrentPlayer called');
    console.log('TURN-COUNTER: turnPhase:', snapshot.turnPhase);
    console.log('TURN-COUNTER: activeHeroPlayerID:', snapshot.activeHeroPlayerID);
    console.log('TURN-COUNTER: electedPlayerId:', snapshot.electedPlayerId);

    // Check turn phase and active player ID from snapshot
    if (snapshot.turnPhase === 'hero_turn' && snapshot.activeHeroPlayerID) {
      // Find the entity for the active hero player
      const heroEntity = snapshot.entities?.find(e => {
        const turnState = snapshot.heroTurnStates?.[e.id];
        return turnState && turnState.playerId === snapshot.activeHeroPlayerID;
      });

      if (heroEntity) {
        const heroName = this.getHeroName(heroEntity.id, snapshot);
        console.log('TURN-COUNTER: Setting active player to:', heroName);
        this.currentPlayerElement.textContent = heroName;
        this.currentPlayerElement.classList.add('text-blue-400');
        this.currentPlayerElement.classList.remove('text-orange-400');
        return;
      }
    } else if (snapshot.turnPhase === 'hero_election' && snapshot.electedPlayerId) {
      // During election, show the elected player
      const heroEntity = snapshot.entities?.find(e => {
        const turnState = snapshot.heroTurnStates?.[e.id];
        return turnState && turnState.playerId === snapshot.electedPlayerId;
      });

      if (heroEntity) {
        const heroName = this.getHeroName(heroEntity.id, snapshot);
        console.log('TURN-COUNTER: Setting elected player to:', heroName);
        this.currentPlayerElement.textContent = heroName + ' (electing)';
        this.currentPlayerElement.classList.add('text-blue-400');
        this.currentPlayerElement.classList.remove('text-orange-400');
        return;
      }
    }

    // Default to Game Master
    console.log('TURN-COUNTER: Defaulting to Game Master');
    this.currentPlayerElement.textContent = 'Game Master';
    this.currentPlayerElement.classList.add('text-orange-400');
    this.currentPlayerElement.classList.remove('text-blue-400');
  }

  /**
   * Get hero name from entity ID
   * @param {string} heroId
   * @param {Object} snapshot
   * @returns {string}
   */
  getHeroName(heroId, snapshot) {
    const hero = snapshot.entities?.find(e => e.id === heroId && e.kind === 'hero');
    if (hero && hero.tags) {
      // Tags might contain class name like 'barbarian', 'wizard', etc.
      const className = hero.tags.find(tag =>
        ['barbarian', 'wizard', 'dwarf', 'elf', 'berserker', 'knight', 'explorer', 'druid', 'monk'].includes(tag)
      );
      if (className) {
        return className.charAt(0).toUpperCase() + className.slice(1);
      }
    }
    return 'Hero';
  }

  /**
   * Update patch count
   * @param {number} count
   */
  updatePatchCount(count) {
    if (this.patchCountElement) {
      this.patchCountElement.textContent = String(count);
    }
  }
}
