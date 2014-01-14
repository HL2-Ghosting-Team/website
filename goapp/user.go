// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine/datastore"
	"net/http"

	"github.com/codegangsta/martini"

	"github.com/HL2-Ghosting-Team/website/goapp/models"
)

var (
	recentlyUploadedPerPage = 10
	recentlyUploadedQuery   = datastore.NewQuery("Run").Order("-UploadTime").Project("UploadTime", "Game", "TotalTime", "RunFile").Limit(runsPerPage) // Get the top 10 runs for this game
)

type recentRunInternal struct {
	Run       *models.Run
	RunKey    *datastore.Key
	RunStatus string
}

func ViewUser(c *Context, params martini.Params) {
	// TODO: Do more with this
	userIDstr := params["id"]
	userKey, err := datastore.DecodeKey(userIDstr)
	if err != nil {
		c.Infof("Unable to decode user key (%s): %s", userIDstr, err)
		http.Error(c.Response, "Invalid user ID: "+userIDstr, http.StatusBadRequest)
		return
	}

	page := 0 // TODO: Actually check the page

	recentlyUploadedChan := make(chan *models.Run, recentlyUploadedPerPage)
	go c.Step("fetch recent runs", func(c *Context) {
		defer close(recentlyUploadedChan)

		q := recentlyUploadedQuery.Offset(page * recentlyUploadedPerPage).Ancestor(userKey)

		for it := c.Goon.Run(q); ; {
			run := new(models.Run)
			if _, err := it.Next(run); err == datastore.Done {
				break
			} else if err != nil {
				panic(err)
			}

			recentlyUploadedChan <- run
		}
	})

	displayUserChan := make(chan *models.User, 1)
	go c.Step("fetch display user", func(c *Context) {
		defer close(displayUserChan)
		displayUser := &models.User{ID: userKey.StringID()}
		if err := c.Goon.Get(displayUser); err == datastore.ErrNoSuchEntity {
			NotFound(c)
			displayUserChan <- nil
			return
		} else if err != nil {
			panic(err)
		}
		displayUserChan <- displayUser
	})

	displayUser := <-displayUserChan
	if displayUser == nil {
		return
	}
	c.SetRenderParam("DisplayUser", displayUser)

	recentRuns := make([]*recentRunInternal, 0, recentlyUploadedPerPage)
	for upload := range recentlyUploadedChan {
		internalStruct := &recentRunInternal{
			Run:    upload,
			RunKey: c.Goon.Key(upload),
		}
		if upload.RunFile == "" {
			internalStruct.RunStatus = "danger"
		} else if upload.TotalTime == 0 {
			internalStruct.RunStatus = "active"
		} else {
			internalStruct.RunStatus = "success"
		}
		recentRuns = append(recentRuns, internalStruct)
	}
	c.SetRenderParam("RecentRuns", recentRuns)

	c.Render()
}
