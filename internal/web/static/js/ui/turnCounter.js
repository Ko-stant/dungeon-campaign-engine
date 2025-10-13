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

    // Update turn number
    if (this.turnNumberElement && typeof snapshot.turn === 'number') {
      this.turnNumberElement.textContent = String(snapshot.turn);
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
    console.log('TURN-COUNTER: heroTurnStates:', snapshot.heroTurnStates);
    console.log('TURN-COUNTER: entities:', snapshot.entities);

    // Check if there are any hero turn states
    if (snapshot.heroTurnStates && Object.keys(snapshot.heroTurnStates).length > 0) {
      console.log('TURN-COUNTER: Found hero turn states:', Object.keys(snapshot.heroTurnStates));

      // Find the first hero entity to get their info
      const firstHero = snapshot.entities?.find(e => e.kind === 'hero');
      console.log('TURN-COUNTER: First hero found:', firstHero);

      if (firstHero) {
        const turnState = snapshot.heroTurnStates[firstHero.id];
        console.log('TURN-COUNTER: Turn state for', firstHero.id, ':', turnState);
        console.log('TURN-COUNTER: actionTaken:', turnState?.actionTaken);

        // Hero's turn if they haven't taken their action yet (or if turnState exists and actionTaken is false)
        if (turnState && turnState.actionTaken === false) {
          const heroName = this.getHeroName(firstHero.id, snapshot);
          console.log('TURN-COUNTER: Setting active player to:', heroName);
          this.currentPlayerElement.textContent = heroName;
          this.currentPlayerElement.classList.add('text-blue-400');
          this.currentPlayerElement.classList.remove('text-orange-400');
          return;
        } else {
          console.log('TURN-COUNTER: Condition failed - turnState exists:', !!turnState, ', actionTaken:', turnState?.actionTaken);
        }
      } else {
        console.log('TURN-COUNTER: No hero entity found');
      }
    } else {
      console.log('TURN-COUNTER: No hero turn states found');
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
