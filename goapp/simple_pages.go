// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine"
	"appengine/user"
	"net/http"
	"net/url"
	"time"
)

func NotFound(c *Context) {
	c.Response.WriteHeader(404)
	c.Render()
}

func Index(c *Context) {
	c.Render()
}

func getLoginRedirect(c *Context) string {
	if requestedRedirect := c.Req.URL.Query().Get("redirect"); len(requestedRedirect) > 0 {
		if requestedRedirectURL, err := url.Parse(requestedRedirect); err == nil && len(requestedRedirectURL.Scheme) == 0 && len(requestedRedirectURL.Host) == 0 {
			return requestedRedirectURL.String()
		}
	}

	if referer := c.Req.Referer(); len(referer) > 0 {
		return referer
	}

	index, err := routerUrl("index")
	if err != nil {
		panic(err)
	}

	return index
}

func LoginGoogle(c *Context) {
	redirectTo := getLoginRedirect(c)

	loginURL, err := user.LoginURL(c, redirectTo)
	if err != nil {
		panic(err)
	}

	http.Redirect(c.Response, c.Req, loginURL, http.StatusTemporaryRedirect)
}

func LogoutGoogle(c *Context) {
	redirectTo := getLoginRedirect(c)

	if appengine.IsDevAppServer() {
		logoutURL, err := user.LogoutURL(c, redirectTo)
		if err != nil {
			serveError(c, err, c.Response)
			return
		}

		http.Redirect(c.Response, c.Req, logoutURL, http.StatusTemporaryRedirect)
	} else {
		// This hackiness is because we don't want to log the user out of their Google account.
		http.SetCookie(c.Response, &http.Cookie{
			Name:    "ACSID",
			Value:   "",
			Expires: time.Time{},
		})
		http.SetCookie(c.Response, &http.Cookie{
			Name:    "SACSID",
			Value:   "",
			Expires: time.Time{},
		})

		http.Redirect(c.Response, c.Req, redirectTo, http.StatusTemporaryRedirect)
	}
}
