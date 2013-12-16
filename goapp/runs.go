// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/taskqueue"
	"appengine/user"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codegangsta/martini"
	"github.com/nightexcessive/bytesize"

	"github.com/hl2-ghosting-team/website/goapp/models"
)

const (
	runsPerPage = 10
	maxRunSize  = 4 * bytesize.MB
)

var (
	validGames  = map[string]string{"half-life2": "Half-Life 2"}
	defaultGame = "half-life2"
	top10Query  = datastore.NewQuery("Run").Order("TotalTime").Project("TotalTime", "UploadTime").Filter("TotalTime >", 0).Limit(runsPerPage) // Get the top 10 runs for this game
)

func getGameName(c *Context) string {
	if requestedGame := c.Req.FormValue("game"); len(requestedGame) > 0 {
		for validGame, _ := range validGames {
			if validGame == requestedGame {
				return validGame
			}
		}
	}

	return defaultGame
}

type exposedRun struct {
	Rank   int
	Run    *models.Run
	RunKey string
	User   *models.User
}

type pagination struct {
	Current    int
	Next, Prev int
	HasPrev    bool
}

func Runs(c *Context) {
	var (
		game = getGameName(c)
		page = 0
	)
	if pageStr := c.Req.URL.Query().Get("page"); len(pageStr) > 0 {
		page64, err := strconv.ParseInt(pageStr, 10, 32)
		if err != nil {
			c.Warningf("Invalid page: %s (%s)", pageStr, err)
		} else {
			page = int(page64)
		}
	}

	runChannel := make(chan *exposedRun, runsPerPage)
	go c.Step("fetch runs", func(c *Context) {
		defer close(runChannel)

		runs := make([]models.Run, 0, runsPerPage) // TODO: We can't use []*models.Run because goon will hate us. Find a fix for this.
		c.Step("run query", func(c *Context) {
			q := top10Query.Offset(page*runsPerPage).Filter("Game =", game)

			if _, err := c.Goon.GetAll(q, &runs); err != nil {
				panic(err)
			}
		})

		users := make([]*models.User, len(runs))
		c.Step("fetch uploaders", func(c *Context) {
			for i, run := range runs {
				users[i] = &models.User{
					ID: run.User.StringID(),
				}
			}
			if err := c.Goon.GetMulti(users); err != nil {
				panic(err)
			}
		})
		// Fetch all of the users

		for i := range runs {
			run := &runs[i]
			runChannel <- &exposedRun{
				Rank:   (page * runsPerPage) + i + 1,
				Run:    run,
				RunKey: c.Goon.Key(run).Encode(),
				User:   users[i],
			}
		}
	})

	c.SetRenderParam("Game", game)
	c.SetRenderParam("PrettyGame", validGames[game])

	exposedRuns := make([]*exposedRun, 0, runsPerPage)
	for run := range runChannel {
		exposedRuns = append(exposedRuns, run)
	}
	c.SetRenderParam("Runs", exposedRuns)

	p := new(pagination)
	p.Current = page
	if len(exposedRuns) == runsPerPage {
		p.Next = page + 1
	}
	if page > 0 {
		p.Prev = page - 1
		p.HasPrev = true
	}
	c.SetRenderParam("Pages", p)

	c.Render()
}

func RunPOST(c *Context) {
	// TODO: Implement
	http.Error(c.Response, "Not yet implemented", http.StatusInternalServerError) // TODO
}

func UploadRun(c *Context) {
	game := getGameName(c)
	c.SetRenderParam("Game", game)
	c.SetRenderParam("MaxRunSize", maxRunSize)

	doneURL, err := routerUrl("upload-run-done")
	if err != nil {
		panic(err)
	}

	if uploadURL, err := blobstore.UploadURL(c, doneURL, &blobstore.UploadURLOptions{MaxUploadBytes: int64(maxRunSize), MaxUploadBytesPerBlob: int64(maxRunSize)}); err == nil {
		c.SetRenderParam("UploadURL", uploadURL)
	} else {
		panic(err)
	}

	c.Render()
}

