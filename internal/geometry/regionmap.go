package geometry

func BuildRegionMap(segment Segment) RegionMap {
	w := segment.Width
	h := segment.Height
	total := w * h
	tileRegionIDs := make([]int, total)
	for i := range tileRegionIDs {
		tileRegionIDs[i] = -1
	}
	blocked := make(map[EdgeAddress]struct{}, len(segment.WallsVertical)+len(segment.WallsHorizontal)+len(segment.DoorSockets))
	for _, e := range segment.WallsVertical {
		blocked[e] = struct{}{}
	}
	for _, e := range segment.WallsHorizontal {
		blocked[e] = struct{}{}
	}
	for _, e := range segment.DoorSockets {
		blocked[e] = struct{}{}
	}

	regionID := 0
	qx := make([]int, 0, total)
	qy := make([]int, 0, total)

	for y := range h {
		for x := range w {
			idx := y*w + x
			if tileRegionIDs[idx] != -1 {
				continue
			}
			tileRegionIDs[idx] = regionID
			qx = qx[:0]
			qy = qy[:0]
			qx = append(qx, x)
			qy = append(qy, y)

			for len(qx) > 0 {
				cx := qx[0]
				cy := qy[0]
				qx = qx[1:]
				qy = qy[1:]

				if cx > 0 {
					leftEdge := EdgeAddress{X: cx - 1, Y: cy, Orientation: Vertical}
					if _, ok := blocked[leftEdge]; !ok {
						nx := cx - 1
						ny := cy
						nidx := ny*w + nx
						if tileRegionIDs[nidx] == -1 {
							tileRegionIDs[nidx] = regionID
							qx = append(qx, nx)
							qy = append(qy, ny)
						}
					}
				}
				if cx < w-1 {
					rightEdge := EdgeAddress{X: cx, Y: cy, Orientation: Vertical}
					if _, ok := blocked[rightEdge]; !ok {
						nx := cx + 1
						ny := cy
						nidx := ny*w + nx
						if tileRegionIDs[nidx] == -1 {
							tileRegionIDs[nidx] = regionID
							qx = append(qx, nx)
							qy = append(qy, ny)
						}
					}
				}
				if cy > 0 {
					upEdge := EdgeAddress{X: cx, Y: cy - 1, Orientation: Horizontal}
					if _, ok := blocked[upEdge]; !ok {
						nx := cx
						ny := cy - 1
						nidx := ny*w + nx
						if tileRegionIDs[nidx] == -1 {
							tileRegionIDs[nidx] = regionID
							qx = append(qx, nx)
							qy = append(qy, ny)
						}
					}
				}
				if cy < h-1 {
					downEdge := EdgeAddress{X: cx, Y: cy, Orientation: Horizontal}
					if _, ok := blocked[downEdge]; !ok {
						nx := cx
						ny := cy + 1
						nidx := ny*w + nx
						if tileRegionIDs[nidx] == -1 {
							tileRegionIDs[nidx] = regionID
							qx = append(qx, nx)
							qy = append(qy, ny)
						}
					}
				}
			}
			regionID++
		}
	}

	return RegionMap{TileRegionIDs: tileRegionIDs, RegionsCount: regionID}
}
