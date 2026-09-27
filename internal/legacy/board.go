// Package legacy reads the original board.json and quest JSON formats so they
// can be imported as map-creator documents (see internal/maps and internal/seed).
package legacy

import (
	"encoding/json"
	"fmt"
	"os"
)

// TileCoordinate represents a single tile position
type TileCoordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Room represents a room with a list of tiles
type Room struct {
	ID    int              `json:"id"`
	Name  string           `json:"name"`
	Tiles []TileCoordinate `json:"tiles"`
}

// BoardDefinition represents the static board layout
type BoardDefinition struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	Rooms []Room `json:"rooms"`
}

// LoadBoardFromFile loads a board definition from a JSON file
func LoadBoardFromFile(filepath string) (*BoardDefinition, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read board file: %w", err)
	}

	var board BoardDefinition
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to parse board JSON: %w", err)
	}

	return &board, nil
}
