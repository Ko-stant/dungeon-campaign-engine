package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ContentManager manages all game content (equipment, treasures, spells, etc.)
type ContentManager struct {
	campaign        *CampaignMetadata
	equipmentCards  map[string]*ItemCard
	artifactCards   map[string]*ItemCard
	treasureCards   map[string]*TreasureCard
	spellCards      map[string]*SpellCard
	dreadSpellCards map[string]*SpellCard
	heroCards       map[string]*HeroCard
	logger          Logger
	mutex           sync.RWMutex
}

// NewContentManager creates a new content manager
func NewContentManager(logger Logger) *ContentManager {
	return &ContentManager{
		equipmentCards:  make(map[string]*ItemCard),
		artifactCards:   make(map[string]*ItemCard),
		treasureCards:   make(map[string]*TreasureCard),
		spellCards:      make(map[string]*SpellCard),
		dreadSpellCards: make(map[string]*SpellCard),
		heroCards:       make(map[string]*HeroCard),
		logger:          logger,
	}
}

// LoadCampaign loads all content for a specific campaign
func (cm *ContentManager) LoadCampaign(campaignID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Try to detect the correct base path
	// If running from root: content/base/campaign.json
	// If running from cmd/server (tests): ../../content/base/campaign.json
	var campaignPath string
	testPath := filepath.Join("content", campaignID, "campaign.json")
	if _, err := os.Stat(testPath); err == nil {
		// Running from root
		campaignPath = filepath.Join("content", campaignID)
	} else {
		// Running from cmd/server (tests)
		campaignPath = filepath.Join("..", "..", "content", campaignID)
	}

	campaignFile := filepath.Join(campaignPath, "campaign.json")
	cm.logger.Printf("Loading campaign from: %s", campaignFile)

	// Load campaign metadata
	campaign, err := cm.loadCampaignMetadata(campaignFile)
	if err != nil {
		return fmt.Errorf("failed to load campaign metadata: %w", err)
	}
	cm.campaign = campaign

	// Load equipment deck
	if err := cm.loadEquipmentDeck(filepath.Join(campaignPath, campaign.Decks.Equipment), campaignPath); err != nil {
		return fmt.Errorf("failed to load equipment deck: %w", err)
	}

	// Load artifact deck
	if err := cm.loadArtifactDeck(filepath.Join(campaignPath, campaign.Decks.Artifacts), campaignPath); err != nil {
		return fmt.Errorf("failed to load artifact deck: %w", err)
	}

	// Load treasure deck
	if err := cm.loadTreasureDeck(filepath.Join(campaignPath, campaign.Decks.Treasures), campaignPath); err != nil {
		return fmt.Errorf("failed to load treasure deck: %w", err)
	}

	// Load spell deck
	if err := cm.loadSpellDeck(filepath.Join(campaignPath, campaign.Decks.Spells), campaignPath); err != nil {
		return fmt.Errorf("failed to load spell deck: %w", err)
	}

	// Load dread spell deck
	if err := cm.loadDreadSpellDeck(filepath.Join(campaignPath, campaign.Decks.DreadSpells), campaignPath); err != nil {
		return fmt.Errorf("failed to load dread spell deck: %w", err)
	}

	// Load heroes
	if campaign.ContentPaths.Heroes != "" {
		heroesPath := filepath.Join(campaignPath, campaign.ContentPaths.Heroes)
		if err := cm.loadHeroes(heroesPath); err != nil {
			return fmt.Errorf("failed to load heroes: %w", err)
		}
	}

	cm.logger.Printf("Campaign '%s' loaded successfully", campaign.Name)
	cm.logger.Printf("  Equipment: %d items", len(cm.equipmentCards))
	cm.logger.Printf("  Artifacts: %d items", len(cm.artifactCards))
	cm.logger.Printf("  Treasures: %d cards", len(cm.treasureCards))
	cm.logger.Printf("  Spells: %d cards", len(cm.spellCards))
	cm.logger.Printf("  Dread Spells: %d cards", len(cm.dreadSpellCards))
	cm.logger.Printf("  Heroes: %d characters", len(cm.heroCards))

	return nil
}