func UploadRunDone(c *Context) {
	var (
		blobs map[string][]*blobstore.BlobInfo
		form  url.Values
	)
	c.Step("parse uploads", func(c *Context) {
		var err error
		blobs, form, err = blobstore.ParseUpload(c.Req)
		if err != nil {
			panic(err)
		}
	})

	var runBlob *blobstore.BlobInfo
	if runBlobs := blobs["run"]; len(runBlobs) > 0 {
		runBlob = blobs["run"][0]
	}

	c.Step("remove unused blobs", func(c *Context) {
		deleteBlobs := make([]appengine.BlobKey, 0)
		for _, blobList := range blobs {
			for _, blobInfo := range blobList {
				if runBlob == nil || blobInfo.BlobKey != runBlob.BlobKey {
					deleteBlobs = append(deleteBlobs, blobInfo.BlobKey)
				}
			}
		}

		if len(deleteBlobs) > 0 {
			if err := blobstore.DeleteMulti(c, deleteBlobs); err != nil {
				panic(err)
			}
		}
	})

	if runBlob == nil {
		c.Infof("No files uploaded: %#v", blobs)
		http.Redirect(c.Response, c.Req, "/runs/upload", http.StatusSeeOther) // TODO: Improve this
		return
	}

	u := user.Current(c)
	game := getGameName(c)

	var (
		run    *models.Run
		runKey *datastore.Key
	)

	c.Step("insert run", func(c *Context) {
		if err := c.RunInTransaction(func(c *Context) error {
			run = &models.Run{
				User:       datastore.NewKey(c, "User", u.ID, 0, nil),
				UploadTime: time.Now(),

				Game:    game,
				RunFile: runBlob.BlobKey,
			}
			if _, err := c.Goon.Put(run); err != nil {
				return err
			}
			runKey = c.Goon.Key(run)

			taskURL, err := routerUrl("task-process-run")
			if err != nil {
				return err
			}

			taskValues := make(url.Values)
			taskValues.Set("id", runKey.Encode())
			task := taskqueue.NewPOSTTask(taskURL, taskValues)
			task.Name = runKey.Encode()
			if _, err := taskqueue.Add(c, task, "runs"); err != nil {
				return err
			}

			return nil
		}, nil); err != nil {
			panic(err)
		}
	})

	runURL, err := routerUrl("view-run", runKey.Encode())
	if err != nil {
		panic(err)
	}

	http.Redirect(c.Response, c.Req, runURL, http.StatusSeeOther)
}

func ViewRun(c *Context, params martini.Params) {
	runIDstr := params["id"]
	runKey, err := datastore.DecodeKey(runIDstr)
	if err != nil {
		c.Infof("Unable to decode run key: %s", err)
		http.Error(c.Response, "Invalid run ID: "+runIDstr, http.StatusBadRequest)
		return
	}

	run := &models.Run{ID: runKey.IntID(), User: runKey.Parent()}
	stop := false // TODO: Make this feel less hacky
	c.Step("fetch run", func(c *Context) {
		if err := c.Goon.Get(run); err != nil {
			if err == datastore.ErrNoSuchEntity {
				NotFound(c)
				stop = true
				return
			}
			panic(err)
		}
	})
	if stop {
		return
	}

	c.SetRenderParam("Run", run)
	c.SetRenderParam("PrettyGame", validGames[run.Game])
	c.SetRenderParam("RunKey", c.Goon.Key(run))

	var uploader *models.User

	if currentUserInterface, ok := c.GetRenderParam("User"); ok {
		if currentUser, ok := currentUserInterface.(*models.User); ok {
			if currentUser.ID == run.User.StringID() {
				uploader = currentUser
			}
		}
	}

	if uploader == nil {
		uploader = &models.User{ID: run.User.StringID()}
		c.Step("fetch run uploader", func(c *Context) {
			if err := c.Goon.Get(uploader); err != nil {
				if err == datastore.ErrNoSuchEntity {
					uploader = models.CreateDeletedUser()
					return
				}
				panic(err)
			}
		})
	}

	c.SetRenderParam("Uploader", uploader)
	c.SetRenderParam("UploaderKey", c.Goon.Key(uploader))

	if run.FullAnalysis == nil {
		c.SetRenderParam("ExtraHead", template.HTML("<meta http-equiv=\"refresh\" content=\"3\"/>"))
	} else {
		c.Step("fetch full analysis", func(c *Context) {
			analysis := &models.Analysis{ID: run.FullAnalysis.IntID(), Run: c.Goon.Key(run)}
			if err := c.Goon.Get(analysis); err == datastore.ErrNoSuchEntity {
				c.GlobalWG.Add(1)
				go c.Step("update invalid analysis", func(c *Context) {
					defer c.GlobalWG.Done()

					run.FullAnalysis = nil
					if _, err := c.Goon.Put(run); err != nil {
						panic(err)
					}
				})
				return
			} else if err != nil {
				panic(err)
			}
			c.SetRenderParam("FullAnalysis", analysis)
			if numPlayers := len(analysis.Players); numPlayers == 0 {
				c.SetRenderParam("PlayerStatement", "There were no players involved.")
			} else if numPlayers == 1 {
				c.SetRenderParam("PlayerStatement", analysis.Players[0]+" was the runner.")
			} else {
				statement := ""
				for i, player := range analysis.Players {
					if i-1 != numPlayers {
						statement += player + ", "
					} else {
						statement += "and " + player + " "
					}
				}

				statement += "were involved."

				c.SetRenderParam("PlayerStatement", statement)
			}
		})
	}
	c.Render()
}

func DownloadRun(c *Context, params martini.Params) {
	runIDstr := params["id"]
	runKey, err := datastore.DecodeKey(runIDstr)
	if err != nil {
		c.Infof("Unable to decode run key: %s", err)
		http.Error(c.Response, "Invalid run ID: "+runIDstr, http.StatusBadRequest)
		return
	}

	run := &models.Run{ID: runKey.IntID(), User: runKey.Parent()}
	stop := false // TODO: Make this feel less hacky
	c.Step("fetch run", func(c *Context) {
		if err := c.Goon.Get(run); err != nil {
			if err == datastore.ErrNoSuchEntity {
				NotFound(c)
				stop = true
				return
			}
			panic(err)
		}
	})
	if stop {
		return
	}

	headers := c.Response.Header()
	headers.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v.run\"", run.ID))
	headers.Set("Cache-Control", "public, max-age=2592000")
	headers.Set("Pragma", "Public")

	blobstore.Send(c.Response, run.RunFile)
}

