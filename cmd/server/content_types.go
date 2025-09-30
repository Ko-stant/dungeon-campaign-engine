package main

// ItemCard represents any equippable item (weapons, armor, potions, etc.)
type ItemCard struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`    // "weapon", "armor", "potion", "jewelry", "equipment"
	Subtype        string   `json:"subtype"` // "melee", "ranged", "body", "helmet", "shield", etc.
	Slot           string   `json:"slot,omitempty"`
	AttackDice     int      `json:"attack_dice"`
	DefenseDice    int      `json:"defense_dice"`
	AttackBonus    int      `json:"attack_bonus"`
	DefenseBonus   int      `json:"defense_bonus"`
	AttackDiagonal bool     `json:"attack_diagonal"`
	AttackAdjacent bool     `json:"attack_adjacent"`
	Ranged         bool     `json:"ranged"`
	Range          int      `json:"range"`
	Cost           int      `json:"cost"`
	UsableBy       []string `json:"usable_by"`
	Effect         string   `json:"effect,omitempty"`       // For potions/artifacts
	EffectValue    any      `json:"effect_value,omitempty"` // Can be int or string (e.g., "1d6")
	UsageRules     []string `json:"usage_rules,omitempty"`
	Description    string   `json:"description"`
	CardImage      string   `json:"card_image"`
}

// TreasureCard represents a treasure card from the deck
type TreasureCard struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"` // "gold", "potion", "monster", "hazard"
	Subtype         string   `json:"subtype,omitempty"`
	Value           int      `json:"value,omitempty"`         // Gold value
	Effect          string   `json:"effect,omitempty"`        // Potion/artifact effect
	EffectValue     any      `json:"effect_value,omitempty"`  // Can be int or string
	Damage          int      `json:"damage,omitempty"`        // Hazard damage
	EndTurn         bool     `json:"end_turn,omitempty"`      // Does hazard end turn?
	SpecialMechanic string   `json:"special_mechanic,omitempty"`
	UsageRules      []string `json:"usage_rules,omitempty"`
	ReturnToDeck    bool     `json:"return_to_deck"`
	Description     string   `json:"description"`
	CardImage       string   `json:"card_image"`
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
