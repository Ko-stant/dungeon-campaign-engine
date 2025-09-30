package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// TreasureDeckManager manages the treasure deck
type TreasureDeckManager struct {
	deck           []*TreasureCard
	discardPile    []*TreasureCard
	contentManager *ContentManager
	logger         Logger
	mutex          sync.Mutex
	rng            *rand.Rand
}

// NewTreasureDeckManager creates a new treasure deck manager
func NewTreasureDeckManager(contentManager *ContentManager, logger Logger) *TreasureDeckManager {
	return &TreasureDeckManager{
		deck:           make([]*TreasureCard, 0),
		discardPile:    make([]*TreasureCard, 0),
		contentManager: contentManager,
		logger:         logger,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// InitializeDeck loads and shuffles the treasure deck for the campaign
func (tdm *TreasureDeckManager) InitializeDeck() error {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	campaign := tdm.contentManager.GetCampaign()
	if campaign == nil {
		return fmt.Errorf("campaign not loaded")
	}

	// Get all treasure cards from content manager
	treasures := tdm.contentManager.GetAllTreasures()

	// Build deck based on treasure_deck.json counts
	// For now, we'll create the deck from the available treasures
	// In a full implementation, we'd read the deck configuration to get counts
	for _, card := range treasures {
		// Add card based on its nature
		// Wandering monsters and hazards typically have multiple copies
		count := 1
		if card.Type == "monster" {
			count = 6 // Wandering monsters typically have 6 copies
		} else if card.Type == "hazard" {
			count = 3 // Hazards typically have 3 copies each
		}

		for i := 0; i < count; i++ {
			tdm.deck = append(tdm.deck, card)
		}
	}

	tdm.logger.Printf("Initialized treasure deck with %d cards", len(tdm.deck))

	// Shuffle the deck
	tdm.shuffleDeck()

	return nil
}

// DrawCard draws a card from the treasure deck
func (tdm *TreasureDeckManager) DrawCard() (*TreasureCard, error) {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	if len(tdm.deck) == 0 {
		// Reshuffle discard pile if deck is empty
		if len(tdm.discardPile) == 0 {
			return nil, fmt.Errorf("treasure deck and discard pile are both empty")
		}

		tdm.logger.Printf("Treasure deck empty, reshuffling discard pile (%d cards)", len(tdm.discardPile))
		tdm.deck = append(tdm.deck, tdm.discardPile...)
		tdm.discardPile = make([]*TreasureCard, 0)
		tdm.shuffleDeck()
	}

	// Draw top card
	card := tdm.deck[0]
	tdm.deck = tdm.deck[1:]

	tdm.logger.Printf("Drew treasure card: %s (%s)", card.Name, card.Type)

	// If card should return to deck, add it back immediately
	if card.ReturnToDeck {
		tdm.deck = append(tdm.deck, card)
		tdm.shuffleDeck()
		tdm.logger.Printf("Card %s returned to deck and reshuffled", card.ID)
	} else {
		// Otherwise, add to discard pile
		tdm.discardPile = append(tdm.discardPile, card)
	}

	return card, nil
}

// ReturnCardToDeck returns a specific card to the deck (for special game mechanics)
func (tdm *TreasureDeckManager) ReturnCardToDeck(cardID string) error {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	// Find card in discard pile
	for i, card := range tdm.discardPile {
		if card.ID == cardID {
			// Remove from discard pile
			tdm.discardPile = append(tdm.discardPile[:i], tdm.discardPile[i+1:]...)
			// Add back to deck
			tdm.deck = append(tdm.deck, card)
			tdm.shuffleDeck()
			tdm.logger.Printf("Returned card %s to deck", cardID)
			return nil
		}
	}

	return fmt.Errorf("card %s not found in discard pile", cardID)
}

// ShuffleDeck shuffles the deck
func (tdm *TreasureDeckManager) ShuffleDeck() error {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	tdm.shuffleDeck()
	return nil
}

// shuffleDeck shuffles the deck (internal, assumes lock is held)
func (tdm *TreasureDeckManager) shuffleDeck() {
	tdm.rng.Shuffle(len(tdm.deck), func(i, j int) {
		tdm.deck[i], tdm.deck[j] = tdm.deck[j], tdm.deck[i]
	})
	tdm.logger.Printf("Shuffled treasure deck (%d cards)", len(tdm.deck))
}

// GetDeckSize returns the current size of the deck
func (tdm *TreasureDeckManager) GetDeckSize() int {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()
	return len(tdm.deck)
}

// GetDiscardSize returns the current size of the discard pile
func (tdm *TreasureDeckManager) GetDiscardSize() int {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()
	return len(tdm.discardPile)
}
