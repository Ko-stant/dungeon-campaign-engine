package geometry

func CorridorsAndRoomsSegment(width, height int) Segment {
	isCorr := func(x, y int) bool {
		if x == 0 || y == 0 || x == width-1 || y == height-1 {
			return true
		}
		v1 := (width/2 - 1)
		v2 := (width / 2)
		if x == v1 || x == v2 {
			return true
		}
		h := height / 2
		return y == h
	}

	wallsV := make([]EdgeAddress, 0, width*height/2)
	wallsH := make([]EdgeAddress, 0, width*height/2)

	for y := 0; y < height; y++ {
		for x := 0; x < width-1; x++ {
			if isCorr(x, y) != isCorr(x+1, y) {
				wallsV = append(wallsV, EdgeAddress{X: x, Y: y, Orientation: Vertical})
			}
		}
	}
	for y := 0; y < height-1; y++ {
		for x := 0; x < width; x++ {
			if isCorr(x, y) != isCorr(x, y+1) {
				wallsH = append(wallsH, EdgeAddress{X: x, Y: y, Orientation: Horizontal})
			}
		}
	}

	return Segment{
		ID:              "dev-seg-0",
		Width:           width,
		Height:          height,
		WallsVertical:   wallsV,
		WallsHorizontal: wallsH,
		DoorSockets:     nil,
	}
}

func CorridorsAndRoomsWithDoorsSegment(width, height int) Segment {
	isCorr := func(x, y int) bool {
		if x == 0 || y == 0 || x == width-1 || y == height-1 {
			return true
		}
		v1 := (width/2 - 1)
		v2 := (width / 2)
		if x == v1 || x == v2 {
			return true
		}
		h := height / 2
		return y == h
	}

	wallsV := make([]EdgeAddress, 0, width*height/2)
	wallsH := make([]EdgeAddress, 0, width*height/2)

	for y := 0; y < height; y++ {
		for x := 0; x < width-1; x++ {
			if isCorr(x, y) != isCorr(x+1, y) {
				wallsV = append(wallsV, EdgeAddress{X: x, Y: y, Orientation: Vertical})
			}
		}
	}
	for y := 0; y < height-1; y++ {
		for x := 0; x < width; x++ {
			if isCorr(x, y) != isCorr(x, y+1) {
				wallsH = append(wallsH, EdgeAddress{X: x, Y: y, Orientation: Horizontal})
			}
		}
	}

	v1 := (width/2 - 1)
	v2 := (width / 2)
	h := height / 2

	mid := func(a, b int) int { return (a + b) / 2 }

	tlx := mid(1, v1-1)
	trx := mid(v2, width-2)
	blx := tlx
	brx := trx

	doorSockets := []EdgeAddress{
		{X: tlx, Y: h - 1, Orientation: Horizontal}, // top-left room → central horizontal corridor
		{X: trx, Y: h - 1, Orientation: Horizontal}, // top-right room → central horizontal corridor
		{X: blx, Y: h, Orientation: Horizontal},     // bottom-left room ← central horizontal corridor
		{X: brx, Y: h, Orientation: Horizontal},     // bottom-right room ← central horizontal corridor
	}

	doorSetH := make(map[EdgeAddress]struct{}, len(doorSockets))
	for _, e := range doorSockets {
		if e.Orientation == Horizontal {
			doorSetH[e] = struct{}{}
		}
	}

	filteredH := make([]EdgeAddress, 0, len(wallsH))
	for _, e := range wallsH {
		if _, isDoor := doorSetH[e]; !isDoor {
			filteredH = append(filteredH, e)
		}
	}

	return Segment{
		ID:              "dev-seg-0",
		Width:           width,
		Height:          height,
		WallsVertical:   wallsV,
		WallsHorizontal: filteredH,
		DoorSockets:     doorSockets,
	}
}
