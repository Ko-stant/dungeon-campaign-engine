package protocol

import "encoding/json"

type IntentEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RequestToggleDoor struct {
	ThresholdID string `json:"thresholdId"`
}

type RequestMove struct {
	EntityID string `json:"entityId"`
	DX       int    `json:"dx"`
	DY       int    `json:"dy"`
}

type RequestToggleAdjacentDoor struct {
	EntityID string `json:"entityId"`
}
