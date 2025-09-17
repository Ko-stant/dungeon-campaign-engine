package main

import (
	"encoding/json"
	"log"
	"sync/atomic"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// BroadcasterImpl implements Broadcaster using WebSocket hub
type BroadcasterImpl struct {
	hub      *ws.Hub
	sequence SequenceGenerator
}

func NewBroadcaster(hub *ws.Hub, sequence SequenceGenerator) *BroadcasterImpl {
	return &BroadcasterImpl{
		hub:      hub,
		sequence: sequence,
	}
}

func (b *BroadcasterImpl) BroadcastEvent(eventType string, payload interface{}) {
	seq := b.sequence.Next()
	envelope := protocol.PatchEnvelope{
		Sequence: seq,
		EventID:  0,
		Type:     eventType,
		Payload:  payload,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("failed to marshal %s: %v", eventType, err)
		return
	}
	log.Printf("broadcasting %s", eventType)
	b.hub.Broadcast(data)
}

// LoggerImpl implements Logger using standard log package
type LoggerImpl struct{}

func NewLogger() *LoggerImpl {
	return &LoggerImpl{}
}

func (l *LoggerImpl) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// SequenceGeneratorImpl implements SequenceGenerator using atomic counter
type SequenceGeneratorImpl struct {
	counter uint64
}

func NewSequenceGenerator() *SequenceGeneratorImpl {
	return &SequenceGeneratorImpl{}
}

func (sg *SequenceGeneratorImpl) Next() uint64 {
	return atomic.AddUint64(&sg.counter, 1)
}

func (sg *SequenceGeneratorImpl) Current() uint64 {
	return atomic.LoadUint64(&sg.counter)
}

// TestableHandlers uses dependency injection for better testability
type TestableHandlers struct {
	engine      GameEngine
	broadcaster Broadcaster
	logger      Logger
}

func NewTestableHandlers(engine GameEngine, broadcaster Broadcaster, logger Logger) *TestableHandlers {
	return &TestableHandlers{
		engine:      engine,
		broadcaster: broadcaster,
		logger:      logger,
	}
}

func (h *TestableHandlers) HandleRequestMove(req protocol.RequestMove) error {
	result, err := h.engine.ProcessMove(req)
	if err != nil {
		h.logger.Printf("Move failed: %v", err)
		return err
	}

	// Broadcast all the events
	if result.EntityUpdated != nil {
		h.broadcaster.BroadcastEvent("EntityUpdated", *result.EntityUpdated)
	}

	if len(result.VisibleRegions) > 0 {
		h.broadcaster.BroadcastEvent("VisibleNow", protocol.VisibleNow{IDs: result.VisibleRegions})
	}

	if len(result.NewlyKnownRegions) > 0 {
		h.broadcaster.BroadcastEvent("RegionsKnown", protocol.RegionsKnown{IDs: result.NewlyKnownRegions})
	}

	if len(result.NewlyVisibleDoors) > 0 {
		h.broadcaster.BroadcastEvent("DoorsVisible", protocol.DoorsVisible{Doors: result.NewlyVisibleDoors})
	}

	if len(result.NewlyVisibleWalls) > 0 {
		h.broadcaster.BroadcastEvent("BlockingWallsVisible", protocol.BlockingWallsVisible{BlockingWalls: result.NewlyVisibleWalls})
	}

	return nil
}

func (h *TestableHandlers) HandleRequestToggleDoor(req protocol.RequestToggleDoor) error {
	result, err := h.engine.ProcessDoorToggle(req)
	if err != nil {
		h.logger.Printf("Door toggle failed: %v", err)
		return err
	}

	// Broadcast all the events
	if result.StateChange != nil {
		h.broadcaster.BroadcastEvent("DoorStateChanged", *result.StateChange)
	}

	if len(result.RegionsToReveal) > 0 {
		h.broadcaster.BroadcastEvent("RegionsRevealed", protocol.RegionsRevealed{IDs: result.RegionsToReveal})
	}

	if len(result.VisibleRegions) > 0 {
		h.broadcaster.BroadcastEvent("VisibleNow", protocol.VisibleNow{IDs: result.VisibleRegions})
	}

	if len(result.NewlyKnownRegions) > 0 {
		h.broadcaster.BroadcastEvent("RegionsKnown", protocol.RegionsKnown{IDs: result.NewlyKnownRegions})
	}

	if len(result.NewlyVisibleDoors) > 0 {
		h.broadcaster.BroadcastEvent("DoorsVisible", protocol.DoorsVisible{Doors: result.NewlyVisibleDoors})
	}

	if len(result.NewlyVisibleWalls) > 0 {
		h.broadcaster.BroadcastEvent("BlockingWallsVisible", protocol.BlockingWallsVisible{BlockingWalls: result.NewlyVisibleWalls})
	}

	return nil
}

func (h *TestableHandlers) HandleWebSocketMessage(data []byte) error {
	var env protocol.IntentEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}

	switch env.Type {
	case "RequestMove":
		var req protocol.RequestMove
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return err
		}
		return h.HandleRequestMove(req)

	case "RequestToggleDoor":
		var req protocol.RequestToggleDoor
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return err
		}
		return h.HandleRequestToggleDoor(req)

	default:
		h.logger.Printf("Unknown message type: %s", env.Type)
		return nil
	}
}