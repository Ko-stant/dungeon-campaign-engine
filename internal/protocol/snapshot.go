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
	ID   string      `json:"id"`
	Kind string      `json:"kind"`
	Tile TileAddress `json:"tile"`
	HP   *HP         `json:"hp,omitempty"`
	Tags []string    `json:"tags,omitempty"`
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
	ID                string      `json:"id"`
	Type              string      `json:"type"`
	Tile              TileAddress `json:"tile"`
	GridSize          struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rotation          int         `json:"rotation,omitempty"` // 0, 90, 180, 270 degrees
	SwapAspectOnRotate bool        `json:"swapAspectOnRotate,omitempty"` // Whether to swap width/height for 90/270 rotations
	TileImage         string      `json:"tileImage"`
	TileImageCleaned  string      `json:"tileImageCleaned"`
	PixelDimensions   struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"pixelDimensions"`
	BlocksLineOfSight bool        `json:"blocksLineOfSight"`
	BlocksMovement    bool        `json:"blocksMovement"`
	Contains          []string    `json:"contains,omitempty"`
}

type Snapshot struct {
	MapID             string             `json:"mapId"`
	PackID            string             `json:"packId"`
	Turn              int                `json:"turn"`
	LastEventID       int64              `json:"lastEventId"`
	MapWidth          int                `json:"mapWidth"`
	MapHeight         int                `json:"mapHeight"`
	RegionsCount      int                `json:"regionsCount"`
	TileRegionIDs     []int              `json:"tileRegionIds"`
	RevealedRegionIDs []int              `json:"revealedRegionIds"`
	DoorStates        []byte             `json:"doorStates"`
	Entities          []EntityLite       `json:"entities"`
	Thresholds        []ThresholdLite    `json:"thresholds"`
	BlockingWalls     []BlockingWallLite `json:"blockingWalls"`
	Furniture         []FurnitureLite    `json:"furniture"`
	Variables         map[string]any     `json:"variables"`
	ProtocolVersion   string             `json:"protocolVersion"`
	VisibleRegionIDs  []int              `json:"visibleRegionIds"`
	CorridorRegionID  int                `json:"corridorRegionId"`
	KnownRegionIDs    []int              `json:"knownRegionIds"`
}
