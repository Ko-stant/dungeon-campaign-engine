package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Throwaway endpoints for the Phase 4 renderer check (/dev/board). They convert
// the legacy board.json + quest-01 into the TypeScript BoardView shape so the
// new renderer can be compared with the legacy /gm view. Deleted in Phase 8,
// when boards and quests come from the database.

type devEdge struct {
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
}

type devTile struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type devDoor struct {
	ID    string  `json:"id"`
	Edge  devEdge `json:"edge"`
	Kind  string  `json:"kind"`
	State string  `json:"state"`
}

type devBlocked struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type devFurniture struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	At       devTile `json:"at"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Rotation int     `json:"rotation"`
	Image    string  `json:"image,omitempty"`
}

type devPiece struct {
	ID    string  `json:"id"`
	Type  string  `json:"type"`
	At    devTile `json:"at"`
	Label string  `json:"label,omitempty"`
	Image string  `json:"image,omitempty"`
}

type devBoardView struct {
	Cols           int            `json:"cols"`
	Rows           int            `json:"rows"`
	Regions        []int          `json:"regions"`
	Doors          []devDoor      `json:"doors"`
	BlockedSquares []devBlocked   `json:"blockedSquares"`
	Furniture      []devFurniture `json:"furniture"`
	Monsters       []devPiece     `json:"monsters"`
	Heroes         []devPiece     `json:"heroes"`
	Traps          []devPiece     `json:"traps"`
}

func buildDevBoardView() (*devBoardView, error) {
	board, quest, err := loadGameContent()
	if err != nil {
		return nil, err
	}
	furniture := NewFurnitureSystem(log.New(io.Discard, "", 0))
	if err := furniture.LoadFurnitureDefinitions("content"); err != nil {
		return nil, err
	}

	regionMap := geometry.CreateRegionMapFromBoard(board)
	view := &devBoardView{
		Cols:           board.Dimensions.Width,
		Rows:           board.Dimensions.Height,
		Regions:        regionMap.TileRegionIDs,
		Doors:          []devDoor{},
		BlockedSquares: []devBlocked{},
		Furniture:      []devFurniture{},
		Monsters:       []devPiece{},
		Heroes:         []devPiece{},
		Traps:          []devPiece{},
	}

	for _, d := range quest.Doors {
		kind := "normal"
		if d.Type == "secret" {
			kind = "secret"
		}
		view.Doors = append(view.Doors, devDoor{
			ID:    d.ID,
			Edge:  devEdge{X: d.X, Y: d.Y, Orientation: d.Orientation},
			Kind:  kind,
			State: d.State,
		})
	}

	// Legacy blocking walls are runs of blocked tiles (see buildBlockedTiles).
	for _, w := range quest.BlockingWalls {
		size := max(w.Size, 1)
		b := devBlocked{X: w.X, Y: w.Y, W: 1, H: size}
		if w.Orientation == "horizontal" {
			b.W, b.H = size, 1
		}
		view.BlockedSquares = append(view.BlockedSquares, b)
	}

	for _, f := range quest.Furniture {
		def := furniture.GetDefinition(f.Type)
		item := devFurniture{ID: f.ID, Type: f.Type, At: devTile{X: f.X, Y: f.Y}, Width: 1, Height: 1, Rotation: f.Rotation}
		if def != nil {
			item.Width, item.Height = def.GridSize.Width, def.GridSize.Height
			item.Image = def.Rendering.TileImageCleaned
			if item.Image == "" {
				item.Image = def.Rendering.TileImage
			}
		}
		view.Furniture = append(view.Furniture, item)
	}

	for _, m := range quest.Monsters {
		view.Monsters = append(view.Monsters, devPiece{
			ID:    m.ID,
			Type:  m.Type,
			At:    devTile{X: m.X, Y: m.Y},
			Label: m.Type,
			Image: "assets/tiles_cleaned/monsters/monster_" + m.Type + ".png",
		})
	}

	return view, nil
}

func registerDevBoardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dev/board", func(w http.ResponseWriter, r *http.Request) {
		if err := views.DevBoardPage().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/dev/board.json", func(w http.ResponseWriter, _ *http.Request) {
		view, err := buildDevBoardView()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(view); err != nil {
			log.Printf("dev board: encode: %v", err)
		}
	})
}
