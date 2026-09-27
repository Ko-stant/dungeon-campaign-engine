package legacy

import (
	"encoding/json"
	"fmt"
	"os"
)

// QuestDoor represents a door in a quest
type QuestDoor struct {
	ID          string `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
	State       string `json:"state"`
	Type        string `json:"type"`
	Notes       string `json:"notes"`
}

// QuestBlockingWall represents a wall that blocks corridor access
type QuestBlockingWall struct {
	ID          string `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
	Notes       string `json:"notes"`
	Size        int    `json:"size"`
}

// QuestMonster represents a monster placement
type QuestMonster struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Room  int    `json:"room"`
	Notes string `json:"notes"`
}

// QuestFurniture represents furniture placement
type QuestFurniture struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	X                  int      `json:"x"`
	Y                  int      `json:"y"`
	Room               int      `json:"room"`
	Rotation           int      `json:"rotation,omitempty"`              // 0, 90, 180, 270 degrees
	SwapAspectOnRotate bool     `json:"swap_aspect_on_rotate,omitempty"` // Whether to swap width/height for 90/270 rotations
	BlocksMovement     bool     `json:"blocks_movement"`
	Contains           []string `json:"contains"`
	Notes              string   `json:"notes"`
}

// QuestObjective represents a quest objective
type QuestObjective struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

// QuestSpecialRules represents special quest rules
type QuestSpecialRules struct {
	HasTraps       bool   `json:"has_traps"`
	HasSecretDoors bool   `json:"has_secret_doors"`
	Notes          string `json:"notes"`
}

// QuestTreasureNote represents a quest-specific treasure note
type QuestTreasureNote struct {
	NoteID           string           `json:"note_id"`
	Location         TreasureLocation `json:"location"`
	Description      string           `json:"description"`
	TreasureType     string           `json:"treasure_type"` // "fixed", "empty", "monster_modifier"
	Items            []ItemReference  `json:"items,omitempty"`
	Gold             int              `json:"gold,omitempty"`
	MonsterModifier  *MonsterModifier `json:"monster_modifier,omitempty"`
	ConsumedForParty bool             `json:"consumed_for_party"`
}

type TreasureLocation struct {
	Room        int    `json:"room"`
	FurnitureID string `json:"furniture_id,omitempty"`
	MonsterID   string `json:"monster_id,omitempty"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

type ItemReference struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type MonsterModifier struct {
	MonsterID        string `json:"monster_id"`
	AttackDiceBonus  int    `json:"attack_dice_bonus,omitempty"`
	DefenseDiceBonus int    `json:"defense_dice_bonus,omitempty"`
}

// QuestDefinition represents the complete quest configuration
type QuestDefinition struct {
	ID               string                        `json:"id"`
	Name             string                        `json:"name"`
	Description      string                        `json:"description"`
	Difficulty       string                        `json:"difficulty"`
	StartingRoom     int                           `json:"starting_room"`
	WanderingMonster string                        `json:"wandering_monster"`
	SpecialRules     QuestSpecialRules             `json:"special_rules"`
	Doors            []QuestDoor                   `json:"doors"`
	BlockingWalls    []QuestBlockingWall           `json:"blocking_walls"`
	Monsters         []QuestMonster                `json:"monsters"`
	Furniture        []QuestFurniture              `json:"furniture"`
	Objectives       []QuestObjective              `json:"objectives"`
	QuestNotes       map[string]*QuestTreasureNote `json:"quest_notes,omitempty"`
}

// LoadQuestFromFile loads a quest definition from a JSON file
func LoadQuestFromFile(filepath string) (*QuestDefinition, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read quest file: %w", err)
	}

	var quest QuestDefinition
	if err := json.Unmarshal(data, &quest); err != nil {
		return nil, fmt.Errorf("failed to parse quest JSON: %w", err)
	}

	return &quest, nil
}
