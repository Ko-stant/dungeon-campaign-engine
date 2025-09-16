package geometry

func RegionsAcrossDoor(regionMap RegionMap, segment Segment, edge EdgeAddress) (int, int) {
	// Helper function to get region ID for a tile, returning -1 if out of bounds
	getTileRegion := func(x, y int) int {
		if x < 0 || x >= segment.Width || y < 0 || y >= segment.Height {
			return -1 // Out of bounds region
		}
		idx := y*segment.Width + x
		if idx < 0 || idx >= len(regionMap.TileRegionIDs) {
			return -1 // Safety check
		}
		return regionMap.TileRegionIDs[idx]
	}

	if edge.Orientation == Vertical {
		// Vertical door at (x,y) = left edge of tile (x,y)
		leftRegion := getTileRegion(edge.X-1, edge.Y)  // tile to the left
		rightRegion := getTileRegion(edge.X, edge.Y)   // tile at (x,y)
		return leftRegion, rightRegion
	}
	// Horizontal door at (x,y) = top edge of tile (x,y)
	upRegion := getTileRegion(edge.X, edge.Y-1)   // tile above
	downRegion := getTileRegion(edge.X, edge.Y)   // tile at (x,y)
	return upRegion, downRegion
}
