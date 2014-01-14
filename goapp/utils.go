// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	//"github.com/MiniProfiler/go/miniprofiler"
	mpg "github.com/MiniProfiler/go/miniprofiler_gae"
	"github.com/codegangsta/martini"
	"github.com/mjibson/goon"

	"github.com/HL2-Ghosting-Team/website/goapp/models"
)

type Context struct {
	*mpg.Context

	ID       string
	Req      *http.Request // We have to name it this or it conflicts with the appengine.Context interface
	Response martini.ResponseWriter
	GlobalWG *sync.WaitGroup
	RenderWG *sync.WaitGroup

	includes    Includes
	includeLock *sync.RWMutex

	Goon *goon.Goon
}

func (c *Context) Step(name string, f func(*Context)) {
	c.Context.Step(name, func(mc mpg.Context) {
		c := &Context{
			Context: &mc,

			ID:       c.ID,
			Req:      c.Req,
			Response: c.Response,
			GlobalWG: c.GlobalWG,

			includes:    c.includes,
			includeLock: c.includeLock,
		}
		c.Goon = goon.FromContext(c)
		f(c)
	})
}

func (c *Context) RunInTransaction(f func(*Context) error, opts *datastore.TransactionOptions) error {
	return c.Goon.RunInTransaction(func(g *goon.Goon) error {
		return f(&Context{
			Context: c.Context,

			ID:       c.ID,
			Req:      c.Req,
			Response: c.Response,
			GlobalWG: c.GlobalWG,

			includes:    c.includes,
			includeLock: c.includeLock,

			Goon: g,
		})
	}, opts)
}

func (c *Context) Render() {
	pc, _, _, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)

	c.RenderWG.Wait()
	c.includeLock.Lock()
	defer c.includeLock.Unlock()

	serveTemplate(c, f.Name()+".html", c.includes)
}

// This should only be called from the root Context. It is normally called at the beginning of a request.
func (c *Context) createIncludes() {
	defer c.RenderWG.Done()

	c.Step("create includes", func(c *Context) {
		c.FillRenderParams(baseRenderParams)
		c.SetRenderParam("MiniProfiler", c.Context.Includes())

		/*if currentRoute := mux.CurrentRoute(c.Req); currentRoute != nil {
			includes["CurrentPage"] = currentRoute.GetName() // TODO: Redo this with Martini
		}*/

		if aeUser := user.Current(c); aeUser != nil {
			c.Step("fetch current user", func(c *Context) {
				user := &models.User{
					ID: aeUser.ID,
				}

				if err := c.Goon.Get(user); err == datastore.ErrNoSuchEntity {
					user.Email = aeUser.Email
					atSign := strings.LastIndex(user.Email, "@")
					user.Nickname = user.Email[:atSign]
					user.Admin = aeUser.Admin

					c.GlobalWG.Add(1)
					go c.Step("create current user", func(c *Context) {
						defer c.GlobalWG.Done()
						if _, err := c.Goon.Put(user); err != nil {
							panic(err)
						}
					})
				} else if err != nil {
					panic(err)
				}

				update := false
				if user.Email != aeUser.Email {
					user.Email = aeUser.Email
					update = true
				}
				if user.Admin == false && user.Admin != aeUser.Admin { // Make them an admin if they're an application admin, but don't de-admin them if they're not.
					user.Email = aeUser.Email
					update = true
				}
				if update {
					c.GlobalWG.Add(1)
					go c.Step("update current user", func(c *Context) {
						defer c.GlobalWG.Done()
						if _, err := c.Goon.Put(user); err != nil {
							panic(err)
						}
					})
				}

				c.FillRenderParams(map[string]interface{}{
					"User":    user,
					"UserKey": c.Goon.Key(user),
				})
			})
		}
	})
}

// Sets one of the render parameters.
// This should be called rather than directly manipulating the map.
func (c *Context) SetRenderParam(key string, value interface{}) {
	c.includeLock.Lock()
	defer c.includeLock.Unlock()

	c.includes[key] = value
}

