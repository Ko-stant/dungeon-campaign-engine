package protocol

type TileAddress struct {
	SegmentID string `json:"segmentId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
}

type HP struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

type EntityLite struct {
	ID         string      `json:"id"`
	Kind       string      `json:"kind"`
	Tile       TileAddress `json:"tile"`
	HP         *HP         `json:"hp,omitempty"`
	MindPoints *HP         `json:"mindPoints,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
}
type ThresholdLite struct {
	ID          string `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
	Kind        string `json:"kind"`
	State       string `json:"state,omitempty"`
}

type BlockingWallLite struct {
	ID          string `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
	Size        int    `json:"size"`
}

type FurnitureLite struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Tile     TileAddress `json:"tile"`
	GridSize struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rotation           int    `json:"rotation,omitempty"`           // 0, 90, 180, 270 degrees
	SwapAspectOnRotate bool   `json:"swapAspectOnRotate,omitempty"` // Whether to swap width/height for 90/270 rotations
	TileImage          string `json:"tileImage"`
	TileImageCleaned   string `json:"tileImageCleaned"`
	PixelDimensions    struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"pixelDimensions"`
	BlocksLineOfSight bool     `json:"blocksLineOfSight"`
	BlocksMovement    bool     `json:"blocksMovement"`
	Contains          []string `json:"contains,omitempty"`
}

type MonsterLite struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Tile        TileAddress `json:"tile"`
	Body        int         `json:"body"`
	MaxBody     int         `json:"MaxBody"`
	Mind        int         `json:"mind"`
	MaxMind     int         `json:"maxMind"`
	AttackDice  int         `json:"attackDice"`
	DefenseDice int         `json:"defenseDice"`
	IsVisible   bool        `json:"isVisible"`
	IsAlive     bool        `json:"isAlive"`
}

type HeroTurnStateLite struct {
	HeroID              string                           `json:"heroId"`
	PlayerID            string                           `json:"playerId"`
	TurnNumber          int                              `json:"turnNumber"`
	MovementDiceRolled  bool                             `json:"movementDiceRolled"`
	MovementDiceResults []int                            `json:"movementDiceResults"`
	MovementTotal       int                              `json:"movementTotal"`
	MovementUsed        int                              `json:"movementUsed"`
	MovementRemaining   int                              `json:"movementRemaining"`
	HasMoved            bool                             `json:"hasMoved"`
	ActionTaken         bool                             `json:"actionTaken"`
	ActionType          string                           `json:"actionType,omitempty"`
	TurnFlags           map[string]bool                  `json:"turnFlags"`
	ActivitiesCount     int                              `json:"activitiesCount"`
	ActiveEffectsCount  int                              `json:"activeEffectsCount"`
	ActiveEffects       []ActiveEffectLite               `json:"activeEffects"`
	LocationSearches    map[string]LocationSearchSummary `json:"locationSearches"`
	TurnStartPosition   TileAddress                      `json:"turnStartPosition"`
	CurrentPosition     TileAddress                      `json:"currentPosition"`
}

type ActiveEffectLite struct {
	Source     string `json:"source"`
	EffectType string `json:"effectType"`
	Value      int    `json:"value"`
	Trigger    string `json:"trigger"`
	Applied    bool   `json:"applied"`
}

type LocationSearchSummary struct {
	LocationKey        string `json:"locationKey"`
	TreasureSearchDone bool   `json:"treasureSearchDone"`
}

type Snapshot struct {
	MapID             string                       `json:"mapId"`
	PackID            string                       `json:"packId"`
	Turn              int                          `json:"turn"`
	LastEventID       int64                        `json:"lastEventId"`
	MapWidth          int                          `json:"mapWidth"`
	MapHeight         int                          `json:"mapHeight"`
	RegionsCount      int                          `json:"regionsCount"`
	TileRegionIDs     []int                        `json:"tileRegionIds"`
	RevealedRegionIDs []int                        `json:"revealedRegionIds"`
	DoorStates        []byte                       `json:"doorStates"`
	Entities          []EntityLite                 `json:"entities"`
	Thresholds        []ThresholdLite              `json:"thresholds"`
	BlockingWalls     []BlockingWallLite           `json:"blockingWalls"`
	Furniture         []FurnitureLite              `json:"furniture"`
	Monsters          []MonsterLite                `json:"monsters"`
	Variables         map[string]any               `json:"variables"`
	HeroTurnStates    map[string]HeroTurnStateLite `json:"heroTurnStates"`
	ProtocolVersion   string                       `json:"protocolVersion"`
	VisibleRegionIDs  []int                        `json:"visibleRegionIds"`
	CorridorRegionID  int                          `json:"corridorRegionId"`
	KnownRegionIDs    []int                        `json:"knownRegionIds"`

	// Viewer identity - who is viewing this snapshot
	ViewerPlayerID string `json:"viewerPlayerId"` // e.g., "player-abc123"
	ViewerRole     string `json:"viewerRole"`     // "gm" or "hero"
	ViewerEntityID string `json:"viewerEntityId"` // e.g., "hero-1" (empty for GM)

	// Dynamic turn order state
	TurnPhase          string   `json:"turnPhase"`
	CycleNumber        int      `json:"cycleNumber"`
	ActiveHeroPlayerID string   `json:"activeHeroPlayerId,omitempty"`
	ElectedPlayerID    string   `json:"electedPlayerId,omitempty"`
	HeroesActedIDs     []string `json:"heroesActedIds"`

	// Quest data for GM
	QuestName         string        `json:"questName,omitempty"`
	QuestDescription  string        `json:"questDescription,omitempty"`
	QuestNotes        string        `json:"questNotes,omitempty"`        // Player-visible notes
	QuestGMNotes      string        `json:"questGMNotes,omitempty"`      // GM-only notes
	QuestObjectives   []string      `json:"questObjectives,omitempty"`   // List of objectives
	StartingPositions []TileAddress `json:"startingPositions,omitempty"` // Valid starting tiles

	// Quest setup state
	PlayersReady         map[string]bool              `json:"playersReady,omitempty"`
	PlayerStartPositions map[string]StartPositionInfo `json:"playerStartPositions,omitempty"`

	// Monster turn states
	MonsterTurnStates map[string]MonsterTurnStateLite `json:"monsterTurnStates,omitempty"`
	SelectedMonsterID string                          `json:"selectedMonsterId,omitempty"`
}

type MonsterTurnStateLite struct {
	MonsterID            string                       `json:"monsterId"`
	EntityID             string                       `json:"entityId"`
	TurnNumber           int                          `json:"turnNumber"`
	CurrentPosition      TileAddress                  `json:"currentPosition"`
	FixedMovement        int                          `json:"fixedMovement"`
	MovementRemaining    int                          `json:"movementRemaining"`
	MovementUsed         int                          `json:"movementUsed"`
	HasMoved             bool                         `json:"hasMoved"`
	ActionTaken          bool                         `json:"actionTaken"`
	ActionType           string                       `json:"actionType,omitempty"`
	AttackDice           int                          `json:"attackDice"`
	DefenseDice          int                          `json:"defenseDice"`
	BodyPoints           int                          `json:"bodyPoints"`
	CurrentBody          int                          `json:"currentBody"`
	SpecialAbilities     []MonsterAbilitySnapshotLite `json:"specialAbilities,omitempty"`
	AbilityUsageThisTurn map[string]int               `json:"abilityUsageThisTurn,omitempty"`
	ActiveEffectsCount   int                          `json:"activeEffectsCount"`
}

type MonsterAbilitySnapshotLite struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	UsesPerTurn    int                    `json:"usesPerTurn"`
	UsesPerQuest   int                    `json:"usesPerQuest"`
	UsesLeftQuest  int                    `json:"usesLeftQuest"`
	RequiresAction bool                   `json:"requiresAction"`
	Range          int                    `json:"range"`
	Description    string                 `json:"description,omitempty"`
	EffectDetails  map[string]interface{} `json:"effectDetails,omitempty"`
}
