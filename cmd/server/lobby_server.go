package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// LobbyServer manages the pre-game lobby and coordinates player connections
type LobbyServer struct {
	lobby            *LobbyManager
	connManager      *ConnectionManager
	contentManager   *ContentManager
	sequenceGen      *SequenceGeneratorImpl
	gameStartHandler func(gameMasterID string, heroPlayers map[string]string) error
	mutex            sync.RWMutex
}

// NewLobbyServer creates a new lobby server
func NewLobbyServer(contentManager *ContentManager, sequenceGen *SequenceGeneratorImpl) *LobbyServer {
	return &LobbyServer{
		lobby:          NewLobbyManager(contentManager),
		connManager:    NewConnectionManager(),
		contentManager: contentManager,
		sequenceGen:    sequenceGen,
	}
}

// HandleNewConnection handles a new WebSocket connection in lobby mode
func (ls *LobbyServer) HandleNewConnection(conn *websocket.Conn) string {
	playerID := ls.connManager.AddConnection(conn)
	log.Printf("New connection: %s", playerID)

	// Send initial lobby state
	ls.broadcastLobbyState()

	return playerID
}

// HandleNewConnectionWithID handles a new WebSocket connection with a specific player ID
func (ls *LobbyServer) HandleNewConnectionWithID(conn *websocket.Conn, playerID string) {
	ls.connManager.AddConnectionWithID(conn, playerID)
	log.Printf("New connection: %s", playerID)

	// Send initial lobby state
	ls.broadcastLobbyState()
}

// HandleDisconnection handles a player disconnection
func (ls *LobbyServer) HandleDisconnection(conn *websocket.Conn) {
	playerID := ls.connManager.RemoveConnection(conn)
	if playerID != "" {
		log.Printf("Player disconnected: %s", playerID)
		ls.lobby.RemovePlayer(playerID)
		ls.broadcastLobbyState()
	}
}

// HandleMessage processes a WebSocket message from a player in the lobby
func (ls *LobbyServer) HandleMessage(conn *websocket.Conn, data []byte) error {
	var env protocol.IntentEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	playerID, ok := ls.connManager.GetPlayerID(conn)
	if !ok {
		return fmt.Errorf("unknown connection")
	}

	switch env.Type {
	case "RequestJoinLobby":
		return ls.handleJoinLobby(playerID, env.Payload)

	case "RequestSelectRole":
		return ls.handleSelectRole(playerID, env.Payload)

	case "RequestToggleReady":
		return ls.handleToggleReady(playerID, env.Payload)

	case "RequestStartGame":
		return ls.handleStartGame(playerID)

	default:
		return fmt.Errorf("unknown lobby message type: %s", env.Type)
	}
}

// handleJoinLobby processes a player join request
func (ls *LobbyServer) handleJoinLobby(playerID string, payload json.RawMessage) error {
	var req protocol.RequestJoinLobby
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	log.Printf("Player %s joining lobby as '%s'", playerID, req.PlayerName)

	if err := ls.lobby.AddPlayer(playerID, req.PlayerName); err != nil {
		log.Printf("Error adding player to lobby: %v", err)
		return err
	}

	log.Printf("Player added successfully, broadcasting lobby state")
	ls.broadcastLobbyState()
	return nil
}

// handleSelectRole processes a role selection request
func (ls *LobbyServer) handleSelectRole(playerID string, payload json.RawMessage) error {
	var req protocol.RequestSelectRole
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	log.Printf("Player %s selecting role: %s (hero: %s)", playerID, req.Role, req.HeroClassID)

	role := PlayerRole(req.Role)
	if err := ls.lobby.SetPlayerRole(playerID, role, req.HeroClassID); err != nil {
		log.Printf("Error setting player role: %v", err)
		return err
	}

	log.Printf("Role set successfully, broadcasting lobby state")
	ls.broadcastLobbyState()
	return nil
}

// handleToggleReady processes a ready status toggle request
func (ls *LobbyServer) handleToggleReady(playerID string, payload json.RawMessage) error {
	var req protocol.RequestToggleReady
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	log.Printf("Player %s toggling ready: %v", playerID, req.IsReady)

	if err := ls.lobby.SetPlayerReady(playerID, req.IsReady); err != nil {
		return err
	}

	ls.broadcastLobbyState()
	return nil
}

