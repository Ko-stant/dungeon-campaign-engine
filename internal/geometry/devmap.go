package geometry

func DevSegment() Segment {
	w := 26
	h := 19

	wallsV := make([]EdgeAddress, 0, 64)
	wallsH := make([]EdgeAddress, 0, 64)
	doorSockets := make([]EdgeAddress, 0, 8)

	for y := 3; y <= 15; y++ {
		wallsV = append(wallsV, EdgeAddress{X: 12, Y: y, Orientation: Vertical})
	}

	for x := 6; x <= 20; x++ {
		wallsH = append(wallsH, EdgeAddress{X: x, Y: 9, Orientation: Horizontal})
	}

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
