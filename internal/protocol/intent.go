package protocol

import "encoding/json"

type IntentEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RequestToggleDoor struct {
	ThresholdID string `json:"thresholdId"`
}