// handleStartGame processes a game start request
func (ls *LobbyServer) handleStartGame(playerID string) error {
	log.Printf("Player %s requesting game start", playerID)

	// Verify player is game master
	player, ok := ls.lobby.GetPlayer(playerID)
	if !ok || player.Role != RoleGameMaster {
		log.Printf("Error: player %s is not game master (ok=%v, role=%s)", playerID, ok, player.Role)
		return fmt.Errorf("only game master can start the game")
	}

	log.Printf("Player verified as GM, calling StartGame()")

	// Start the game
	gameMasterID, heroPlayers, err := ls.lobby.StartGame()
	if err != nil {
		log.Printf("Error starting game: %v", err)
		return err
	}

	log.Printf("Game starting: GM=%s, Heroes=%v", gameMasterID, heroPlayers)

	// Broadcast game starting message
	ls.broadcastGameStarting()

	// Call game start handler if set
	if ls.gameStartHandler != nil {
		return ls.gameStartHandler(gameMasterID, heroPlayers)
	}

	return nil
}

// broadcastLobbyState sends the current lobby state to all connected clients
func (ls *LobbyServer) broadcastLobbyState() {
	lobbyState := ls.lobby.GetLobbyState()

	// Convert to protocol format
	players := make(map[string]*protocol.PlayerLobbyInfo)
	for id, p := range lobbyState.Players {
		players[id] = &protocol.PlayerLobbyInfo{
			ID:          p.ID,
			Name:        p.Name,
			Role:        string(p.Role),
			HeroClassID: p.HeroClassID,
			IsReady:     p.IsReady,
		}
	}

	payload := protocol.LobbyStateChanged{
		Players:         players,
		CanStartGame:    lobbyState.CanStartGame,
		GameStarted:     lobbyState.GameStarted,
		AvailableHeroes: lobbyState.AvailableHeroes,
	}

	log.Printf("Broadcasting lobby state: %d players, canStart=%v", len(players), lobbyState.CanStartGame)
	for id, p := range players {
		log.Printf("  Player %s: name=%s, role=%s, heroClass=%s, ready=%v", id, p.Name, p.Role, p.HeroClassID, p.IsReady)
	}

	patch := protocol.PatchEnvelope{
		Sequence: ls.sequenceGen.Next(),
		EventID:  int64(ls.sequenceGen.Next()),
		Type:     "LobbyStateChanged",
		Payload:  payload,
	}

	data, err := json.Marshal(patch)
	if err != nil {
		log.Printf("Failed to marshal lobby state: %v", err)
		return
	}

	// Broadcast to all lobby connections
	lobbyConns := ls.connManager.GetLobbyConnections()
	log.Printf("Sending to %d lobby connections", len(lobbyConns))
	for _, conn := range lobbyConns {
		if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
			log.Printf("Failed to send lobby state to connection: %v", err)
		}
	}
}

// broadcastGameStarting sends a game starting message to all clients
func (ls *LobbyServer) broadcastGameStarting() {
	payload := protocol.GameStarting{
		Message: "Game is starting! Loading quest...",
	}

	patch := protocol.PatchEnvelope{
		Sequence: ls.sequenceGen.Next(),
		EventID:  int64(ls.sequenceGen.Next()),
		Type:     "GameStarting",
		Payload:  payload,
	}

	data, err := json.Marshal(patch)
	if err != nil {
		log.Printf("Failed to marshal game starting message: %v", err)
		return
	}

	// Broadcast to all lobby connections
	for _, conn := range ls.connManager.GetLobbyConnections() {
		if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
			log.Printf("Failed to send game starting message: %v", err)
		}
	}
}

// SetGameStartHandler sets the callback function to be called when the game starts
func (ls *LobbyServer) SetGameStartHandler(handler func(gameMasterID string, heroPlayers map[string]string) error) {
	ls.gameStartHandler = handler
}

// GetConnectionManager returns the connection manager for integration with game phase
func (ls *LobbyServer) GetConnectionManager() *ConnectionManager {
	return ls.connManager
}