func failedAnalysis(c *Context, run *models.Run, reason string) {
	c.GlobalWG.Add(1)
	go c.Step("insert failed analysis", func(c *Context) {
		defer c.GlobalWG.Done()

		analysis := &models.Analysis{
			Run: c.Goon.Key(run),

			Fail:       true,
			FailReason: reason,
		}

		if err := c.RunInTransaction(func(c *Context) error {
			if err := blobstore.Delete(c, run.RunFile); err != nil && err != datastore.ErrNoSuchEntity {
				return err
			}
			run.RunFile = ""
			if _, err := c.Goon.Put(analysis); err != nil {
				return err
			}
			run.FullAnalysis = c.Goon.Key(analysis) // Unfortunately, we can't do a PutMulti because we need to know the key of Analysis.
			if _, err := c.Goon.Put(run); err != nil {
				return err
			}

			return nil
		}, nil); err != nil {
			panic(err)
		}
	})
}

func ProcessRun(c *Context) {
	runIDstr := c.Req.FormValue("id")
	runKey, err := datastore.DecodeKey(runIDstr)
	if err != nil {
		c.Infof("Unable to decode run key (%s): %s", runIDstr, err)
		http.Error(c.Response, "Unable to decode run key: "+runIDstr, http.StatusBadRequest)
		return
	}

	run := &models.Run{ID: runKey.IntID(), User: runKey.Parent()}
	stop := false // TODO: Make this feel less hacky
	c.Step("fetch run", func(c *Context) {
		if err := c.Goon.Get(run); err != nil {
			if err == datastore.ErrNoSuchEntity {
				NotFound(c)
				stop = true
				return
			}
			panic(err)
		}
	})
	if stop {
		return
	}

	if run.FullAnalysis != nil {
		c.Warningf("This run has already been analyzed.")
		if _, err := io.WriteString(c.Response, "Run already analyzed."); err != nil {
			panic(err)
		}
		return
	}

	blobReader := blobstore.NewReader(c, run.RunFile)
	runReader := &models.RunReader{blobReader}

	if verified, err := runReader.VerifyPreamble(); err != nil {
		failedAnalysis(c, run, fmt.Sprintf("Failed to read the preamble (%s)", err))
		return
	} else if !verified {
		failedAnalysis(c, run, "The given file is not a valid run file.")
		return
	}

	header, err := runReader.ReadHeader()
	if err != nil {
		failedAnalysis(c, run, fmt.Sprintf("Failed to read the run header (%s)", err))
		return
	}

	lineNumber := 1

	analysis := &models.Analysis{
		Run: runKey,

		Maps:      make([]models.MapAnalysis, 0),
		Players:   make([]string, 1),
		RawHeader: header.MakeRaw(),
	}

	c.Step("analyzing", func(c *Context) {
		runLine, err := runReader.ReadLine()

		currentMap := models.MapAnalysis{
			Name: runLine.MapName,
		}
		currentMapStart := runLine.Time

		lastPlayerName := runLine.PlayerName
		analysis.Players[0] = lastPlayerName

		lastLine := runLine
		for ; err != io.EOF; runLine, err = runReader.ReadLine() {
			if err != nil { // If the error isn't nil and we've made it here, it's an unexpected error
				failedAnalysis(c, run, fmt.Sprintf("Failed to read line #%d (%s)", lineNumber, err))
				return
			}

			if len(runLine.MapName) > 0 && currentMap.Name != runLine.MapName {
				currentMap.Time = time.Duration((runLine.Time - currentMapStart) * float32(time.Second))
				analysis.Maps = append(analysis.Maps, currentMap)

				currentMap = models.MapAnalysis{
					Name: runLine.MapName,
				}
				currentMapStart = runLine.Time
			}

			if len(runLine.PlayerName) > 0 && lastPlayerName != runLine.PlayerName {
				found := false
				for _, name := range analysis.Players {
					if runLine.PlayerName == name {
						found = true
						break
					}
				}

				if !found {
					analysis.Players = append(analysis.Players, runLine.PlayerName)
				}

				lastPlayerName = runLine.PlayerName
			}

			lastLine = runLine
			lineNumber++
		}

		run.TotalTime = time.Duration(lastLine.Time * float32(time.Second))
	})

	c.Step("insert analysis", func(c *Context) {
		if err := c.RunInTransaction(func(c *Context) error {
			if _, err := c.Goon.Put(analysis); err != nil {
				return err
			}
			run.FullAnalysis = c.Goon.Key(analysis) // Unfortunately, we can't do a PutMulti because we need to know the key of Analysis.
			if _, err := c.Goon.Put(run); err != nil {
				return err
			}

			return nil
		}, nil); err != nil {
			panic(err)
		}
	})

	c.Response.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(c.Response, "Successfully analyzed."); err != nil {
		panic(err)
	}
}