// loadCampaignMetadata loads the campaign.json file
func (cm *ContentManager) loadCampaignMetadata(path string) (*CampaignMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read campaign file: %w", err)
	}

	var campaign CampaignMetadata
	if err := json.Unmarshal(data, &campaign); err != nil {
		return nil, fmt.Errorf("failed to parse campaign file: %w", err)
	}

	return &campaign, nil
}

// loadEquipmentDeck loads equipment_deck.json and all referenced items
func (cm *ContentManager) loadEquipmentDeck(deckPath string, basePath string) error {
	data, err := os.ReadFile(deckPath)
	if err != nil {
		return fmt.Errorf("failed to read equipment deck: %w", err)
	}

	var deck EquipmentDeck
	if err := json.Unmarshal(data, &deck); err != nil {
		return fmt.Errorf("failed to parse equipment deck: %w", err)
	}

	// Load each equipment card
	for _, ref := range deck.Items {
		cardPath := filepath.Join(basePath, ref.Path)
		card, err := cm.loadItemCard(cardPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load equipment card %s: %v", ref.ID, err)
			continue
		}
		cm.equipmentCards[card.ID] = card
	}

	return nil
}

// loadArtifactDeck loads artifact_deck.json and all referenced items
func (cm *ContentManager) loadArtifactDeck(deckPath string, basePath string) error {
	data, err := os.ReadFile(deckPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact deck: %w", err)
	}

	var deck ArtifactDeck
	if err := json.Unmarshal(data, &deck); err != nil {
		return fmt.Errorf("failed to parse artifact deck: %w", err)
	}

	// Load each artifact card
	for _, ref := range deck.Items {
		cardPath := filepath.Join(basePath, ref.Path)
		card, err := cm.loadItemCard(cardPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load artifact card %s: %v", ref.ID, err)
			continue
		}
		cm.artifactCards[card.ID] = card
	}

	return nil
}

// loadTreasureDeck loads treasure_deck.json and all referenced cards
func (cm *ContentManager) loadTreasureDeck(deckPath string, basePath string) error {
	data, err := os.ReadFile(deckPath)
	if err != nil {
		return fmt.Errorf("failed to read treasure deck: %w", err)
	}

	var deck TreasureDeck
	if err := json.Unmarshal(data, &deck); err != nil {
		return fmt.Errorf("failed to parse treasure deck: %w", err)
	}

	// Load each treasure card
	for _, ref := range deck.Cards {
		cardPath := filepath.Join(basePath, ref.Path)
		card, err := cm.loadTreasureCard(cardPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load treasure card %s: %v", ref.ID, err)
			continue
		}
		cm.treasureCards[card.ID] = card
	}

	return nil
}

// loadSpellDeck loads spell_deck.json and all referenced spells
func (cm *ContentManager) loadSpellDeck(deckPath string, basePath string) error {
	data, err := os.ReadFile(deckPath)
	if err != nil {
		return fmt.Errorf("failed to read spell deck: %w", err)
	}

	var deck SpellDeck
	if err := json.Unmarshal(data, &deck); err != nil {
		return fmt.Errorf("failed to parse spell deck: %w", err)
	}

	// Load each spell card
	for _, ref := range deck.Spells {
		cardPath := filepath.Join(basePath, ref.Path)
		card, err := cm.loadSpellCard(cardPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load spell card %s: %v", ref.ID, err)
			continue
		}
		cm.spellCards[card.ID] = card
	}

	return nil
}

// loadDreadSpellDeck loads dread_spell_deck.json and all referenced spells
func (cm *ContentManager) loadDreadSpellDeck(deckPath string, basePath string) error {
	data, err := os.ReadFile(deckPath)
	if err != nil {
		return fmt.Errorf("failed to read dread spell deck: %w", err)
	}

	var deck DreadSpellDeck
	if err := json.Unmarshal(data, &deck); err != nil {
		return fmt.Errorf("failed to parse dread spell deck: %w", err)
	}

	// Load each dread spell card
	for _, ref := range deck.Spells {
		cardPath := filepath.Join(basePath, ref.Path)
		card, err := cm.loadSpellCard(cardPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load dread spell card %s: %v", ref.ID, err)
			continue
		}
		cm.dreadSpellCards[card.ID] = card
	}

	return nil
}

