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

type Snapshot struct {
	MapID           string         `json:"mapId"`
	PackID          string         `json:"packId"`
	Turn            int            `json:"turn"`
	LastEventID     int64          `json:"lastEventId"`
	MapWidth        int            `json:"mapWidth"`
	MapHeight       int            `json:"mapHeight"`
	RegionsCount    int            `json:"regionsCount"`
	DoorStates      []byte         `json:"doorStates"`
	RevealedRegions []byte         `json:"revealedRegions"`
	Entities        []EntityLite   `json:"entities"`
	Variables       map[string]any `json:"variables"`
	ProtocolVersion string         `json:"protocolVersion"`
}
