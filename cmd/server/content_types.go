package main

import "encoding/json"

// ItemCard represents any equippable item (weapons, armor, potions, etc.)
type ItemCard struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Category           string            `json:"category,omitempty"`
	Type               string            `json:"type"`    // "weapon", "armor", "potion", "jewelry", "equipment"
	Subtype            string            `json:"subtype"` // "melee", "ranged", "body", "helmet", "shield", etc.
	Slot               string            `json:"slot,omitempty"`
	AttackDice         int               `json:"attack_dice"`
	DefenseDice        int               `json:"defense_dice"`
	AttackBonus        int               `json:"attack_bonus"`
	DefenseBonus       int               `json:"defense_bonus"`
	BodyBonus          int               `json:"body_bonus"`
	MindBonus          int               `json:"mind_bonus"`
	MovementBonus      int               `json:"movement_bonus"`
	AttackDiagonal     bool              `json:"attack_diagonal"`
	AttackAdjacent     bool              `json:"attack_adjacent"`
	Ranged             bool              `json:"ranged"`
	Throwable          bool              `json:"throwable"`
	Range              int               `json:"range"`
	Uses               int               `json:"uses,omitempty"`
	UsesPerQuest       int               `json:"uses_per_quest,omitempty"`
	Cost               int               `json:"cost"`
	UsableBy           []string          `json:"usable_by"`
	Restrictions       []string          `json:"restrictions,omitempty"`
	Effect             *EffectDefinition `json:"effect,omitempty"`       // Structured effect object
	UsageRestrictions  *UsageRestriction `json:"usage_restrictions,omitempty"`
	Description        string            `json:"description"`
	CardImage          string            `json:"card_image"`
}

// TreasureCard represents a treasure card from the deck
type TreasureCard struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Category          string            `json:"category,omitempty"`
	Type              string            `json:"type"` // "gold", "potion", "monster", "hazard"
	Subtype           string            `json:"subtype,omitempty"`
	Value             int               `json:"value,omitempty"`         // Gold value
	Uses              int               `json:"uses,omitempty"`
	Effect            *EffectDefinition `json:"effect,omitempty"`        // Structured effect object
	UsageRestrictions *UsageRestriction `json:"usage_restrictions,omitempty"`
	Damage            int               `json:"damage,omitempty"`        // Hazard damage
	EndTurn           bool              `json:"end_turn,omitempty"`      // Does hazard end turn?
	SpecialMechanic   string            `json:"special_mechanic,omitempty"`
	ReturnToDeck      bool              `json:"return_to_deck"`
	Description       string            `json:"description"`
	CardImage         string            `json:"card_image"`
}

// SpellCard represents a spell card
type SpellCard struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "spell" or "dread_spell"
	Category    string   `json:"category"`
	Target      string   `json:"target"`
	Range       int      `json:"range,omitempty"`
	Effect      string   `json:"effect"`
	EffectValue any      `json:"effect_value,omitempty"`
	AttackType  string   `json:"attack_type,omitempty"`
	Duration    string   `json:"duration,omitempty"`
	UsableBy    []string `json:"usable_by,omitempty"`
	Description string   `json:"description"`
	CardImage   string   `json:"card_image"`
}

// CampaignMetadata represents campaign.json
type CampaignMetadata struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Decks        CampaignDecks          `json:"decks"`
	ContentPaths CampaignContentPaths   `json:"content_paths"`
	Quests       []CampaignQuestRef     `json:"quests"`
}

type CampaignDecks struct {
	Equipment   string `json:"equipment"`
	Artifacts   string `json:"artifacts"`
	Treasures   string `json:"treasures"`
	Spells      string `json:"spells"`
	DreadSpells string `json:"dread_spells"`
}

type CampaignContentPaths struct {
	Monsters  string `json:"monsters"`
	Heroes    string `json:"heroes"`
	Furniture string `json:"furniture"`
	QuestsDir string `json:"quests_dir"`
}

type CampaignQuestRef struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Order int    `json:"order"`
	Name  string `json:"name,omitempty"`
}

// EquipmentDeck represents equipment_deck.json
type EquipmentDeck struct {
	Campaign    string              `json:"campaign"`
	DeckType    string              `json:"deck_type"`
	Description string              `json:"description,omitempty"`
	Items       []DeckItemReference `json:"items"`
}

// ArtifactDeck represents artifact_deck.json
type ArtifactDeck struct {
	Campaign    string              `json:"campaign"`
	DeckType    string              `json:"deck_type"`
	Description string              `json:"description,omitempty"`
	Items       []DeckItemReference `json:"items"`
}

// TreasureDeck represents treasure_deck.json
type TreasureDeck struct {
	Campaign      string             `json:"campaign"`
	DeckType      string             `json:"deck_type"`
	Description   string             `json:"description,omitempty"`
	ShuffleOnLoad bool               `json:"shuffle_on_load"`
	Cards         []TreasureDeckCard `json:"cards"`
}

type TreasureDeckCard struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// SpellDeck represents spell_deck.json
type SpellDeck struct {
	Campaign    string              `json:"campaign"`
	DeckType    string              `json:"deck_type"`
	Description string              `json:"description,omitempty"`
	Spells      []SpellDeckReference `json:"spells"`
}

type SpellDeckReference struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Element string `json:"element,omitempty"`
}

// DreadSpellDeck represents dread_spell_deck.json
type DreadSpellDeck struct {
	Campaign    string              `json:"campaign"`
	DeckType    string              `json:"deck_type"`
	Description string              `json:"description,omitempty"`
	Spells      []DeckItemReference `json:"spells"`
}

// DeckItemReference is a generic reference to a card file
type DeckItemReference struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// EffectDefinition represents a structured effect object for items and treasures
// This is a flexible structure that can handle all effect types via map[string]any
type EffectDefinition struct {
	Type string         `json:"type"` // e.g., "dual_mode_attack", "restore_points", "bonus_dice"
	Data map[string]any `json:"-"`    // All other fields captured here
}

// UnmarshalJSON custom unmarshaler to capture all fields
func (e *EffectDefinition) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to get all fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract the type field
	if typeVal, ok := raw["type"].(string); ok {
		e.Type = typeVal
	}

	// Store all fields (including type) in Data
	e.Data = raw

	return nil
}

// MarshalJSON custom marshaler
func (e *EffectDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Data)
}

// UsageRestriction defines limits on when/how items can be used
type UsageRestriction struct {
	MaxPerTurn     int    `json:"max_per_turn,omitempty"`
	Timing         string `json:"timing,omitempty"`         // "before_movement", "during_combat", "any_time"
	RequiresAction bool   `json:"requires_action,omitempty"`
}

// HeroCard represents a hero character definition
// Note: Uses HeroStats from turn_system.go
type HeroCard struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Class             string                `json:"class"`
	Description       string                `json:"description"`
	Stats             HeroStats             `json:"stats"`
	StartingEquipment HeroStartingEquipment `json:"startingEquipment"`
	SpecialAbilities  []string              `json:"specialAbilities"`
}

// HeroStartingEquipment defines what items a hero starts with
type HeroStartingEquipment struct {
	Weapons []string `json:"weapons"`
	Armor   []string `json:"armor"`
	Items   []string `json:"items"`
}