// loadItemCard loads a single item card (equipment or artifact)
func (cm *ContentManager) loadItemCard(path string) (*ItemCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read item card: %w", err)
	}

	var card ItemCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse item card: %w", err)
	}

	return &card, nil
}

// loadTreasureCard loads a single treasure card
func (cm *ContentManager) loadTreasureCard(path string) (*TreasureCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read treasure card: %w", err)
	}

	var card TreasureCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse treasure card: %w", err)
	}

	return &card, nil
}

// loadSpellCard loads a single spell or dread spell card
func (cm *ContentManager) loadSpellCard(path string) (*SpellCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spell card: %w", err)
	}

	var card SpellCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse spell card: %w", err)
	}

	return &card, nil
}

// GetEquipmentCard retrieves an equipment card by ID
func (cm *ContentManager) GetEquipmentCard(id string) (*ItemCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.equipmentCards[id]
	return card, ok
}

// GetArtifactCard retrieves an artifact card by ID
func (cm *ContentManager) GetArtifactCard(id string) (*ItemCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.artifactCards[id]
	return card, ok
}

// GetTreasureCard retrieves a treasure card by ID
func (cm *ContentManager) GetTreasureCard(id string) (*TreasureCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.treasureCards[id]
	return card, ok
}

// GetSpellCard retrieves a spell card by ID
func (cm *ContentManager) GetSpellCard(id string) (*SpellCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.spellCards[id]
	return card, ok
}

// GetDreadSpellCard retrieves a dread spell card by ID
func (cm *ContentManager) GetDreadSpellCard(id string) (*SpellCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.dreadSpellCards[id]
	return card, ok
}

// GetAllEquipment returns all equipment cards
func (cm *ContentManager) GetAllEquipment() map[string]*ItemCard {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]*ItemCard, len(cm.equipmentCards))
	for k, v := range cm.equipmentCards {
		result[k] = v
	}
	return result
}

// GetAllArtifacts returns all artifact cards
func (cm *ContentManager) GetAllArtifacts() map[string]*ItemCard {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	result := make(map[string]*ItemCard, len(cm.artifactCards))
	for k, v := range cm.artifactCards {
		result[k] = v
	}
	return result
}

// GetAllTreasures returns all treasure cards
func (cm *ContentManager) GetAllTreasures() map[string]*TreasureCard {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	result := make(map[string]*TreasureCard, len(cm.treasureCards))
	for k, v := range cm.treasureCards {
		result[k] = v
	}
	return result
}

// GetCampaign returns the loaded campaign metadata
func (cm *ContentManager) GetCampaign() *CampaignMetadata {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.campaign
}

// loadHeroes loads all hero character definitions from the heroes directory
func (cm *ContentManager) loadHeroes(heroesPath string) error {
	// Read all .json files in the heroes directory
	entries, err := os.ReadDir(heroesPath)
	if err != nil {
		return fmt.Errorf("failed to read heroes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		heroPath := filepath.Join(heroesPath, entry.Name())
		hero, err := cm.loadHeroCard(heroPath)
		if err != nil {
			cm.logger.Printf("Warning: Failed to load hero %s: %v", entry.Name(), err)
			continue
		}
		cm.heroCards[hero.ID] = hero
	}

	return nil
}

// loadHeroCard loads a single hero card
func (cm *ContentManager) loadHeroCard(path string) (*HeroCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read hero card: %w", err)
	}

	var card HeroCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse hero card: %w", err)
	}

	return &card, nil
}

// GetHeroCard retrieves a hero card by ID
func (cm *ContentManager) GetHeroCard(id string) (*HeroCard, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	card, ok := cm.heroCards[id]
	return card, ok
}

// GetAllHeroes returns all hero cards
func (cm *ContentManager) GetAllHeroes() map[string]*HeroCard {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	result := make(map[string]*HeroCard, len(cm.heroCards))
	for k, v := range cm.heroCards {
		result[k] = v
	}
	return result
}
