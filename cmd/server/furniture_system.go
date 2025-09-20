package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// FurnitureDefinition represents the complete furniture metadata from JSON files
type FurnitureDefinition struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	BlocksLineOfSight bool   `json:"blocksLineOfSight"`
	BlocksMovement    bool   `json:"blocksMovement"`
	GridSize          struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rendering struct {
		TileImage        string `json:"tileImage"`
		TileImageCleaned string `json:"tileImageCleaned"`
		PixelDimensions  struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"pixelDimensions"`
	} `json:"rendering"`
	GameplayProperties struct {
		Searchable     bool           `json:"searchable,omitempty"`
		Container      bool           `json:"container,omitempty"`
		Interactable   bool           `json:"interactable,omitempty"`
		CustomProperties map[string]any `json:"customProperties,omitempty"`
	} `json:"gameplayProperties,omitempty"`
}

// FurnitureInstance represents a placed furniture piece in the game world
type FurnitureInstance struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"` // References FurnitureDefinition.ID
	Position   protocol.TileAddress  `json:"position"`
	Room       int                   `json:"room"`
	Contains   []string              `json:"contains,omitempty"`
	Definition *FurnitureDefinition  `json:"-"` // Runtime reference, not serialized
}

// FurnitureSystem manages furniture definitions and instances
type FurnitureSystem struct {
	definitions map[string]*FurnitureDefinition // furniture type -> definition
	instances   map[string]*FurnitureInstance   // furniture instance ID -> instance
	logger      *log.Logger
}

// NewFurnitureSystem creates a new furniture management system
func NewFurnitureSystem(logger *log.Logger) *FurnitureSystem {
	return &FurnitureSystem{
		definitions: make(map[string]*FurnitureDefinition),
		instances:   make(map[string]*FurnitureInstance),
		logger:      logger,
	}
}

// LoadFurnitureDefinitions loads all furniture definitions from the content directory
func (fs *FurnitureSystem) LoadFurnitureDefinitions(contentPath string) error {
	furnitureDir := filepath.Join(contentPath, "furniture")

	// Check if furniture directory exists
	if _, err := os.Stat(furnitureDir); os.IsNotExist(err) {
		fs.logger.Printf("Furniture directory does not exist: %s", furnitureDir)
		return nil // Not an error, just no furniture to load
	}

	// Read all JSON files in the furniture directory
	files, err := filepath.Glob(filepath.Join(furnitureDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to glob furniture files: %v", err)
	}

	fs.logger.Printf("Loading furniture definitions from %d files", len(files))

	for _, file := range files {
		if err := fs.loadFurnitureDefinition(file); err != nil {
			fs.logger.Printf("Failed to load furniture definition from %s: %v", file, err)
			continue // Continue loading other files
		}
	}

	fs.logger.Printf("Loaded %d furniture definitions", len(fs.definitions))
	return nil
}

// loadFurnitureDefinition loads a single furniture definition from a JSON file
func (fs *FurnitureSystem) loadFurnitureDefinition(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var def FurnitureDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Validate required fields
	if def.ID == "" {
		return fmt.Errorf("furniture definition missing required field: id")
	}
	if def.Name == "" {
		return fmt.Errorf("furniture definition missing required field: name")
	}

	fs.definitions[def.ID] = &def
	fs.logger.Printf("Loaded furniture definition: %s (%s)", def.ID, def.Name)

	return nil
}

// CreateFurnitureInstancesFromQuest creates furniture instances based on quest furniture placements
func (fs *FurnitureSystem) CreateFurnitureInstancesFromQuest(quest *geometry.QuestDefinition) error {
	for _, questFurniture := range quest.Furniture {
		// Look up the furniture definition
		definition, exists := fs.definitions[questFurniture.Type]
		if !exists {
			fs.logger.Printf("Warning: Unknown furniture type '%s' referenced in quest", questFurniture.Type)
			continue
		}

		// Create furniture instance
		instance := &FurnitureInstance{
			ID:   questFurniture.ID,
			Type: questFurniture.Type,
			Position: protocol.TileAddress{
				X: questFurniture.X,
				Y: questFurniture.Y,
			},
			Room:       questFurniture.Room,
			Contains:   questFurniture.Contains,
			Definition: definition,
		}

		fs.instances[instance.ID] = instance
		fs.logger.Printf("Created furniture instance: %s (%s) at (%d,%d) in room %d",
			instance.ID, instance.Type, instance.Position.X, instance.Position.Y, instance.Room)
	}

	fs.logger.Printf("Created %d furniture instances from quest", len(fs.instances))
	return nil
}

// GetDefinition returns a furniture definition by type
func (fs *FurnitureSystem) GetDefinition(furnitureType string) *FurnitureDefinition {
	return fs.definitions[furnitureType]
}

// GetInstance returns a furniture instance by ID
func (fs *FurnitureSystem) GetInstance(instanceID string) *FurnitureInstance {
	return fs.instances[instanceID]
}

// GetAllInstances returns all furniture instances
func (fs *FurnitureSystem) GetAllInstances() map[string]*FurnitureInstance {
	return fs.instances
}

// GetInstancesInRoom returns all furniture instances in a specific room
func (fs *FurnitureSystem) GetInstancesInRoom(roomID int) []*FurnitureInstance {
	var roomFurniture []*FurnitureInstance
	for _, instance := range fs.instances {
		if instance.Room == roomID {
			roomFurniture = append(roomFurniture, instance)
		}
	}
	return roomFurniture
}

// BlocksLineOfSight checks if furniture at a position blocks line of sight
func (fs *FurnitureSystem) BlocksLineOfSight(x, y int) bool {
	for _, instance := range fs.instances {
		if instance.Definition == nil {
			continue
		}

		// Check if the given position overlaps with this furniture
		if fs.positionOverlaps(x, y, instance) && instance.Definition.BlocksLineOfSight {
			return true
		}
	}
	return false
}

// BlocksMovement checks if furniture at a position blocks movement
func (fs *FurnitureSystem) BlocksMovement(x, y int) bool {
	for _, instance := range fs.instances {
		if instance.Definition == nil {
			continue
		}

		// Check if the given position overlaps with this furniture
		if fs.positionOverlaps(x, y, instance) && instance.Definition.BlocksMovement {
			return true
		}
	}
	return false
}

// positionOverlaps checks if a given x,y position overlaps with furniture instance
func (fs *FurnitureSystem) positionOverlaps(x, y int, instance *FurnitureInstance) bool {
	if instance.Definition == nil {
		return false
	}

	// Check if position is within the furniture's grid area
	startX := instance.Position.X
	startY := instance.Position.Y
	endX := startX + instance.Definition.GridSize.Width - 1
	endY := startY + instance.Definition.GridSize.Height - 1

	return x >= startX && x <= endX && y >= startY && y <= endY
}