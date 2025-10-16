/**
 * Hero Turn State Management
 * Manages per-hero turn state synchronized from server
 */

// Hero turn state (synchronized from server)
export const heroTurnState = {
    heroID: null,
    playerID: null,
    turnNumber: 0,

    // Movement
    movementDice: {
        rolled: false,
        diceResults: [],      // e.g., [2, 3, 4]
        total: 0,             // 9
        used: 0,              // 4
        remaining: 0,         // 5
    },

    // Movement/Action tracking (simple flags)
    hasMoved: false,
    actionTaken: false,
    turnFlags: {},           // { "can_split_movement": true }

    // Active effects
    activeEffects: [],       // [{source, effectType, value, trigger, applied}]

    // Location tracking
    locationSearches: {},    // { "room-17": { treasureSearchDone: false } }

    // Positions
    turnStartPosition: null,
    currentPosition: null,

    // Client-side planning state (not synced to server)
    isPlanning: false,
    plannedPath: [],
};

// Validation helpers (mirrors server logic)
export function canMove() {
    if (!heroTurnState.movementDice.rolled) {
        console.log('HERO-TURN-STATE: Cannot move - dice not rolled');
        return false;
    }
    if (heroTurnState.movementDice.remaining <= 0) {
        console.log('HERO-TURN-STATE: Cannot move - no movement remaining');
        return false;
    }

    // Act-first: can still move after action if haven't moved
    if (heroTurnState.actionTaken && !heroTurnState.hasMoved) {
        console.log('HERO-TURN-STATE: Can move (act-first strategy)');
        return true;
    }

    // Move-first: can continue moving before action
    if (heroTurnState.hasMoved && !heroTurnState.actionTaken) {
        console.log('HERO-TURN-STATE: Can move (continuing move-first)');
        return true;
    }

    // Both done: only if split movement allowed
    if (heroTurnState.hasMoved && heroTurnState.actionTaken) {
        const canSplit = heroTurnState.turnFlags["can_split_movement"] || false;
        console.log(`HERO-TURN-STATE: ${canSplit ? 'Can' : 'Cannot'} move (split movement: ${canSplit})`);
        return canSplit;
    }

    // Neither done: can move
    console.log('HERO-TURN-STATE: Can move (initial state)');
    return true;
}

export function canTakeAction() {
    if (!heroTurnState.movementDice.rolled) {
        console.log('HERO-TURN-STATE: Cannot act - dice not rolled');
        return false;
    }

    if (heroTurnState.actionTaken) {
        const canExtra = heroTurnState.turnFlags["can_make_extra_attack"] || false;
        console.log(`HERO-TURN-STATE: ${canExtra ? 'Can' : 'Cannot'} act (extra attack: ${canExtra})`);
        return canExtra;
    }

    console.log('HERO-TURN-STATE: Can take action');
    return true;
}

export function getTurnStrategy() {
    if (!heroTurnState.hasMoved && !heroTurnState.actionTaken) {
        return "choose"; // Can choose either
    }
    if (heroTurnState.hasMoved && !heroTurnState.actionTaken) {
        return "move_first"; // Locked into moving first
    }
    if (!heroTurnState.hasMoved && heroTurnState.actionTaken) {
        return "act_first"; // Locked into acting first
    }
    return "complete"; // Both done
}

// Sync from server (called when snapshot/patch arrives)
export function syncHeroTurnStateFromServer(serverState) {
    console.log('HERO-TURN-STATE: Syncing from server:', serverState);

    if (!serverState) {
        console.log('HERO-TURN-STATE: No server state provided');
        return;
    }

    heroTurnState.heroID = serverState.heroId || serverState.heroID;
    heroTurnState.playerID = serverState.playerId || serverState.playerID;
    heroTurnState.turnNumber = serverState.turnNumber || 0;

    heroTurnState.movementDice = {
        rolled: serverState.movementDiceRolled || false,
        diceResults: serverState.movementDiceResults || [],
        total: serverState.movementTotal || 0,
        used: serverState.movementUsed || 0,
        remaining: serverState.movementRemaining || 0,
    };

    heroTurnState.hasMoved = serverState.hasMoved || false;
    heroTurnState.actionTaken = serverState.actionTaken || false;
    heroTurnState.turnFlags = serverState.turnFlags || {};
    heroTurnState.activeEffects = serverState.activeEffects || [];
    heroTurnState.locationSearches = serverState.locationSearches || {};

    heroTurnState.turnStartPosition = serverState.turnStartPosition;
    heroTurnState.currentPosition = serverState.currentPosition;

    console.log('HERO-TURN-STATE: Sync complete');
    console.log('HERO-TURN-STATE: - Movement rolled:', heroTurnState.movementDice.rolled);
    console.log('HERO-TURN-STATE: - Movement remaining:', heroTurnState.movementDice.remaining);
    console.log('HERO-TURN-STATE: - Has moved:', heroTurnState.hasMoved);
    console.log('HERO-TURN-STATE: - Action taken:', heroTurnState.actionTaken);
    console.log('HERO-TURN-STATE: - Turn strategy:', getTurnStrategy());

    // Restore movement planning if movement remaining
    if (canMove() && heroTurnState.movementDice.remaining > 0) {
        console.log('HERO-TURN-STATE: Movement available, restoring movement planning');

        // Import both movementPlanning and actionSystem modules
        Promise.all([
            import('./movementPlanning.js'),
            import('./actionSystem.js')
        ]).then(([mp, actionSystem]) => {
            mp.setMovementDiceRoll(heroTurnState.movementDice.total);

            // Update the movement tracking to match server state
            const trackingState = mp.turnMovementState;
            trackingState.diceRolled = true;
            trackingState.maxMovementForTurn = heroTurnState.movementDice.total;
            trackingState.movementUsedThisTurn = heroTurnState.movementDice.used;

            // Start planning if not already planning
            if (heroTurnState.movementDice.remaining > 0) {
                mp.startMovementPlanning();

                // Show the movement planning UI panel
                actionSystem.showMovementPlanningControls();
            }
        });
    } else {
        console.log('HERO-TURN-STATE: No movement available for planning');
    }
}

// Initialize from snapshot (called on page load)
export function initializeFromSnapshot(snapshot) {
    console.log('HERO-TURN-STATE: Initializing from snapshot');

    if (!snapshot || !snapshot.heroTurnStates) {
        console.log('HERO-TURN-STATE: No hero turn states in snapshot');
        return;
    }

    // For now, assume we're "hero-1" (in future, this will be dynamic based on player)
    const heroID = 'hero-1';
    const state = snapshot.heroTurnStates[heroID];

    if (state) {
        console.log('HERO-TURN-STATE: Found state for', heroID);
        syncHeroTurnStateFromServer(state);
    } else {
        console.log('HERO-TURN-STATE: No state found for', heroID);
    }
}