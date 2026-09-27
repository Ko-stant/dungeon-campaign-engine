// Package seed imports the legacy board.json and quest files into the database
// as map-creator documents.
package seed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// Result reports what an import did.
type Result struct {
	BoardID      string
	QuestID      string
	BoardCreated bool
	QuestCreated bool
}

// ImportLegacy converts a legacy board and quest and stores them. A board with
// the same name, or a quest with the same name on that board, is reused rather
// than duplicated, so running it twice is safe.
func ImportLegacy(ctx context.Context, st *store.Store, boardDef *geometry.BoardDefinition, questDef *geometry.QuestDefinition) (Result, error) {
	board, err := maps.BoardFromLegacy(boardDef)
	if err != nil {
		return Result{}, fmt.Errorf("convert board: %w", err)
	}
	quest, err := maps.QuestFromLegacy(questDef, board)
	if err != nil {
		return Result{}, fmt.Errorf("convert quest: %w", err)
	}

	var res Result
	boards, err := st.ListBoards(ctx)
	if err != nil {
		return Result{}, err
	}
	for _, b := range boards {
		if b.Name == boardDef.Name {
			res.BoardID = b.ID
			break
		}
	}
	if res.BoardID == "" {
		doc, err := json.Marshal(board)
		if err != nil {
			return Result{}, err
		}
		created, err := st.CreateBoard(ctx, boardDef.Name, board.Width, board.Height, doc)
		if err != nil {
			return Result{}, fmt.Errorf("store board: %w", err)
		}
		res.BoardID, res.BoardCreated = created.ID, true
	}

	quests, err := st.ListQuests(ctx, res.BoardID)
	if err != nil {
		return Result{}, err
	}
	for _, q := range quests {
		if q.Name == questDef.Name {
			res.QuestID = q.ID
			return res, nil
		}
	}
	doc, err := json.Marshal(quest)
	if err != nil {
		return Result{}, err
	}
	created, err := st.CreateQuest(ctx, res.BoardID, questDef.Name, doc)
	if err != nil {
		return Result{}, fmt.Errorf("store quest: %w", err)
	}
	res.QuestID, res.QuestCreated = created.ID, true
	return res, nil
}
