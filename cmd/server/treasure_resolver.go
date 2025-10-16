package main

import (
	"fmt"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// TreasureResult represents the result of a treasure search
type TreasureResult struct {
	Success      bool          `json:"success"`
	FoundItems   []*ItemCard   `json:"foundItems"`
	FoundGold    int           `json:"foundGold"`
	IsEmpty      bool          `json:"isEmpty"`
	IsHazard     bool          `json:"isHazard"`
	HazardDamage int           `json:"hazardDamage"`
	EndTurn      bool          `json:"endTurn"`
	IsMonster    bool          `json:"isMonster"`
	MonsterType  string        `json:"monsterType,omitempty"`
	Message      string        `json:"message"`
	NoteID       string        `json:"noteId,omitempty"`
	CardDrawn    *TreasureCard `json:"cardDrawn,omitempty"`
}

// TreasureResolver resolves treasure searches
type TreasureResolver struct {
	contentManager *ContentManager
	treasureDeck   *TreasureDeckManager
	quest          *geometry.QuestDefinition
	consumedNotes  map[string]bool // Track which quest notes have been consumed
	logger         Logger
}

// NewTreasureResolver creates a new treasure resolver
func NewTreasureResolver(contentManager *ContentManager, treasureDeck *TreasureDeckManager, quest *geometry.QuestDefinition, logger Logger) *TreasureResolver {
	return &TreasureResolver{
		contentManager: contentManager,
		treasureDeck:   treasureDeck,
		quest:          quest,
		consumedNotes:  make(map[string]bool),
		logger:         logger,
	}
}

// ResolveTreasureSearch resolves a treasure search at a specific location
func (tr *TreasureResolver) ResolveTreasureSearch(heroID string, room int, position protocol.TileAddress, furnitureID string) (*TreasureResult, error) {
	tr.logger.Printf("Resolving treasure search for hero %s in room %d at (%d,%d)", heroID, room, position.X, position.Y)

	// First, check for quest treasure notes
	if tr.quest != nil && tr.quest.QuestNotes != nil {
		for noteID, note := range tr.quest.QuestNotes {
			// Check if note matches this location
			if note.Location.Room == room &&
				note.Location.X == position.X &&
				note.Location.Y == position.Y {

				// Check if furniture ID matches (if specified)
				if note.Location.FurnitureID != "" && note.Location.FurnitureID != furnitureID {
					continue
				}

				// Check if note has been consumed
				if note.ConsumedForParty && tr.consumedNotes[noteID] {
					tr.logger.Printf("Quest note %s already consumed", noteID)
					return &TreasureResult{
						Success: true,
						IsEmpty: true,
						Message: "This has already been searched.",
					}, nil
				}

				// Mark as consumed if applicable
				if note.ConsumedForParty {
					tr.consumedNotes[noteID] = true
				}

				// Resolve based on treasure type
				return tr.resolveQuestNote(note, noteID)
			}
		}
	}

	// No quest note found, draw from treasure deck
	return tr.drawFromTreasureDeck()
}

// resolveQuestNote resolves a specific quest treasure note
func (tr *TreasureResolver) resolveQuestNote(note *geometry.QuestTreasureNote, noteID string) (*TreasureResult, error) {
	tr.logger.Printf("Resolving quest note %s: %s", noteID, note.TreasureType)

	result := &TreasureResult{
		Success: true,
		NoteID:  noteID,
		Message: note.Description,
	}

	switch note.TreasureType {
	case "fixed":
		// Fixed treasure with items or gold
		if note.Gold > 0 {
			result.FoundGold = note.Gold
			tr.logger.Printf("Found %d gold from quest note %s", note.Gold, noteID)
		}

		if len(note.Items) > 0 {
			items := make([]*ItemCard, 0)
			for _, itemRef := range note.Items {
				// Try equipment first
				if item, ok := tr.contentManager.GetEquipmentCard(itemRef.ID); ok {
					items = append(items, item)
				} else if item, ok := tr.contentManager.GetArtifactCard(itemRef.ID); ok {
					items = append(items, item)
				} else {
					tr.logger.Printf("Warning: Item %s from quest note not found", itemRef.ID)
				}
			}
			result.FoundItems = items
			tr.logger.Printf("Found %d items from quest note %s", len(items), noteID)
		}

	case "empty":
		result.IsEmpty = true

	case "monster_modifier":
		// This is handled separately by the monster system
		result.Message = note.Description
	}

	return result, nil
}

// drawFromTreasureDeck draws a random treasure card from the deck
func (tr *TreasureResolver) drawFromTreasureDeck() (*TreasureResult, error) {
	card, err := tr.treasureDeck.DrawCard()
	if err != nil {
		return nil, fmt.Errorf("failed to draw treasure card: %w", err)
	}

	tr.logger.Printf("Drew treasure card: %s (type: %s)", card.Name, card.Type)

	result := &TreasureResult{
		Success:   true,
		Message:   card.Description,
		CardDrawn: card,
	}

	switch card.Type {
	case "gold":
		result.FoundGold = card.Value
		tr.logger.Printf("Found %d gold", card.Value)

	case "potion", "artifact":
		// Treasure deck can contain potions or artifacts
		// We need to convert the treasure card reference to the actual item
		if item, ok := tr.contentManager.GetEquipmentCard(card.ID); ok {
			result.FoundItems = []*ItemCard{item}
		} else if item, ok := tr.contentManager.GetArtifactCard(card.ID); ok {
			result.FoundItems = []*ItemCard{item}
		}

	case "hazard":
		result.IsHazard = true
		result.HazardDamage = card.Damage
		result.EndTurn = card.EndTurn
		tr.logger.Printf("Hazard! %d damage, end turn: %v", card.Damage, card.EndTurn)

	case "monster":
		result.IsMonster = true
		// The wandering monster type is specified in the quest
		if tr.quest != nil {
			result.MonsterType = tr.quest.WanderingMonster
		}
		tr.logger.Printf("Wandering monster! Type: %s", result.MonsterType)
	}

	return result, nil
}

// IsNoteConsumed checks if a quest note has been consumed
func (tr *TreasureResolver) IsNoteConsumed(noteID string) bool {
	return tr.consumedNotes[noteID]
}
