package geometry

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
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	X              int      `json:"x"`
	Y              int      `json:"y"`
	Room           int      `json:"room"`
	BlocksMovement bool     `json:"blocks_movement"`
	Contains       []string `json:"contains"`
	Notes          string   `json:"notes"`
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

// QuestDefinition represents the complete quest configuration
type QuestDefinition struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Difficulty       string              `json:"difficulty"`
	StartingRoom     int                 `json:"starting_room"`
	WanderingMonster string              `json:"wandering_monster"`
	SpecialRules     QuestSpecialRules   `json:"special_rules"`
	Doors            []QuestDoor         `json:"doors"`
	BlockingWalls    []QuestBlockingWall `json:"blocking_walls"`
	Monsters         []QuestMonster      `json:"monsters"`
	Furniture        []QuestFurniture    `json:"furniture"`
	Objectives       []QuestObjective    `json:"objectives"`
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

// ConvertQuestDoorsToEdges converts quest doors to EdgeAddress structures
func ConvertQuestDoorsToEdges(doors []QuestDoor) []EdgeAddress {
	edges := make([]EdgeAddress, len(doors))
	for i, door := range doors {
		orientation := Vertical
		if door.Orientation == "horizontal" {
			orientation = Horizontal
		}

		edges[i] = EdgeAddress{
			X:           door.X,
			Y:           door.Y,
			Orientation: orientation,
		}
	}
	return edges
}

// ConvertQuestBlockingWallsToEdges converts quest blocking walls to EdgeAddress structures
func ConvertQuestBlockingWallsToEdges(walls []QuestBlockingWall) []EdgeAddress {
	var edges []EdgeAddress

	for _, wall := range walls {
		orientation := Vertical
		if wall.Orientation == "horizontal" {
			orientation = Horizontal
		}

		// Handle multi-tile walls
		size := wall.Size
		if size <= 0 {
			size = 1 // Default to single tile
		}

		for i := 0; i < size; i++ {
			edge := EdgeAddress{
				X:           wall.X,
				Y:           wall.Y,
				Orientation: orientation,
			}

			// Offset for multi-tile walls
			if orientation == Horizontal {
				edge.X += i
			} else {
				edge.Y += i
			}

			edges = append(edges, edge)
		}
	}

	return edges
}

// FindStartingTileInRoom finds the first available tile in a specific room
func FindStartingTileInRoom(board *BoardDefinition, roomID int) (int, int, error) {
	for _, room := range board.Rooms {
		if room.ID == roomID {
			if len(room.Tiles) == 0 {
				return 0, 0, fmt.Errorf("room %d has no tiles", roomID)
			}
			// Return the first tile in the room
			return room.Tiles[0].X, room.Tiles[0].Y, nil
		}
	}
	return 0, 0, fmt.Errorf("room %d not found", roomID)
}
