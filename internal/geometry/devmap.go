package geometry

func DevSegment() Segment {
	w := 26
	h := 19

	wallsV := make([]EdgeAddress, 0, h)
	wallsH := make([]EdgeAddress, 0)
	doorSockets := make([]EdgeAddress, 0, 1)

	// Full-height vertical wall at x=12, splits the board into left/right.
	for y := 0; y < h; y++ {
		wallsV = append(wallsV, EdgeAddress{X: 12, Y: y, Orientation: Vertical})
	}

	// Single door socket along that wall at (12,9) vertical edge.
	doorSockets = append(doorSockets, EdgeAddress{X: 12, Y: 9, Orientation: Vertical})

	return Segment{
		ID:              "dev-seg-0",
		Width:           w,
		Height:          h,
		WallsVertical:   wallsV,
		WallsHorizontal: wallsH,
		DoorSockets:     doorSockets,
	}
}
