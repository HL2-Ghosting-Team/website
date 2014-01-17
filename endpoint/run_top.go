// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"appengine/datastore"
	"net/http"
	"time"

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
	topQuery = datastore.NewQuery("Run").Project("UploadTime", "TotalTime").Order("TotalTime").Filter("Ranked =", true).Filter("TotalTime >", 0)
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
	Game   string `json:"game,omitempty" endpoints:"req" endpoints_desc:"The ID, short name, or long name of the game (e.g.: 0, half-life2, Half-Life 2)"`
	Limit  int    `json:"limit,omitempty" endpoints:"d=10" endpoints_desc:"How many runs to limit the response to. Note: This may not exceed 50."`
	Cursor string `json:"cursor,omitempty" endpoints_desc:"Cursor representing the next page"`
}

type internalRun struct {
	ID         *datastore.Key `json:"id" endpoints_desc:"The ID of this run"`
	Uploader   *datastore.Key `json:"uploader" endpoints_desc:"The ID of the user that uploaded this run"`
	UploadTime time.Time      `json:"upload_time" endpoints_desc:"The upload time of this run"`
	TotalTime  time.Duration  `json:"total_time_ns" endpoints_desc:"The total time (in nanoseconds) that this run took"`
}

type GetTopRunsResponse struct {
	Runs     []*internalRun     `json:"items" endpoints_desc:"The returned runs"`
	RunCount int                `json:"currentItemCount"`
	Limit    int                `json:"itemsPerPage"`
	Next     *GetTopRunsRequest `json:"next,omitempty" endpoints_desc:"The parameters for the next set of runs"`
}

// Retrieves the top runs for a given game.
func (*GhostingService) GetTopRuns(r *http.Request, req *GetTopRunsRequest, res *GetTopRunsResponse) error {
	c := utils.NewContext(r)

	if req.Limit <= 0 {
		req.Limit = 10
	} else if req.Limit > 50 {
		req.Limit = 50
	}

	res.Limit = req.Limit

	gameID, ok := runs.DetermineGame(req.Game)
	if !ok {
		return &ErrInvalidGameID{ID: req.Game}
	}

	prettyGame, ok := runs.PrettyGameNames[gameID]
	if !ok {
		return &ErrInvalidGameID{ID: req.Game}
	}

	var (
		lastCursor string
		hasNext    bool
	)
	res.Runs = make([]*internalRun, 0, req.Limit)

	q := topQuery.Limit(req.Limit+1).Filter("Game =", int(gameID))

	if len(req.Cursor) > 0 {
		if cursor, err := datastore.DecodeCursor(req.Cursor); err != nil {
			return new(ErrCursorInvalid)
		} else {
			q = q.Start(cursor)
		}
	}

	for t, i := c.Goon.Run(q), 0; ; i++ {
		var run models.Run
		runKey, err := t.Next(&run)
		if err == datastore.Done {
			break
		} else if err != nil {
			panic(err)
		}

		if i < req.Limit {
			res.Runs = append(res.Runs, &internalRun{
				ID:         runKey,
				Uploader:   runKey.Parent(),
				UploadTime: run.UploadTime,
				TotalTime:  run.TotalTime,
			})

			if lastCursor_, err := t.Cursor(); err != nil {
				panic(err)
			} else {
				lastCursor = lastCursor_.String()
			}
		} else {
			hasNext = true
		}
	}
	res.RunCount = len(res.Runs)
	if hasNext {
		res.Next = &GetTopRunsRequest{
			Game:   prettyGame,
			Limit:  req.Limit,
			Cursor: lastCursor,
		}
	}

	return nil
}
