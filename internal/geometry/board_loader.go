package geometry

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

// CreateSegmentFromBoard converts a BoardDefinition into a Segment
// by generating walls around room boundaries, and leaving corridors open.
func CreateSegmentFromBoard(board *BoardDefinition) Segment {
	segment := Segment{
		ID:              board.ID,
		Width:           board.Dimensions.Width,
		Height:          board.Dimensions.Height,
		WallsVertical:   []EdgeAddress{},
		WallsHorizontal: []EdgeAddress{},
		DoorSockets:     []EdgeAddress{},
	}

	// Create a map of all room tiles for easy lookup
	roomTiles := make(map[string]bool)
	for _, room := range board.Rooms {
		for _, tile := range room.Tiles {
			key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
			roomTiles[key] = true
		}
	}

	// Generate walls around room boundaries
	segment.WallsVertical, segment.WallsHorizontal = generateWallsFromRooms(board, roomTiles)

	return segment
}

// generateWallsFromRooms creates wall edges around room boundaries
func generateWallsFromRooms(board *BoardDefinition, roomTiles map[string]bool) ([]EdgeAddress, []EdgeAddress) {
	var verticalWalls []EdgeAddress
	var horizontalWalls []EdgeAddress

	// Create region map for each tile to check different regions
	regionMap := CreateRegionMapFromBoard(board)

	// Check each tile position to see if walls are needed
	for y := 0; y < board.Dimensions.Height; y++ {
		for x := 0; x < board.Dimensions.Width; x++ {
			currentIdx := y*board.Dimensions.Width + x
			currentRegion := regionMap.TileRegionIDs[currentIdx]

			// Check need for vertical wall to the right (between x,y and x+1,y)
			if x < board.Dimensions.Width-1 {
				rightIdx := y*board.Dimensions.Width + (x + 1)
				rightRegion := regionMap.TileRegionIDs[rightIdx]

				// Add wall if regions are different
				// Wall should be at the left edge of the right tile (x+1,y)
				if currentRegion != rightRegion {
					verticalWalls = append(verticalWalls, EdgeAddress{
						X:           x + 1,
						Y:           y,
						Orientation: Vertical,
					})
				}
			}

			// Check need for horizontal wall below (between x,y and x,y+1)
			if y < board.Dimensions.Height-1 {
				belowIdx := (y+1)*board.Dimensions.Width + x
				belowRegion := regionMap.TileRegionIDs[belowIdx]

				// Add wall if regions are different
				// Wall should be at the top edge of the below tile (x,y+1)
				if currentRegion != belowRegion {
					horizontalWalls = append(horizontalWalls, EdgeAddress{
						X:           x,
						Y:           y + 1,
						Orientation: Horizontal,
					})
				}
			}
		}
	}

	// Add boundary walls
	verticalWalls, horizontalWalls = addBoundaryWalls(board, verticalWalls, horizontalWalls)

	return verticalWalls, horizontalWalls
}

// addBoundaryWalls adds walls around the entire board perimeter
func addBoundaryWalls(board *BoardDefinition, verticalWalls, horizontalWalls []EdgeAddress) ([]EdgeAddress, []EdgeAddress) {
	// Top and bottom boundary walls
	for x := 0; x < board.Dimensions.Width; x++ {
		// Top boundary - top edge of tile (x,0)
		horizontalWalls = append(horizontalWalls, EdgeAddress{
			X:           x,
			Y:           0,
			Orientation: Horizontal,
		})
		// Bottom boundary - top edge of tile (x,Height) which is outside the board
		horizontalWalls = append(horizontalWalls, EdgeAddress{
			X:           x,
			Y:           board.Dimensions.Height,
			Orientation: Horizontal,
		})
	}

	// Left and right boundary walls
	for y := 0; y < board.Dimensions.Height; y++ {
		// Left boundary - left edge of tile (0,y)
		verticalWalls = append(verticalWalls, EdgeAddress{
			X:           0,
			Y:           y,
			Orientation: Vertical,
		})
		// Right boundary - left edge of tile (Width,y) which is outside the board
		verticalWalls = append(verticalWalls, EdgeAddress{
			X:           board.Dimensions.Width,
			Y:           y,
			Orientation: Vertical,
		})
	}

	return verticalWalls, horizontalWalls
}

// CreateRegionMapFromBoard creates a RegionMap where each room is a separate region
func CreateRegionMapFromBoard(board *BoardDefinition) RegionMap {
	totalTiles := board.Dimensions.Width * board.Dimensions.Height
	tileRegionIDs := make([]int, totalTiles)

	// Initialize all tiles as corridor (region 0)
	for i := range tileRegionIDs {
		tileRegionIDs[i] = 0
	}

	// Assign room tiles to their respective regions
	for _, room := range board.Rooms {
		for _, tile := range room.Tiles {
			idx := tile.Y*board.Dimensions.Width + tile.X
			if idx >= 0 && idx < totalTiles {
				tileRegionIDs[idx] = room.ID
			}
		}
	}

	// Count total regions (max room ID + 1 for corridor region 0)
	maxRegion := 0
	for _, room := range board.Rooms {
		if room.ID > maxRegion {
			maxRegion = room.ID
		}
	}

	return RegionMap{
		TileRegionIDs: tileRegionIDs,
		RegionsCount:  maxRegion + 1, // +1 to include region 0 (corridors)
	}
}
