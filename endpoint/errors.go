// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"net/http"
)

// An error returned whenever a user is required to be logged in, but is not
// logged in.
type ErrLoginRequired struct {
}

func (*ErrLoginRequired) Error() string {
	return "Login Required"
}

func (*ErrLoginRequired) HTTPStatus() int {
	return http.StatusUnauthorized
}

// An error returned whenever a cursor is given, but is not in a valid format.
type ErrCursorInvalid struct {
}

func (*ErrCursorInvalid) Error() string {
	return "The given cursor is invalid."
}

func (*ErrCursorInvalid) HTTPStatus() int {
	return http.StatusBadRequest
}
