// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine"
	"appengine/user"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MiniProfiler/go/miniprofiler"
	mpg "github.com/MiniProfiler/go/miniprofiler_gae"
	"github.com/codegangsta/martini"
	"github.com/mjibson/appstats"
	"github.com/mjibson/goon"
)

var templates *template.Template
var routes = make(map[string]martini.Route)

type router struct {
	*martini.Martini
	martini.Router
}

func enableDebugStats(r *http.Request) bool {
	if appengine.IsDevAppServer() {
		return true
	}

	c := appengine.NewContext(r)
	if u := user.Current(c); u != nil {
		return u.Admin
	}

	return strings.HasPrefix(r.URL.Path, miniprofiler.PATH)
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	} else if err != nil {
		panic(err)
	}
	return true
}

// This traverses the file tree to find the "main" directory. The main directory is the directory with the app.yaml file.
func findMainDirectory() {
	startingPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for !fileExists(filepath.Join(startingPath, "app.yaml")) {
		oldStartingPath := startingPath
		startingPath = filepath.Dir(startingPath)
		if startingPath == oldStartingPath {
			panic("Unable to find main directory (a directory containing app.yaml)")
		}
	}

	os.Chdir(startingPath)
}

func init() {
	miniprofiler.Position = "right"
	miniprofiler.ShowControls = false
	miniprofiler.ShowTrivial = true
	miniprofiler.TrivialMilliseconds = 5
	miniprofiler.Enable = enableDebugStats
	appstats.ShouldRecord = enableDebugStats
	goon.LogErrors = false

	findMainDirectory()

	var err error
	if templates, err = template.New("").Funcs(funcs).ParseGlob("templates/*.html"); err != nil {
		panic(err)
	}

	m := &router{
		Martini: martini.New(),
		Router:  martini.NewRouter(),
	}

	m.Use(panicRecoverer)
	m.Use(contextCreator)

	routes["index"] = m.Get("/", Index)

	routes["upload-run"] = m.Get("/runs/upload", UploadRun)
	routes["upload-run-done"] = m.Post("/runs/upload/done", UploadRunDone)
	routes["task-process-run"] = m.Post("/tasks/run/process", ProcessRun)

	routes["runs"] = m.Get("/runs", Runs)
	routes["download-run"] = m.Get("/runs/:id/download", DownloadRun)
	routes["view-run"] = m.Get("/runs/:id", ViewRun)
	routes["update-run"] = m.Post("/runs/:id", RunPOST)

	routes["login"] = m.Get("/login", LoginGoogle)
	routes["logout"] = m.Get("/logout", LogoutGoogle)
	routes["view-user"] = m.Get("/user/:id", ViewUser)

	m.NotFound(NotFound)

	m.Action(m.Router.Handle)
	http.Handle("/", m)
}

func panicRecoverer(c martini.Context, r *http.Request, w http.ResponseWriter) {
	defer func() {
		if err := recover(); err != nil {
			serveError(appengine.NewContext(r), fmt.Errorf("recovered from panic: %s", err), w)
		}
	}()

	c.Next()
}

func contextCreator(c martini.Context, r *http.Request, rawWriter http.ResponseWriter) {
	statsEnabled := enableDebugStats(r)
	w := rawWriter.(martini.ResponseWriter)

	appstatsContext := appstats.NewContext(r)

	nc := &Context{
		ID:         appengine.RequestID(appstatsContext),
		Req:        r,
		Response:   w,
		GlobalWG:   new(sync.WaitGroup),
		RenderWG:   new(sync.WaitGroup),
		IncludesWG: new(sync.WaitGroup),
	}

	profile := miniprofiler.NewProfile(w, r, "TODO")
	nc.Context = &mpg.Context{
		Context: appstatsContext,
		Timer:   profile,
	}

	nc.Goon = goon.FromContext(nc)

	c.Map(nc)

	nc.Infof("Request ID: %s", nc.ID)
	if statsEnabled {
		values := make(url.Values)
		values.Set("id", profile.Id)

		resultURL := &url.URL{
			Host:     r.Host,
			Path:     miniprofiler.PATH + "results",
			RawQuery: values.Encode(),
		}

		if r.TLS == nil {
			resultURL.Scheme = "http"
		} else {
			resultURL.Scheme = "https"
		}

		nc.Infof("Profile: %s", resultURL)
	}

	nc.includeLock = new(sync.RWMutex)
	nc.includes = make(map[string]interface{})

	nc.RenderWG.Add(1)
	nc.IncludesWG.Add(1)
	go nc.createIncludes()

	c.Next()

	nc.RenderWG.Wait()
	nc.GlobalWG.Wait()

	if statsEnabled {
		nc.Context.Timer.AddCustomLink("appstats", appstatsContext.URL())
		appstatsContext.Stats.Status = w.Status()
		appstatsContext.Save()
		profile.Finalize()
	}
}

func checkReferer(c *Context) {
	if c.Req.Method != "POST" {
		return
	}

	referer := c.Req.Referer()
	if len(referer) <= 0 {
		return
	}

	refererURL, err := url.Parse(referer)
	if err != nil {
		c.Criticalf("Potential XSS attack. Invalid referer: %s", referer)
		http.Error(c.Response, "Potential XSS attack detected.\nRequest ID: "+c.ID, http.StatusForbidden)
		return
	}

	if refererURL.Host != c.Req.Host {
		c.Criticalf("Potential XSS attack. Bad host: %s != %s", refererURL.Host, c.Req.Host)
		http.Error(c.Response, "Potential XSS attack detected.\nRequest ID: "+c.ID, http.StatusForbidden)
		return
	}
}
