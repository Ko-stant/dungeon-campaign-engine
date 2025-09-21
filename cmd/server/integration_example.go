// integration_example.go - Example of how to use the new testable architecture

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// ExampleOfNewArchitecture demonstrates how to use the refactored, testable components
func ExampleOfNewArchitecture() {
	// 1. Load game content
	board, quest, err := loadGameContent()
	if err != nil {
		log.Fatalf("Failed to load game content: %v", err)
	}

	furnitureSystem := NewFurnitureSystem(log.New(os.Stdout, "", log.LstdFlags))

	// 2. Initialize game state
	state, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		log.Fatalf("Failed to initialize game state: %v", err)
	}

	// 3. Set up profiling (optional)
	profilingConfig := GetProfilingConfigFromEnv()
	StartProfiling(profilingConfig)

	// 4. Create performance metrics tracker
	metrics := NewPerformanceMetrics()
	StartMetricsReporting(metrics, 30*time.Second) // Log metrics every 30 seconds

	// 5. Create dependencies with dependency injection
	logger := NewLogger()
	sequenceGenerator := NewSequenceGenerator()
	hub := ws.NewHub()
	broadcaster := NewBroadcaster(hub, sequenceGenerator)

	// 6. Create visibility calculator and movement validator
	visibilityCalculator := NewVisibilityCalculator(logger)
	movementValidator := NewMovementValidator(logger)

	// 7. Create game engine with instrumentation
	baseEngine := NewGameEngine(state, visibilityCalculator, movementValidator, logger, quest)
	instrumentedEngine := NewInstrumentedGameEngine(baseEngine, metrics)

	// 8. Create testable handlers
	handlers := NewTestableHandlers(instrumentedEngine, broadcaster, logger)

	// 9. Set up HTTP routes
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// 10. WebSocket endpoint with new handlers
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)

		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer c.Close(websocket.StatusNormalClosure, "")
			for {
				_, data, err := c.Read(context.Background())
				if err != nil {
					return
				}
				// Use new testable handler
				if err := handlers.HandleWebSocketMessage(data); err != nil {
					logger.Printf("Error handling WebSocket message: %v", err)
				}
			}
		}(conn)
	})

	// 11. Main page handler (unchanged)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hero := state.Entities["hero-1"]
		visibleNow := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
		var revealed []int
		for id := range state.RevealedRegions {
			revealed = append(revealed, id)
		}
		entities := []protocol.EntityLite{
			{ID: "hero-1", Kind: "hero", Tile: state.Entities["hero-1"]},
		}

		// Include all known doors in initial snapshot
		thresholds := make([]protocol.ThresholdLite, 0, len(state.KnownDoors))
		for id := range state.KnownDoors {
			if info, exists := state.Doors[id]; exists {
				thresholds = append(thresholds, protocol.ThresholdLite{
					ID:          id,
					X:           info.Edge.X,
					Y:           info.Edge.Y,
					Orientation: string(info.Edge.Orientation),
					Kind:        "DoorSocket",
					State:       info.State,
				})
			}
		}

		// Include visible blocking walls
		blockingWalls, _ := getVisibleBlockingWalls(state, hero, quest)

		known := make([]int, 0, len(state.KnownRegions))
		for rid := range state.KnownRegions {
			known = append(known, rid)
		}

		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              1,
			LastEventID:       0,
			MapWidth:          state.Segment.Width,
			MapHeight:         state.Segment.Height,
			RegionsCount:      state.RegionMap.RegionsCount,
			TileRegionIDs:     state.RegionMap.TileRegionIDs,
			RevealedRegionIDs: revealed,
			DoorStates:        []byte{},
			Entities:          entities,
			Variables:         map[string]any{"ui.debug": true},
			ProtocolVersion:   "v0",
			Thresholds:        thresholds,
			BlockingWalls:     blockingWalls,
			VisibleRegionIDs:  visibleNow,
			CorridorRegionID:  state.CorridorRegion,
			KnownRegionIDs:    known,
		}
		if err := views.IndexPage(s).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// 12. Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)

	// Log that new architecture is being used
	logger.Printf("Using new testable architecture with dependency injection")
	logger.Printf("Profiling enabled: %v", profilingConfig.Enabled)
	if profilingConfig.Enabled {
		logger.Printf("pprof server available at http://localhost:%s/debug/pprof/", profilingConfig.Port)
	}

	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// ExampleTestUsage shows how to use mocks in tests
// This is documented here but the actual mocks are defined in *_test.go files
func ExampleTestUsage() {
	log.Printf("See *_test.go files for actual mock usage examples")
	log.Printf("Mocks available: MockLogger, MockBroadcaster, MockVisibilityCalculator, MockMovementValidator")
	log.Printf("Test helpers: createTestGameState(), createSimpleTestState()")
}

// Recommendation: Replace the current main() function with this new architecture
// The old main() function can be renamed to mainOld() for comparison
func mainWithNewArchitecture() {
	ExampleOfNewArchitecture()
}
