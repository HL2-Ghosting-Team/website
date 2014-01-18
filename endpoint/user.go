// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"appengine/datastore"
	"net/http"

	"github.com/HL2-Ghosting-Team/website/endpoint/utils"
	"github.com/HL2-Ghosting-Team/website/models"
)

func init() {
	if getUser := ghostingService.MethodByName("GetUser").Info(); getUser != nil {
		getUser.Name, getUser.HttpMethod, getUser.Path, getUser.Desc, getUser.Scopes =
			"users.get", "GET", "users/get/{game}", "Retrieve information about a user", []string{utils.UserScope}
	}
}

// An error returned when a given user ID is invalid. This is usually due to the
// user ID being in an incorrect format.
type ErrInvalidUserID struct {
	ID string
}

func (e *ErrInvalidUserID) Error() string {
	return "Invalid user ID: " + e.ID
}

func (*ErrInvalidUserID) HTTPStatus() int {
	return http.StatusBadRequest
}

// An error returned when a given user ID does not exist in the datastore.
type ErrNoSuchUser struct {
}

func (e *ErrNoSuchUser) Error() string {
	return "No such user exists"
}

func (*ErrNoSuchUser) HTTPStatus() int {
	return http.StatusNotFound
}

type ErrUserLoginRequired struct {
	ErrLoginRequired
}

func (*ErrUserLoginRequired) Error() string {
	return "In order to use the \"current\" keyword, you must be logged in."
}

type GetUserRequest struct {
	ID string `json:"user" endpoints:"d=current" endpoints_desc:"The ID of a user or 'current' for the current user."`
}

type GetUserResponse struct {
	ID *datastore.Key `json:"id" endpoints_desc:"The ID of the returned user"`

	Nickname string `json:"nickname"`
	Admin    bool   `json:"admin"`

	Avatar string `json:"avatar"`
}

func (*GhostingService) GetUser(r *http.Request, req *GetUserRequest, res *GetUserResponse) error {
	c := utils.NewContext(r)

	if len(req.ID) <= 0 {
		return &ErrInvalidUserID{ID: req.ID}
	}

	var userKey *datastore.Key
	if req.ID == "current" {
		currentUser, err := c.GetUser()
		if err != nil {
			panic(err)
		} else if currentUser == nil {
			return new(ErrUserLoginRequired)
		}
		c.Infof("Current user: %#v", currentUser)
		res.ID = c.Goon.Key(&models.User{ID: currentUser.ID})
	} else {
		var err error
		userKey, err = datastore.DecodeKey(req.ID)
		if err != nil {
			c.Infof("Unable to decode run key: %s", err)
			return &ErrInvalidRunID{ID: req.ID}
		}
		res.ID = userKey
	}

	user := &models.User{
		ID: res.ID.StringID(),
	}
	if err := c.Goon.Get(user); err == datastore.ErrNoSuchEntity {
		res.ID = nil
		return new(ErrNoSuchUser)
	} else if err != nil {
		panic(err)
	}

	res.Admin, res.Nickname, res.Avatar = user.Admin, user.Nickname, user.Avatar()

	return nil
}