// Fills the render parameters from a map.
func (c *Context) FillRenderParams(filler Includes) {
	c.includeLock.Lock()
	defer c.includeLock.Unlock()

	for key, value := range filler {
		c.includes[key] = value
	}
}

// Gets one of the render parameters.
// This should be called rather than directly manipulating the map.
func (c *Context) GetRenderParam(key string) (value interface{}, ok bool) {
	c.includeLock.RLock()
	defer c.includeLock.RUnlock()

	value, ok = c.includes[key]
	return
}

// Safely executes a template.
func serveTemplate(c *Context, templateName string, includes Includes) (success bool) {
	c.Step("render template", func(c *Context) {
		buf := new(bytes.Buffer) // We create a buffer so that the template output doesn't go directly to the response. This lets us avoid writing anything during an error.
		c.Step("execute", func(c *Context) {
			if err := templates.ExecuteTemplate(buf, templateName, includes); err != nil {
				serveError(c, err, c.Response)
				success = false
				return
			}
		})

		c.Step("copy output", func(c *Context) {
			_, err := io.Copy(c.Response, buf) // Copy the template response over to the actual HTTP response.
			if err != nil {
				serveError(c, err, c.Response)
				success = false
				return
			}
		})

		success = true
	})
	return
}

func serveError(c appengine.Context, err error, response http.ResponseWriter) {
	ID := appengine.RequestID(c)
	c.Criticalf("%s\n%s", err.Error(), string(debug.Stack()))
	if appengine.IsDevAppServer() {
		http.Error(response, fmt.Sprintf("%s\n%s", err, string(debug.Stack())), http.StatusInternalServerError)
	} else {
		http.Error(response, fmt.Sprintf("An internal error has occured. It has been logged.\nRequest ID: %s (keep this if you want to email support)", ID), http.StatusInternalServerError)
	}
}

func getAppEmail(c *Context, user string) string {
	appIDUnsplit := appengine.AppID(c)
	split := strings.SplitN(appIDUnsplit, ":", 1)

	var appID string
	if len(split) > 1 {
		appID = split[len(split)-1]
	} else {
		appID = split[0]
	}

	return user + "@" + appID + ".appspotmail.com"
}

type Includes map[string]interface{}

var (
	BootstrapCss     string
	BootstrapJs      string
	Jquery           string
	baseRenderParams Includes
)

func init() {
	const (
		angular_ver   = "1.0.5"
		bootstrap_ver = "3.0.2"
		jquery_ver    = "2.0.3"
	)

	if appengine.IsDevAppServer() {
		//BootstrapCss = fmt.Sprintf("/static/css/cosmo-%s.css", bootstrap_ver)
		BootstrapCss = fmt.Sprintf("/static/css/bootstrap-%s.css", bootstrap_ver) // Default Bootstrap
		BootstrapJs = fmt.Sprintf("/static/js/bootstrap-%s.js", bootstrap_ver)
		Jquery = fmt.Sprintf("/static/js/jquery-%s.js", jquery_ver)
	} else {
		//BootstrapCss = fmt.Sprintf("//netdna.bootstrapcdn.com/bootswatch/%s/cosmo/bootstrap.min.css", bootstrap_ver)
		BootstrapCss = fmt.Sprintf("//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.min.css", bootstrap_ver) // Default Bootstrap
		BootstrapJs = fmt.Sprintf("//netdna.bootstrapcdn.com/bootstrap/%s/js/bootstrap.min.js", bootstrap_ver)
		Jquery = fmt.Sprintf("//ajax.googleapis.com/ajax/libs/jquery/%s/jquery.min.js", jquery_ver)
	}

	baseRenderParams = Includes{
		"BootstrapCss": BootstrapCss,
		"BootstrapJs":  BootstrapJs,
		"Jquery":       Jquery,
		"AppVersion":   VERSION,
	}
}
