// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"appengine/datastore"
	"net/http"

	"github.com/HL2-Ghosting-Team/website/endpoint/utils"
	"github.com/HL2-Ghosting-Team/website/models"
	"github.com/HL2-Ghosting-Team/website/runs"
)

func init() {
	if getTopRuns := ghostingService.MethodByName("GetTopRuns").Info(); getTopRuns != nil {
		getTopRuns.Name, getTopRuns.HttpMethod, getTopRuns.Path, getTopRuns.Desc =
			"runs.top", "GET", "runs/top/{game}", "Retrieve the top runs."
	}
}

var (
	topQuery = datastore.NewQuery("Run").Order("TotalTime").Filter("Ranked =", true).Filter("TotalTime >", 0)
)

// An error returned when a given game ID is invalid. This is usually due to the
// game ID not being known.
type ErrInvalidGameID struct {
	ID string
}

func (e *ErrInvalidGameID) Error() string {
	return "Invalid game ID: " + e.ID
}

func (e *ErrInvalidGameID) HTTPStatus() int {
	return http.StatusBadRequest
}

type GetTopRunsRequest struct {
	Game  string `json:"game" endpoints:"req,desc=The ID, short name, or long name of the game (e.g.: 0, half-life2, Half-Life 2)"`
	Limit int    `json:"limit" endpoints:"d=10,desc=How many runs to limit the response to. Note: This may not exceed 50."`
}

type GetTopRunsResponse struct {
	Runs []models.Run `json:"items" endpoints:"desc=The returned runs"`
}

// Retrieves the top runs for a given game.
func (*GhostingService) GetTopRuns(r *http.Request, req *GetTopRunsRequest, res *GetTopRunsResponse) error {
	c := utils.NewContext(r)

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}

	gameID, ok := runs.DetermineGame(req.Game)
	if !ok {
		return &ErrInvalidGameID{ID: req.Game}
	}

	_, ok = runs.PrettyGameNames[gameID]
	if !ok {
		return &ErrInvalidGameID{ID: req.Game}
	}

	res.Runs = make([]models.Run, 0, req.Limit)
	q := topQuery.Limit(req.Limit).Filter("Game =", int(gameID))
	for t := c.Goon.Run(q); ; {
		var run models.Run
		if _, err := t.Next(&run); err == datastore.Done {
			break
		} else if err != nil {
			panic(err)
		}

		res.Runs = append(res.Runs, run)
	}

	return nil
}
