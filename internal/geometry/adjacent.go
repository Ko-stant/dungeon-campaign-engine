package geometry

func RegionsAcrossDoor(regionMap RegionMap, segment Segment, edge EdgeAddress) (int, int) {
	if edge.Orientation == Vertical {
		leftIdx := edge.Y*segment.Width + edge.X
		rightIdx := edge.Y*segment.Width + (edge.X + 1)
		return regionMap.TileRegionIDs[leftIdx], regionMap.TileRegionIDs[rightIdx]
	}
	upIdx := edge.Y*segment.Width + edge.X
	downIdx := (edge.Y+1)*segment.Width + edge.X
	return regionMap.TileRegionIDs[upIdx], regionMap.TileRegionIDs[downIdx]
}
