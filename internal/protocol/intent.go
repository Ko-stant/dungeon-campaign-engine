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

type RequestSelectStartingPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type RequestElectSelfAsNextPlayer struct {
}

type RequestCancelPlayerElection struct {
}

type RequestConfirmElectionAndStartTurn struct {
}

type RequestCompleteHeroTurn struct {
}

type RequestCompleteGMTurn struct {
}

type RequestSelectMonster struct {
	MonsterID string `json:"monsterId"`
}

type RequestMoveMonster struct {
	MonsterID string `json:"monsterId"`
	ToX       int    `json:"toX"`
	ToY       int    `json:"toY"`
}

type RequestMonsterAttack struct {
	MonsterID string `json:"monsterId"`
	TargetID  string `json:"targetId"`
}

type RequestUseMonsterAbility struct {
	MonsterID string `json:"monsterId"`
	AbilityID string `json:"abilityId"`
	TargetID  string `json:"targetId,omitempty"`
	TargetX   *int   `json:"targetX,omitempty"`
	TargetY   *int   `json:"targetY,omitempty"`
}
