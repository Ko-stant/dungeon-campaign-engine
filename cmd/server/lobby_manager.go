package main

import (
	"fmt"
	"sync"
)

// PlayerRole represents the role a player has chosen in the lobby
type PlayerRole string

const (
	RoleNone       PlayerRole = ""
	RoleGameMaster PlayerRole = "gamemaster"
	RoleHero       PlayerRole = "hero"
)

// PlayerLobbyInfo tracks a player's information in the lobby
type PlayerLobbyInfo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Role        PlayerRole `json:"role"`
	HeroClassID string     `json:"heroClassId"` // Only set if Role is RoleHero
	IsReady     bool       `json:"isReady"`
}

// LobbyState represents the current state of the game lobby
type LobbyState struct {
	Players       map[string]*PlayerLobbyInfo `json:"players"`
	CanStartGame  bool                        `json:"canStartGame"`
	GameStarted   bool                        `json:"gameStarted"`
	AvailableHeroes []string                  `json:"availableHeroes"`
}

// LobbyManager manages the pre-game lobby where players join and select roles
type LobbyManager struct {
	players         map[string]*PlayerLobbyInfo
	contentManager  *ContentManager
	mutex           sync.RWMutex
	gameStarted     bool
}

// NewLobbyManager creates a new lobby manager
func NewLobbyManager(contentManager *ContentManager) *LobbyManager {
	return &LobbyManager{
		players:        make(map[string]*PlayerLobbyInfo),
		contentManager: contentManager,
		gameStarted:    false,
	}
}

// AddPlayer adds a new player to the lobby
func (lm *LobbyManager) AddPlayer(playerID, playerName string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if lm.gameStarted {
		return fmt.Errorf("game has already started")
	}

	if _, exists := lm.players[playerID]; exists {
		return fmt.Errorf("player already in lobby")
	}

	lm.players[playerID] = &PlayerLobbyInfo{
		ID:      playerID,
		Name:    playerName,
		Role:    RoleNone,
		IsReady: false,
	}

	return nil
}

// RemovePlayer removes a player from the lobby
func (lm *LobbyManager) RemovePlayer(playerID string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if _, exists := lm.players[playerID]; !exists {
		return fmt.Errorf("player not in lobby")
	}

	delete(lm.players, playerID)
	return nil
}

// SetPlayerRole sets the role for a player
func (lm *LobbyManager) SetPlayerRole(playerID string, role PlayerRole, heroClassID string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	player, exists := lm.players[playerID]
	if !exists {
		return fmt.Errorf("player not in lobby")
	}

	// Validate role selection
	if role == RoleGameMaster {
		// Check if another player is already game master
		for _, p := range lm.players {
			if p.ID != playerID && p.Role == RoleGameMaster {
				return fmt.Errorf("game master role already taken")
			}
		}
		player.Role = RoleGameMaster
		player.HeroClassID = ""
	} else if role == RoleHero {
		// Validate hero class exists
		if heroClassID == "" {
			return fmt.Errorf("hero class must be specified for hero role")
		}
		if _, ok := lm.contentManager.GetHeroCard(heroClassID); !ok {
			return fmt.Errorf("invalid hero class: %s", heroClassID)
		}

		// Check if hero class is already taken
		for _, p := range lm.players {
			if p.ID != playerID && p.Role == RoleHero && p.HeroClassID == heroClassID {
				return fmt.Errorf("hero class %s already taken", heroClassID)
			}
		}

		player.Role = RoleHero
		player.HeroClassID = heroClassID
	} else {
		player.Role = RoleNone
		player.HeroClassID = ""
	}

	// Reset ready status when role changes
	player.IsReady = false

	return nil
}

// SetPlayerReady sets the ready status for a player
func (lm *LobbyManager) SetPlayerReady(playerID string, isReady bool) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	player, exists := lm.players[playerID]
	if !exists {
		return fmt.Errorf("player not in lobby")
	}

	if player.Role == RoleNone {
		return fmt.Errorf("player must select a role before readying up")
	}

	player.IsReady = isReady
	return nil
}

// CanStartGame checks if the game can be started
func (lm *LobbyManager) CanStartGame() bool {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	if lm.gameStarted {
		return false
	}

	if len(lm.players) < 2 {
		return false
	}

	hasGameMaster := false
	hasHero := false
	allReady := true

	for _, player := range lm.players {
		if player.Role == RoleGameMaster {
			hasGameMaster = true
		} else if player.Role == RoleHero {
			hasHero = true
		}

		if !player.IsReady {
			allReady = false
		}
	}

	return hasGameMaster && hasHero && allReady
}

// GetLobbyState returns the current lobby state
func (lm *LobbyManager) GetLobbyState() *LobbyState {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	// Copy players map to prevent external modification
	playersCopy := make(map[string]*PlayerLobbyInfo)
	for id, player := range lm.players {
		playerCopy := *player
		playersCopy[id] = &playerCopy
	}

	// Get available heroes from content manager
	availableHeroes := []string{}
	if lm.contentManager != nil {
		heroes := lm.contentManager.GetAllHeroes()
		for heroID := range heroes {
			availableHeroes = append(availableHeroes, heroID)
		}
	}

	return &LobbyState{
		Players:         playersCopy,
		CanStartGame:    lm.CanStartGame(),
		GameStarted:     lm.gameStarted,
		AvailableHeroes: availableHeroes,
	}
}

// StartGame marks the game as started and returns the player configurations
func (lm *LobbyManager) StartGame() (gameMasterID string, heroPlayers map[string]string, err error) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if lm.gameStarted {
		return "", nil, fmt.Errorf("game already started")
	}

	// Check conditions inline to avoid deadlock (can't call CanStartGame while holding lock)
	if len(lm.players) < 2 {
		return "", nil, fmt.Errorf("need at least 2 players")
	}

	hasGameMaster := false
	hasHero := false
	allReady := true

	for _, player := range lm.players {
		if player.Role == RoleGameMaster {
			hasGameMaster = true
		} else if player.Role == RoleHero {
			hasHero = true
		}

		if !player.IsReady {
			allReady = false
		}
	}

	if !hasGameMaster {
		return "", nil, fmt.Errorf("no game master")
	}

	if !hasHero {
		return "", nil, fmt.Errorf("no hero players")
	}

	if !allReady {
		return "", nil, fmt.Errorf("not all players are ready")
	}

	// Build player configurations
	heroPlayers = make(map[string]string)

	for playerID, player := range lm.players {
		if player.Role == RoleGameMaster {
			gameMasterID = playerID
		} else if player.Role == RoleHero {
			heroPlayers[playerID] = player.HeroClassID
		}
	}

	lm.gameStarted = true
	return gameMasterID, heroPlayers, nil
}

// GetPlayer returns a player's lobby info
func (lm *LobbyManager) GetPlayer(playerID string) (*PlayerLobbyInfo, bool) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	player, exists := lm.players[playerID]
	if !exists {
		return nil, false
	}

	// Return a copy
	playerCopy := *player
	return &playerCopy, true
}
