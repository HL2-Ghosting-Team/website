// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"appengine/blobstore"
	"appengine/datastore"
	"net/http"

	"github.com/HL2-Ghosting-Team/website/endpoint/utils"
	"github.com/HL2-Ghosting-Team/website/models"
)

func init() {
	if getRunInfo := ghostingService.MethodByName("GetRun").Info(); getRunInfo != nil {
		getRunInfo.Name, getRunInfo.HttpMethod, getRunInfo.Path, getRunInfo.Desc =
			"runs.get", "GET", "runs/get/{id}", "Retrieve information about a single run along with its analysis."
	}

	if getRunUploadURL := ghostingService.MethodByName("GetUploadURL").Info(); getRunUploadURL != nil {
		getRunUploadURL.Name, getRunUploadURL.HttpMethod, getRunUploadURL.Path, getRunUploadURL.Desc, getRunUploadURL.Scopes =
			"runs.upload", "POST", "runs/upload", "Retrieve a URL used for uploading a run.", []string{utils.UserScope}
	}
}

// An error returned when a given run ID is invalid. This is usually due to the
// run ID being in an incorrect format.
type ErrInvalidRunID struct {
	ID string
}

func (e *ErrInvalidRunID) Error() string {
	return "Invalid run ID: " + e.ID
}

func (e *ErrInvalidRunID) HTTPStatus() int {
	return 400
}

// An error returned when a given run ID does not exist in the datastore.
type ErrNoSuchRun struct {
}

func (e *ErrNoSuchRun) Error() string {
	return "No such run exists"
}

func (e *ErrNoSuchRun) HTTPStatus() int {
	return 404
}

type GetRunRequest struct {
	ID string `json:"id" endpoints:"req,desc=The ID of the desired run"`
}

type GetRunResponse struct {
	ID      *datastore.Key `json:"id" endpoints:"desc=The ID of the returned run"`
	Deleted bool           `json:"deleted,omitempty" endpoints:"desc=Whether or not the run has been deleted"`

	Run *models.Run `json:"run" endpoints:"desc=The run's data"`

	Analysis *models.Analysis `json:"analysis" endpoints:"desc=The analysis of the returned run"`
}

// Retrieves a run with a specified ID. The method is named ghosting.runs.get
// and is located at runs/get/{id}.
func (*GhostingService) GetRun(r *http.Request, req *GetRunRequest, res *GetRunResponse) error {
	c := utils.NewContext(r)

	runKey, err := datastore.DecodeKey(req.ID)
	if err != nil {
		c.Infof("Unable to decode run key: %s", err)
		return &ErrInvalidRunID{ID: req.ID}
	}
	res.ID = runKey

	res.Run = &models.Run{ID: runKey.IntID(), User: runKey.Parent()}
	if err := c.Goon.Get(res.Run); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return new(ErrNoSuchRun)
		}
		panic(err)
	}

	res.Analysis = &models.Analysis{
		ID:  res.Run.FullAnalysis.IntID(),
		Run: c.Goon.Key(res.Run),
	}
	if err := c.Goon.Get(res.Analysis); err != nil {
		if err == datastore.ErrNoSuchEntity {
			res.Analysis = nil
		}
		panic(err)
	}
	if res.Analysis != nil {
		res.Analysis.MakeHeader()
	}

	return nil
}

type GetUploadURLRequest struct {
}

type GetUploadURLResponse struct {
	UploadURL string `json:"upload_url" endpoints:"desc=The URL to which the file form should be posted. NOTE: This upload URL is tied to its creator. It should not be shared."`
}

// Retrieves a URL to be used for uploading a run file.
func (*GhostingService) GetUploadURL(r *http.Request, req *GetUploadURLRequest, res *GetUploadURLResponse) error {
	c := utils.NewContext(r)

	currentUser, err := c.GetUser()
	if err != nil {
		panic(err)
	} else if currentUser == nil {
		return new(ErrLoginRequired)
	}

	if uploadURL, err := blobstore.UploadURL(c, "/internal/upload-done/"+currentUser.ID, &blobstore.UploadURLOptions{
		MaxUploadBytes:        4 * 1024 * 1024,
		MaxUploadBytesPerBlob: 4 * 1024 * 1024,
	}); err == nil {
		res.UploadURL = uploadURL.String()
	} else {
		panic(err)
	}
	return nil
}
