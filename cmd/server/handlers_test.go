package main

import (
	"encoding/json"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// Mock implementations for testing handlers
type MockBroadcaster struct {
	events      []BroadcastEvent
	LastEvent   string // For compatibility with hero_actions_test.go
	LastPayload any    // For compatibility with hero_actions_test.go
}

type BroadcastEvent struct {
	EventType string
	Payload   any
}

func (m *MockBroadcaster) BroadcastEvent(eventType string, payload any) {
	m.events = append(m.events, BroadcastEvent{
		EventType: eventType,
		Payload:   payload,
	})
	// Also track last event for compatibility
	m.LastEvent = eventType
	m.LastPayload = payload
}

func (m *MockBroadcaster) GetEvents() []BroadcastEvent {
	return m.events
}

func (m *MockBroadcaster) Reset() {
	m.events = nil
}

type MockGameEngine struct {
	moveResult       *MoveResult
	moveError        error
	doorToggleResult *DoorToggleResult
	doorToggleError  error
	state            *GameState
}

func (m *MockGameEngine) ProcessMove(req protocol.RequestMove) (*MoveResult, error) {
	return m.moveResult, m.moveError
}

func (m *MockGameEngine) ProcessDoorToggle(req protocol.RequestToggleDoor) (*DoorToggleResult, error) {
	return m.doorToggleResult, m.doorToggleError
}

func (m *MockGameEngine) GetState() *GameState {
	return m.state
}

func TestTestableHandlers_HandleRequestMove_Success(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		moveResult: &MoveResult{
			EntityUpdated: &protocol.EntityUpdated{
				ID:   "hero-1",
				Tile: protocol.TileAddress{X: 6, Y: 5},
			},
			VisibleRegions:    []int{1, 2, 3},
			NewlyKnownRegions: []int{4},
			NewlyVisibleDoors: []protocol.ThresholdLite{
				{ID: "door1", X: 7, Y: 5, Orientation: "vertical"},
			},
			NewlyVisibleWalls: []protocol.BlockingWallLite{
				{ID: "wall1", X: 8, Y: 5, Orientation: "horizontal"},
			},
		},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	req := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	// Act
	err := handlers.HandleRequestMove(req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	events := broadcaster.GetEvents()
	if len(events) != 5 {
		t.Fatalf("Expected 5 broadcast events, got: %d", len(events))
	}

	// Verify event types
	expectedEventTypes := []string{
		"EntityUpdated",
		"VisibleNow",
		"RegionsKnown",
		"DoorsVisible",
		"BlockingWallsVisible",
	}

	for i, expectedType := range expectedEventTypes {
		if events[i].EventType != expectedType {
			t.Errorf("Expected event %d to be %s, got: %s", i, expectedType, events[i].EventType)
		}
	}
}

func TestTestableHandlers_HandleRequestMove_EngineError(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		moveError: &MovementError{Reason: "blocked by wall"},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	req := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	// Act
	err := handlers.HandleRequestMove(req)

	// Assert
	if err == nil {
		t.Fatal("Expected error from engine")
	}

	events := broadcaster.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected no broadcast events on error, got: %d", len(events))
	}

	// Verify logger was called
	if len(logger.messages) == 0 {
		t.Error("Expected logger to be called on error")
	}
}

func TestTestableHandlers_HandleRequestToggleDoor_Success(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		doorToggleResult: &DoorToggleResult{
			StateChange: &protocol.DoorStateChanged{
				ThresholdID: "door1",
				State:       "open",
			},
			RegionsToReveal:   []int{5},
			VisibleRegions:    []int{1, 2, 3, 5},
			NewlyKnownRegions: []int{6},
		},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	req := protocol.RequestToggleDoor{
		ThresholdID: "door1",
	}

	// Act
	err := handlers.HandleRequestToggleDoor(req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	events := broadcaster.GetEvents()
	if len(events) != 4 {
		t.Fatalf("Expected 4 broadcast events, got: %d", len(events))
	}

	// Verify specific event content
	if events[0].EventType != "DoorStateChanged" {
		t.Errorf("Expected first event to be DoorStateChanged, got: %s", events[0].EventType)
	}

	if events[1].EventType != "RegionsRevealed" {
		t.Errorf("Expected second event to be RegionsRevealed, got: %s", events[1].EventType)
	}
}

func TestTestableHandlers_HandleWebSocketMessage_Move(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		moveResult: &MoveResult{
			EntityUpdated: &protocol.EntityUpdated{
				ID:   "hero-1",
				Tile: protocol.TileAddress{X: 6, Y: 5},
			},
		},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	// Create WebSocket message
	req := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	reqData, _ := json.Marshal(req)
	envelope := protocol.IntentEnvelope{
		Type:    "RequestMove",
		Payload: reqData,
	}

	data, _ := json.Marshal(envelope)

	// Act
	err := handlers.HandleWebSocketMessage(data)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	events := broadcaster.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 broadcast event, got: %d", len(events))
	}

	if events[0].EventType != "EntityUpdated" {
		t.Errorf("Expected EntityUpdated event, got: %s", events[0].EventType)
	}
}

func TestTestableHandlers_HandleWebSocketMessage_InvalidJSON(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	invalidData := []byte("{invalid json")

	// Act
	err := handlers.HandleWebSocketMessage(invalidData)

	// Assert
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	events := broadcaster.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected no broadcast events on JSON error, got: %d", len(events))
	}
}

func TestTestableHandlers_HandleWebSocketMessage_UnknownType(t *testing.T) {
	// Arrange
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	envelope := protocol.IntentEnvelope{
		Type:    "UnknownType",
		Payload: json.RawMessage("{}"),
	}

	data, _ := json.Marshal(envelope)

	// Act
	err := handlers.HandleWebSocketMessage(data)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error for unknown type, got: %v", err)
	}

	events := broadcaster.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected no broadcast events for unknown type, got: %d", len(events))
	}

	// Verify logger was called for unknown type
	if len(logger.messages) == 0 {
		t.Error("Expected logger to be called for unknown message type")
	}
}

// Benchmark tests for handler performance
func BenchmarkTestableHandlers_HandleRequestMove(b *testing.B) {
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		moveResult: &MoveResult{
			EntityUpdated: &protocol.EntityUpdated{
				ID:   "hero-1",
				Tile: protocol.TileAddress{X: 6, Y: 5},
			},
			VisibleRegions: []int{1, 2, 3},
		},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	req := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcaster.Reset()
		err := handlers.HandleRequestMove(req)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkTestableHandlers_HandleWebSocketMessage(b *testing.B) {
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	engine := &MockGameEngine{
		moveResult: &MoveResult{
			EntityUpdated: &protocol.EntityUpdated{
				ID:   "hero-1",
				Tile: protocol.TileAddress{X: 6, Y: 5},
			},
		},
	}

	handlers := NewTestableHandlers(engine, broadcaster, logger)

	// Pre-marshal the message for realistic benchmark
	req := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	reqData, _ := json.Marshal(req)
	envelope := protocol.IntentEnvelope{
		Type:    "RequestMove",
		Payload: reqData,
	}

	data, _ := json.Marshal(envelope)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcaster.Reset()
		err := handlers.HandleWebSocketMessage(data)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
