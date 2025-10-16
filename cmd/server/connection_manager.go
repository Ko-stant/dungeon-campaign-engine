package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/coder/websocket"
)

// ConnectionInfo tracks a WebSocket connection and its associated player
type ConnectionInfo struct {
	Conn     *websocket.Conn
	PlayerID string
	InLobby  bool
}

// ConnectionManager manages WebSocket connections and player associations
type ConnectionManager struct {
	connections map[*websocket.Conn]*ConnectionInfo
	playerConns map[string]*websocket.Conn // playerID -> conn
	mutex       sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[*websocket.Conn]*ConnectionInfo),
		playerConns: make(map[string]*websocket.Conn),
	}
}

// AddConnection registers a new WebSocket connection with a generated player ID
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn) string {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	playerID := generatePlayerID()

	info := &ConnectionInfo{
		Conn:     conn,
		PlayerID: playerID,
		InLobby:  true,
	}

	cm.connections[conn] = info
	cm.playerConns[playerID] = conn

	return playerID
}

// AddConnectionWithID registers a new WebSocket connection with a specific player ID
func (cm *ConnectionManager) AddConnectionWithID(conn *websocket.Conn, playerID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Remove old connection for this player ID if it exists
	if oldConn, exists := cm.playerConns[playerID]; exists {
		delete(cm.connections, oldConn)
	}

	info := &ConnectionInfo{
		Conn:     conn,
		PlayerID: playerID,
		InLobby:  true,
	}

	cm.connections[conn] = info
	cm.playerConns[playerID] = conn
}

// RemoveConnection removes a connection and returns the player ID
func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) string {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	info, exists := cm.connections[conn]
	if !exists {
		return ""
	}

	playerID := info.PlayerID
	delete(cm.connections, conn)
	delete(cm.playerConns, playerID)

	return playerID
}

// GetPlayerID returns the player ID for a connection
func (cm *ConnectionManager) GetPlayerID(conn *websocket.Conn) (string, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	info, exists := cm.connections[conn]
	if !exists {
		return "", false
	}
	return info.PlayerID, true
}

// GetConnection returns the connection for a player ID
func (cm *ConnectionManager) GetConnection(playerID string) (*websocket.Conn, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conn, exists := cm.playerConns[playerID]
	return conn, exists
}

// SetInLobby marks whether a player is in the lobby or in-game
func (cm *ConnectionManager) SetInLobby(conn *websocket.Conn, inLobby bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if info, exists := cm.connections[conn]; exists {
		info.InLobby = inLobby
	}
}

// IsInLobby checks if a connection is in the lobby
func (cm *ConnectionManager) IsInLobby(conn *websocket.Conn) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	info, exists := cm.connections[conn]
	if !exists {
		return false
	}
	return info.InLobby
}

// GetAllConnections returns all active connections
func (cm *ConnectionManager) GetAllConnections() []*websocket.Conn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conns := make([]*websocket.Conn, 0, len(cm.connections))
	for conn := range cm.connections {
		conns = append(conns, conn)
	}
	return conns
}

// GetLobbyConnections returns all connections that are in the lobby
func (cm *ConnectionManager) GetLobbyConnections() []*websocket.Conn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conns := make([]*websocket.Conn, 0)
	for conn, info := range cm.connections {
		if info.InLobby {
			conns = append(conns, conn)
		}
	}
	return conns
}

// generatePlayerID creates a unique player ID
func generatePlayerID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple counter-based ID if random fails
		return fmt.Sprintf("player-%d", len(bytes))
	}
	return "player-" + hex.EncodeToString(bytes)
}
