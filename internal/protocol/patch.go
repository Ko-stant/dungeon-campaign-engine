package protocol

type PatchEnvelope struct {
	Sequence uint64 `json:"seq"`
	EventID  int64  `json:"eventId"`
	Type     string `json:"type"`
	Payload  any    `json:"payload"`
}

type VariablesChanged struct {
	Entries map[string]any `json:"entries"`
}

type DoorStateChanged struct {
	ThresholdID string `json:"thresholdId"`
	State       string `json:"state"`
}

type RegionsRevealed struct {
	IDs []int `json:"ids"`
}

type EntityUpdated struct {
	ID   string      `json:"id"`
	Tile TileAddress `json:"tile"`
}

type VisibleNow struct {
	IDs []int `json:"ids"`
}

type RegionsKnown struct {
	IDs []int `json:"ids"`
}

type DoorsVisible struct {
	Doors []ThresholdLite `json:"doors"`
}

type BlockingWallsVisible struct {
	BlockingWalls []BlockingWallLite `json:"blockingWalls"`
}

type FurnitureVisible struct {
	Furniture []FurnitureLite `json:"furniture"`
}

type TurnStateChanged struct {
	TurnNumber     int    `json:"turnNumber"`
	CurrentTurn    string `json:"currentTurn"`
	CurrentPhase   string `json:"currentPhase"`
	ActivePlayerID string `json:"activePlayerId"`
	ActionsLeft    int    `json:"actionsLeft"`
	MovementLeft   int    `json:"movementLeft"`
	HasMoved       bool   `json:"hasMoved"`
	ActionTaken    bool   `json:"actionTaken"`
	CanEndTurn     bool   `json:"canEndTurn"`
}
