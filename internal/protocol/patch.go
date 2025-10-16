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

type MonstersVisible struct {
	Monsters []MonsterLite `json:"monsters"`
}

type TurnStateChanged struct {
	HeroID         string `json:"heroId,omitempty"`   // Added for hero turn states
	PlayerID       string `json:"playerId,omitempty"` // Added for hero turn states
	TurnNumber     int    `json:"turnNumber"`
	CurrentTurn    string `json:"currentTurn"`
	CurrentPhase   string `json:"currentPhase"`
	ActivePlayerID string `json:"activePlayerId"`
	ActionsLeft    int    `json:"actionsLeft"`
	MovementLeft   int    `json:"movementLeft"`
	HasMoved       bool   `json:"hasMoved"`
	ActionTaken    bool   `json:"actionTaken"`
	CanEndTurn     bool   `json:"canEndTurn"`
	// Movement dice fields for hero turn states
	MovementDiceRolled  bool  `json:"movementDiceRolled,omitempty"`
	MovementDiceResults []int `json:"movementDiceResults,omitempty"`
	MovementTotal       int   `json:"movementTotal,omitempty"`
	MovementUsed        int   `json:"movementUsed,omitempty"`
}

type MovementHistorySync struct {
	History            []MovementSegment `json:"history"`
	CurrentSegment     *MovementSegment  `json:"currentSegment"`
	InitialPosition    *MovementStep     `json:"initialPosition"`
	MovementLeft       int               `json:"movementLeft"`
	MovementDiceRolled bool              `json:"movementDiceRolled"`
}

type MovementStep struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MovementSegment struct {
	Type          string         `json:"type"`
	StartPosition MovementStep   `json:"startPosition"`
	Path          []MovementStep `json:"path"`
	StartTime     string         `json:"startTime"`
	EndTime       *string        `json:"endTime,omitempty"`
	Executed      bool           `json:"executed"`
	ExecutedTime  *string        `json:"executedTime,omitempty"`
}

type LobbyStateChanged struct {
	Players         map[string]*PlayerLobbyInfo `json:"players"`
	CanStartGame    bool                        `json:"canStartGame"`
	GameStarted     bool                        `json:"gameStarted"`
	AvailableHeroes []string                    `json:"availableHeroes"`
}

type PlayerLobbyInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	HeroClassID string `json:"heroClassId"`
	IsReady     bool   `json:"isReady"`
}

type GameStarting struct {
	Message string `json:"message"`
}

type PlayerIDAssigned struct {
	PlayerID string `json:"playerId"`
}

type TurnPhaseChanged struct {
	CurrentPhase       string   `json:"currentPhase"`
	CycleNumber        int      `json:"cycleNumber"`
	ActiveHeroPlayerID string   `json:"activeHeroPlayerId,omitempty"`
	ElectedPlayerID    string   `json:"electedPlayerId,omitempty"`
	HeroesActedIDs     []string `json:"heroesActedIds"`
	EligibleHeroIDs    []string `json:"eligibleHeroIds"`
}

type QuestSetupStateChanged struct {
	PlayersReady         map[string]bool              `json:"playersReady"`
	PlayerStartPositions map[string]StartPositionInfo `json:"playerStartPositions"`
	AllPlayersReady      bool                         `json:"allPlayersReady"`
}

type StartPositionInfo struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MonsterTurnStateChanged struct {
	MonsterID            string               `json:"monsterId"`
	EntityID             string               `json:"entityId"`
	TurnNumber           int                  `json:"turnNumber"`
	CurrentPosition      TileAddress          `json:"currentPosition"`
	FixedMovement        int                  `json:"fixedMovement"`
	MovementRemaining    int                  `json:"movementRemaining"`
	MovementUsed         int                  `json:"movementUsed"`
	HasMoved             bool                 `json:"hasMoved"`
	ActionTaken          bool                 `json:"actionTaken"`
	ActionType           string               `json:"actionType,omitempty"`
	AttackDice           int                  `json:"attackDice"`
	DefenseDice          int                  `json:"defenseDice"`
	BodyPoints           int                  `json:"bodyPoints"`
	CurrentBody          int                  `json:"currentBody"`
	SpecialAbilities     []MonsterAbilityLite `json:"specialAbilities"`
	AbilityUsageThisTurn map[string]int       `json:"abilityUsageThisTurn"`
	ActiveEffectsCount   int                  `json:"activeEffectsCount"`
}

type MonsterAbilityLite struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	UsesPerTurn    int                    `json:"usesPerTurn"`
	UsesPerQuest   int                    `json:"usesPerQuest"`
	UsesLeftQuest  int                    `json:"usesLeftQuest"`
	RequiresAction bool                   `json:"requiresAction"`
	Range          int                    `json:"range"`
	Description    string                 `json:"description"`
	EffectDetails  map[string]interface{} `json:"effectDetails"`
}

type MonsterSelectionChanged struct {
	SelectedMonsterID string `json:"selectedMonsterId,omitempty"`
}

type AllMonsterStatesSync struct {
	MonsterStates map[string]*MonsterTurnStateChanged `json:"monsterStates"`
}
