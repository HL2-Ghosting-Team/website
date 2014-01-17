// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"testing"
)

func eqHelper(t *testing.T, expectedEq bool, args ...interface{}) {
	if expectedEq {
		if !eq(args...) {
			t.Errorf("Expected %v to be equal, got false", args)
		}
	} else {
		if eq(args...) {
			t.Errorf("Expected %v to not be equal, got true", args)
		}
	}
}

func TestEq(t *testing.T) {
	t.Parallel()

	eqHelper(t, true, 1, 1)
	eqHelper(t, true, 1, 1, 1)
	eqHelper(t, false, 1, 1, 3)
	eqHelper(t, false, 1, 2)
	eqHelper(t, false, 1, 2, 3)
}
