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

    // Check if there are any hero turn states
    if (snapshot.heroTurnStates && Object.keys(snapshot.heroTurnStates).length > 0) {
      // Find the first hero entity to get their info
      const firstHero = snapshot.entities?.find(e => e.kind === 'hero');
      if (firstHero) {
        const turnState = snapshot.heroTurnStates[firstHero.id];
        // Hero's turn if they haven't taken their action yet (or if turnState exists and actionTaken is false)
        if (turnState && turnState.actionTaken === false) {
          this.currentPlayerElement.textContent = this.getHeroName(firstHero.id, snapshot);
          this.currentPlayerElement.classList.add('text-blue-400');
          this.currentPlayerElement.classList.remove('text-orange-400');
          return;
        }
      }
    }

    // Default to Game Master
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
