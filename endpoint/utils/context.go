// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package utils

import (
	"appengine/datastore"
	"net/http"
	"strings"

	"github.com/mjibson/goon"
	"github.com/nightexcessive/go-endpoints/endpoints"

	"github.com/HL2-Ghosting-Team/website/models"
)

// Creates a new Context based on the given Request.
func NewContext(r *http.Request) *Context {
	context := &Context{
		Context: endpoints.NewContext(r),
	}
	context.Init()

	return context
}

type Context struct {
	endpoints.Context

	Goon *goon.Goon
}

// Initializes the Context. This should be called before using the context.
// Note: NewContext will automatically call this for you.
func (c *Context) Init() {
	c.Goon = goon.FromContext(c)
}

func (c *Context) RunInTransaction(f func(*Context) error, opts *datastore.TransactionOptions) error {
	return c.Goon.RunInTransaction(func(g *goon.Goon) error {
		return f(&Context{
			Context: c.Context,

			Goon: g,
		})
	}, opts)
}

// Determines whether a given error from a call to the endpoints GetUser is due
// to the user not being logged in.
func isNotLoggedInError(err error) bool {
	return strings.HasSuffix(err.Error(), "(user: NOT_ALLOWED)")
}

func (c *Context) GetUser() (*models.User, error) {
	aeUser, err := c.CurrentOAuthUser(UserScope)
	if err != nil {
		if isNotLoggedInError(err) {
			return nil, nil
		}
		return nil, err
	}
	if aeUser == nil {
		return nil, nil
	}

	user := &models.User{
		ID: aeUser.ID,
	}
	if err := c.Goon.Get(user); err == datastore.ErrNoSuchEntity {
		user.Email, user.Admin = aeUser.Email, aeUser.Admin

		atSign := strings.LastIndex(user.Email, "@")
		user.Nickname = user.Email[:atSign]

		if _, err := c.Goon.Put(user); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return user, nil
}
