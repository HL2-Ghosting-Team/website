// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package models

// Creates a fake user to use in substitute of a user that's been deleted.
func CreateDeletedUser() *User {
	return &User{
		Email:    "deleted@ghosting.nightexcessive.us",
		Nickname: "[USER DELETED]",
		Admin:    false,
	}
}

type User struct {
	ID string `datastore:"-" goon:"id"`

	Email    string
	Nickname string

	Admin bool
}
