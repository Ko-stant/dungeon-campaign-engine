package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// ProfilingConfig holds configuration for profiling
type ProfilingConfig struct {
	Enabled         bool
	Port            string
	CPUProfilePath  string
	MemProfilePath  string
	ProfileDuration time.Duration
}

// StartProfiling starts the profiling server and sets up profiling
func StartProfiling(config ProfilingConfig) {
	if !config.Enabled {
		return
	}

	// Set up runtime profiling parameters
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	// Start pprof server on separate port
	if config.Port != "" {
		go func() {
			log.Printf("Starting pprof server on :%s", config.Port)
			log.Printf("CPU profile: http://localhost:%s/debug/pprof/profile", config.Port)
			log.Printf("Heap profile: http://localhost:%s/debug/pprof/heap", config.Port)
			log.Printf("Goroutine profile: http://localhost:%s/debug/pprof/goroutine", config.Port)
			log.Printf("Block profile: http://localhost:%s/debug/pprof/block", config.Port)
			log.Printf("Mutex profile: http://localhost:%s/debug/pprof/mutex", config.Port)

			if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
				log.Printf("pprof server failed: %v", err)
			}
		}()
	}

	log.Printf("Profiling enabled. Access profiles at:")
	log.Printf("  - CPU: curl http://localhost:%s/debug/pprof/profile?seconds=30 > cpu.prof", config.Port)
	log.Printf("  - Memory: curl http://localhost:%s/debug/pprof/heap > mem.prof", config.Port)
	log.Printf("  - Goroutines: curl http://localhost:%s/debug/pprof/goroutine > goroutine.prof", config.Port)
}

// GetProfilingConfigFromEnv creates profiling config from environment variables
func GetProfilingConfigFromEnv() ProfilingConfig {
	enabled := os.Getenv("ENABLE_PROFILING") == "true"
	port := os.Getenv("PPROF_PORT")
	if port == "" {
		port = "42069"
	}

	return ProfilingConfig{
		Enabled:         enabled,
		Port:            port,
		CPUProfilePath:  os.Getenv("CPU_PROFILE_PATH"),
		MemProfilePath:  os.Getenv("MEM_PROFILE_PATH"),
		ProfileDuration: 30 * time.Second,
	}
}

// PerformanceMetrics holds performance tracking data
type PerformanceMetrics struct {
	MovesProcessed    int64
	DoorsToggled      int64
	VisibilityCalcs   int64
	AvgMoveTime       time.Duration
	AvgDoorToggleTime time.Duration
	AvgVisibilityTime time.Duration
	PeakGoroutines    int
	PeakMemoryUsage   uint64
	StartTime         time.Time
}

// NewPerformanceMetrics creates a new performance metrics tracker
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		StartTime: time.Now(),
	}
}

// TrackMove records metrics for a move operation
func (pm *PerformanceMetrics) TrackMove(duration time.Duration) {
	pm.MovesProcessed++
	// Simple running average (for demo purposes)
	pm.AvgMoveTime = (pm.AvgMoveTime*time.Duration(pm.MovesProcessed-1) + duration) / time.Duration(pm.MovesProcessed)
}

// TrackDoorToggle records metrics for a door toggle operation
func (pm *PerformanceMetrics) TrackDoorToggle(duration time.Duration) {
	pm.DoorsToggled++
	pm.AvgDoorToggleTime = (pm.AvgDoorToggleTime*time.Duration(pm.DoorsToggled-1) + duration) / time.Duration(pm.DoorsToggled)
}

// TrackVisibility records metrics for visibility calculations
func (pm *PerformanceMetrics) TrackVisibility(duration time.Duration) {
	pm.VisibilityCalcs++
	pm.AvgVisibilityTime = (pm.AvgVisibilityTime*time.Duration(pm.VisibilityCalcs-1) + duration) / time.Duration(pm.VisibilityCalcs)
}

// UpdateSystemMetrics updates system-level metrics
func (pm *PerformanceMetrics) UpdateSystemMetrics() {
	// Track goroutines
	goroutines := runtime.NumGoroutine()
	if goroutines > pm.PeakGoroutines {
		pm.PeakGoroutines = goroutines
	}

	// Track memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Alloc > pm.PeakMemoryUsage {
		pm.PeakMemoryUsage = m.Alloc
	}
}

// LogMetrics logs current performance metrics
func (pm *PerformanceMetrics) LogMetrics() {
	uptime := time.Since(pm.StartTime)
	log.Printf("=== Performance Metrics ===")
	log.Printf("Uptime: %v", uptime)
	log.Printf("Moves processed: %d", pm.MovesProcessed)
	log.Printf("Doors toggled: %d", pm.DoorsToggled)
	log.Printf("Visibility calculations: %d", pm.VisibilityCalcs)
	log.Printf("Average move time: %v", pm.AvgMoveTime)
	log.Printf("Average door toggle time: %v", pm.AvgDoorToggleTime)
	log.Printf("Average visibility time: %v", pm.AvgVisibilityTime)
	log.Printf("Peak goroutines: %d", pm.PeakGoroutines)
	log.Printf("Peak memory usage: %d bytes", pm.PeakMemoryUsage)

	if pm.MovesProcessed > 0 {
		movesPerSecond := float64(pm.MovesProcessed) / uptime.Seconds()
		log.Printf("Moves per second: %.2f", movesPerSecond)
	}
}

// InstrumentedGameEngine wraps GameEngine with performance tracking
type InstrumentedGameEngine struct {
	engine  GameEngine
	metrics *PerformanceMetrics
}

func NewInstrumentedGameEngine(engine GameEngine, metrics *PerformanceMetrics) *InstrumentedGameEngine {
	return &InstrumentedGameEngine{
		engine:  engine,
		metrics: metrics,
	}
}

func (ie *InstrumentedGameEngine) ProcessMove(req protocol.RequestMove) (*MoveResult, error) {
	start := time.Now()
	result, err := ie.engine.ProcessMove(req)
	duration := time.Since(start)

	ie.metrics.TrackMove(duration)
	ie.metrics.UpdateSystemMetrics()

	return result, err
}

func (ie *InstrumentedGameEngine) ProcessDoorToggle(req protocol.RequestToggleDoor) (*DoorToggleResult, error) {
	start := time.Now()
	result, err := ie.engine.ProcessDoorToggle(req)
	duration := time.Since(start)

	ie.metrics.TrackDoorToggle(duration)
	ie.metrics.UpdateSystemMetrics()

	return result, err
}

func (ie *InstrumentedGameEngine) GetState() *GameState {
	return ie.engine.GetState()
}

// StartMetricsReporting starts periodic metrics reporting
func StartMetricsReporting(metrics *PerformanceMetrics, interval time.Duration) {
	if interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			metrics.LogMetrics()
		}
	}()
}
