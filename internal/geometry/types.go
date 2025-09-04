package geometry

type Orientation string

const (
	Vertical   Orientation = "vertical"
	Horizontal Orientation = "horizontal"
)

type EdgeAddress struct {
	X           int
	Y           int
	Orientation Orientation
}

type Segment struct {
	ID              string
	Width           int
	Height          int
	WallsVertical   []EdgeAddress
	WallsHorizontal []EdgeAddress
	DoorSockets     []EdgeAddress
}

type RegionMap struct {
	TileRegionIDs []int
	RegionsCount  int
}
