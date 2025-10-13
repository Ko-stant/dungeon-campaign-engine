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

type RequestJoinLobby struct {
	PlayerName string `json:"playerName"`
}

type RequestSelectRole struct {
	Role        string `json:"role"`
	HeroClassID string `json:"heroClassId"`
}

type RequestToggleReady struct {
	IsReady bool `json:"isReady"`
}

type RequestStartGame struct {
}
