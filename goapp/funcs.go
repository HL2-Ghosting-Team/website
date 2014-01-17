// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package goapp

import (
	"appengine/datastore"
	"fmt"
	"html/template"
	"reflect"

	"github.com/ftrvxmtrx/gravatar"

	"github.com/HL2-Ghosting-Team/website/models"
	"github.com/HL2-Ghosting-Team/website/runs"
)

// eq reports whether the first argument is equal to
// all of the remaining arguments.
func eq(args ...interface{}) bool {
	if len(args) == 0 {
		return false
	}
	x := args[0]
	switch x := x.(type) {
	case string, int, int64, byte, float32, float64:
		for _, y := range args[1:] {
			if x != y {
				return false
			}
		}
		return true
	}

	for _, y := range args[1:] {
		if !reflect.DeepEqual(x, y) {
			return false
		}
	}
	return true
}

func set(renderArgs Includes, key string, value interface{}) template.HTML {
	renderArgs[key] = value
	return template.HTML("")
}

func routerUrl(name string, pairsRaw ...interface{}) (string, error) {
	if route, ok := routes[name]; ok {
		pairs := make([]string, len(pairsRaw))
		for i, value := range pairsRaw {
			pairs[i] = fmt.Sprintf("%v", value)
		}

		return route.URLWith(pairs), nil
	}

	return "", fmt.Errorf("not a known route: %s", name)
}

func avatarUrl(user *models.User, size int) string {
	return gravatar.GetAvatarURL("https", gravatar.EmailHash(user.Email), gravatar.DefaultIdentIcon, gravatar.RatingPG, size).String()
}

func getDatastoreKey(c *Context, model interface{}) *datastore.Key {
	return c.Goon.Key(model)
}

func prettyGameName(gameID int) string {
	if name, ok := runs.PrettyGameNames[byte(gameID)]; ok {
		return name
	}

	return "Unknown"
}

var funcs = template.FuncMap{
	"avatarUrl":       avatarUrl,
	"eq":              eq,
	"set":             set,
	"url":             routerUrl,
	"getDatastoreKey": getDatastoreKey,
	"prettyGameName":  prettyGameName,
}
