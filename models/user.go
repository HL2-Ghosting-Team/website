// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package models

import (
	"github.com/ftrvxmtrx/gravatar"
)

// Creates a fake user to use in substitute of a user that's been deleted.
func CreateDeletedUser() *User {
	return &User{
		Email:    "deleted@ghosting.nightexcessive.us",
		Nickname: "[USER DELETED]",
		Admin:    false,
	}
}

type User struct {
	ID string `datastore:"-" goon:"id" json:"-"`

	Email    string `json:"-"`
	Nickname string `json:"nickname"`

	Admin bool `json:"admin"`
}

func (u *User) Avatar() string {
	return gravatar.GetAvatarURL("https", gravatar.EmailHash(u.Email), gravatar.DefaultIdentIcon, gravatar.RatingPG).String()
}
