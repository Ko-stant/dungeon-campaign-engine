package main

import (
	"fmt"
	"sync"
)

// HeroInventory represents a hero's inventory
type HeroInventory struct {
	HeroID    string                `json:"hero_id"`
	Gold      int                   `json:"gold"`
	Equipment map[string]*ItemCard  `json:"equipment"` // slot -> equipped item
	Carried   []*ItemCard           `json:"carried"`   // Items not equipped
	Spells    []*SpellCard          `json:"spells"`    // Spell cards (for Wizard/Elf)
}

// InventoryManager manages hero inventories
type InventoryManager struct {
	inventories    map[string]*HeroInventory // heroID -> inventory
	contentManager *ContentManager
	logger         Logger
	mutex          sync.RWMutex
}

// NewInventoryManager creates a new inventory manager
func NewInventoryManager(contentManager *ContentManager, logger Logger) *InventoryManager {
	return &InventoryManager{
		inventories:    make(map[string]*HeroInventory),
		contentManager: contentManager,
		logger:         logger,
	}
}

// InitializeHeroInventory creates a new inventory for a hero
func (im *InventoryManager) InitializeHeroInventory(heroID string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	if _, exists := im.inventories[heroID]; exists {
		return fmt.Errorf("inventory for hero %s already exists", heroID)
	}

	im.inventories[heroID] = &HeroInventory{
		HeroID:    heroID,
		Gold:      0,
		Equipment: make(map[string]*ItemCard),
		Carried:   make([]*ItemCard, 0),
		Spells:    make([]*SpellCard, 0),
	}

	im.logger.Printf("Initialized inventory for hero %s", heroID)
	return nil
}

// GetInventory retrieves a hero's inventory
func (im *InventoryManager) GetInventory(heroID string) (*HeroInventory, error) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return nil, fmt.Errorf("inventory for hero %s not found", heroID)
	}

	return inventory, nil
}

// AddItem adds an item to a hero's carried items
func (im *InventoryManager) AddItem(heroID string, itemID string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return fmt.Errorf("inventory for hero %s not found", heroID)
	}

	// Check if it's equipment or artifact
	var item *ItemCard
	if equip, ok := im.contentManager.GetEquipmentCard(itemID); ok {
		item = equip
	} else if artifact, ok := im.contentManager.GetArtifactCard(itemID); ok {
		item = artifact
	} else {
		return fmt.Errorf("item %s not found in content", itemID)
	}

	inventory.Carried = append(inventory.Carried, item)
	im.logger.Printf("Added item %s to hero %s inventory", itemID, heroID)
	return nil
}

// AddGold adds gold to a hero's inventory
func (im *InventoryManager) AddGold(heroID string, amount int) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return fmt.Errorf("inventory for hero %s not found", heroID)
	}

	inventory.Gold += amount
	im.logger.Printf("Added %d gold to hero %s (total: %d)", amount, heroID, inventory.Gold)
	return nil
}

// EquipItem equips an item from carried items
func (im *InventoryManager) EquipItem(heroID string, itemID string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return fmt.Errorf("inventory for hero %s not found", heroID)
	}

	// Find item in carried items
	var itemToEquip *ItemCard
	itemIndex := -1
	for i, item := range inventory.Carried {
		if item.ID == itemID {
			itemToEquip = item
			itemIndex = i
			break
		}
	}

	if itemToEquip == nil {
		return fmt.Errorf("item %s not found in carried items", itemID)
	}

	// Determine slot based on item type and subtype
	slot := determineSlot(itemToEquip)
	if slot == "" {
		return fmt.Errorf("cannot determine equipment slot for item %s", itemID)
	}

	// Check if slot already occupied
	if existing, occupied := inventory.Equipment[slot]; occupied {
		// Unequip existing item and move to carried
		inventory.Carried = append(inventory.Carried, existing)
		im.logger.Printf("Unequipped %s from slot %s", existing.ID, slot)
	}

	// Equip new item
	inventory.Equipment[slot] = itemToEquip

	// Remove from carried items
	inventory.Carried = append(inventory.Carried[:itemIndex], inventory.Carried[itemIndex+1:]...)

	im.logger.Printf("Equipped item %s to slot %s for hero %s", itemID, slot, heroID)
	return nil
}

// UnequipItem removes an equipped item and moves it to carried
func (im *InventoryManager) UnequipItem(heroID string, slot string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return fmt.Errorf("inventory for hero %s not found", heroID)
	}

	item, occupied := inventory.Equipment[slot]
	if !occupied {
		return fmt.Errorf("no item equipped in slot %s", slot)
	}

	// Move to carried
	inventory.Carried = append(inventory.Carried, item)

	// Remove from equipment
	delete(inventory.Equipment, slot)

	im.logger.Printf("Unequipped item %s from slot %s for hero %s", item.ID, slot, heroID)
	return nil
}

// AddSpell adds a spell to a hero's spell list
func (im *InventoryManager) AddSpell(heroID string, spellID string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	inventory, exists := im.inventories[heroID]
	if !exists {
		return fmt.Errorf("inventory for hero %s not found", heroID)
	}

	spell, ok := im.contentManager.GetSpellCard(spellID)
	if !ok {
		return fmt.Errorf("spell %s not found in content", spellID)
	}

	inventory.Spells = append(inventory.Spells, spell)
	im.logger.Printf("Added spell %s to hero %s", spellID, heroID)
	return nil
}

// determineSlot determines the equipment slot based on item type and subtype
func determineSlot(item *ItemCard) string {
	switch item.Type {
	case "weapon":
		return "weapon"
	case "armor":
		switch item.Subtype {
		case "body":
			return "armor_body"
		case "helmet":
			return "armor_helmet"
		case "shield":
			return "shield"
		case "bracers", "gloves":
			return "gloves"
		case "boots":
			return "boots"
		default:
			return "armor_body"
		}
	case "jewelry":
		switch item.Subtype {
		case "ring":
			return "ring"
		case "amulet":
			return "amulet"
		}
	}
	return ""
}
